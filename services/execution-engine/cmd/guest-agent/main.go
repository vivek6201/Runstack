package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"code-runner/shared/dto"
)

const (
	vsockPort       = 5000
	workspaceDir    = "/workspace"
	maxOutputBytes  = 256 * 1024
	compileTimeout  = 15 * time.Second
)

func main() {
	log.Println("[Guest-Agent] Starting as PID 1 (/init)...")

	// 1. Mount pseudo-filesystems if running as PID 1
	if os.Getpid() == 1 {
		mountFilesystems()
	}

	// 2. Ensure clean workspace on tmpfs
	_ = os.MkdirAll(workspaceDir, 0777)

	// 3. Listen on vsock (or fallback to TCP port 5000 in test mode)
	listener, err := listenVsock(vsockPort)
	if err != nil {
		log.Fatalf("[Guest-Agent] Failed to bind listener on port %d: %v", vsockPort, err)
	}
	defer listener.Close()

	log.Printf("[Guest-Agent] Ready and listening for execution payloads on port %d...", vsockPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[Guest-Agent] Accept error: %v", err)
			continue
		}

		handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read job request from host
	var job dto.Job
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&job); err != nil {
		log.Printf("[Guest-Agent] Failed to decode job JSON: %v", err)
		return
	}

	log.Printf("[Guest-Agent] Received job %s (lang: %s, test_cases: %d)", job.ID, job.Language, len(job.TestCases))

	// Execute job using Compile-Once, Run-N-Times
	result := executeJob(&job)

	// Stream execution result back to host
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(result); err != nil {
		log.Printf("[Guest-Agent] Failed to send execution result: %v", err)
		return
	}

	log.Printf("[Guest-Agent] Completed job %s in %d ms", job.ID, result.TimeTakenMs)
}

func executeJob(job *dto.Job) *dto.ExecutionResult {
	// Clean workspace between runs (clean contents without removing mount point)
	entries, err := os.ReadDir(workspaceDir)
	if err == nil {
		for _, e := range entries {
			_ = os.RemoveAll(filepath.Join(workspaceDir, e.Name()))
		}
	} else {
		_ = os.MkdirAll(workspaceDir, 0777)
	}

	cfg, err := getLanguageConfig(job.Language, workspaceDir)
	if err != nil {
		return &dto.ExecutionResult{
			ExitCode: 1,
			Stderr:   fmt.Sprintf("Unsupported language in microVM: %s", job.Language),
		}
	}

	// 1. Write source file
	srcPath := filepath.Join(workspaceDir, cfg.srcFile)
	if err := os.WriteFile(srcPath, []byte(job.Code), 0644); err != nil {
		return &dto.ExecutionResult{
			ExitCode: 1,
			Stderr:   fmt.Sprintf("Failed to write source file: %v", err),
		}
	}

	result := &dto.ExecutionResult{
		TestResults: make([]dto.TestResult, 0),
	}

	// 2. COMPILE PHASE (once for compiled languages)
	if cfg.compileCmd != "" {
		compileCtx, cancel := context.WithTimeout(context.Background(), compileTimeout)
		defer cancel()

		cmd := exec.CommandContext(compileCtx, cfg.compileCmd, cfg.compileArgs...)
		cmd.Dir = workspaceDir

		var stderrBuf, stdoutBuf strings.Builder
		cmd.Stderr = &stderrBuf
		cmd.Stdout = &stdoutBuf

		start := time.Now()
		err := cmd.Run()
		result.CompilationTimeMs = time.Since(start).Milliseconds()

		if err != nil {
			errOutput := stderrBuf.String()
			if errOutput == "" {
				errOutput = stdoutBuf.String()
			}
			result.ExitCode = 1
			result.Stderr = capOutput("Compilation Error:\n" + errOutput)
			return result // Skip test cases
		}
	}

	// 3. EXECUTE TEST CASES PHASE
	timeout := time.Duration(job.TimeLimitMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 3000 * time.Millisecond
	}

	if len(job.TestCases) == 0 {
		singleRes := runSingle(workspaceDir, cfg.runCmd, cfg.runArgs, "", timeout)
		singleRes.CompilationTimeMs = result.CompilationTimeMs
		return singleRes
	}

	var totalExecutionTime int64
	for _, tc := range job.TestCases {
		res := runSingle(workspaceDir, cfg.runCmd, cfg.runArgs, tc.Input, timeout)
		totalExecutionTime += res.TimeTakenMs

		trimmedActual := strings.TrimRight(res.Stdout, "\r\n")
		trimmedExpected := strings.TrimRight(tc.ExpectedOutput, "\r\n")
		passed := (trimmedActual == trimmedExpected) && (res.ExitCode == 0)

		result.TestResults = append(result.TestResults, dto.TestResult{
			TestCaseID:     tc.ID,
			Passed:         passed,
			ActualOutput:   res.Stdout,
			ExpectedOutput: tc.ExpectedOutput,
			TimeTakenMs:    res.TimeTakenMs,
		})

		result.Stdout = res.Stdout
		result.Stderr = res.Stderr
		result.ExitCode = res.ExitCode
	}

	result.TimeTakenMs = totalExecutionTime
	result.MemoryUsedKB = readCgroupPeakMemory()
	return result
}

func runSingle(dir, cmdName string, args []string, stdinInput string, timeout time.Duration) *dto.ExecutionResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, cmdName, args...)
	cmd.Dir = dir

	if stdinInput != "" {
		cmd.Stdin = strings.NewReader(stdinInput)
	}

	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start).Milliseconds()

	stdoutStr := capOutput(stdoutBuf.String())
	stderrStr := capOutput(stderrBuf.String())

	exitCode := 0
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &dto.ExecutionResult{
				Stdout:      stdoutStr,
				Stderr:      "Time Limit Exceeded\n" + stderrStr,
				ExitCode:    124,
				TimeTakenMs: duration,
			}
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return &dto.ExecutionResult{
		Stdout:      stdoutStr,
		Stderr:      stderrStr,
		ExitCode:    exitCode,
		TimeTakenMs: duration,
	}
}

type vmConfig struct {
	srcFile     string
	compileCmd  string
	compileArgs []string
	runCmd      string
	runArgs     []string
}

func getLanguageConfig(language, dir string) (*vmConfig, error) {
	binaryPath := filepath.Join(dir, "solution")

	switch strings.ToLower(language) {
	case "c":
		return &vmConfig{
			srcFile:     "solution.c",
			compileCmd:  "gcc",
			compileArgs: []string{"-O2", "-o", binaryPath, filepath.Join(dir, "solution.c")},
			runCmd:      binaryPath,
		}, nil

	case "cpp", "c++":
		return &vmConfig{
			srcFile:     "solution.cpp",
			compileCmd:  "g++",
			compileArgs: []string{"-O2", "-std=c++20", "-o", binaryPath, filepath.Join(dir, "solution.cpp")},
			runCmd:      binaryPath,
		}, nil

	case "go", "golang":
		return &vmConfig{
			srcFile:     "solution.go",
			compileCmd:  "go",
			compileArgs: []string{"build", "-o", binaryPath, filepath.Join(dir, "solution.go")},
			runCmd:      binaryPath,
		}, nil

	case "java":
		return &vmConfig{
			srcFile:     "Solution.java",
			compileCmd:  "javac",
			compileArgs: []string{filepath.Join(dir, "Solution.java")},
			runCmd:      "java",
			runArgs:     []string{"-cp", dir, "Solution"},
		}, nil

	case "python", "py", "python3":
		return &vmConfig{
			srcFile: "solution.py",
			runCmd:  "python3",
			runArgs: []string{filepath.Join(dir, "solution.py")},
		}, nil

	case "javascript", "js", "node":
		return &vmConfig{
			srcFile: "solution.js",
			runCmd:  "node",
			runArgs: []string{filepath.Join(dir, "solution.js")},
		}, nil

	case "typescript", "ts":
		return &vmConfig{
			srcFile: "solution.ts",
			runCmd:  "bun",
			runArgs: []string{"run", filepath.Join(dir, "solution.ts")},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

func capOutput(s string) string {
	if len(s) > maxOutputBytes {
		return s[:maxOutputBytes] + "\n[Output truncated at 256KB limit]"
	}
	return s
}

// readCgroupPeakMemory reads peak RSS from cgroups v2 inside the VM
func readCgroupPeakMemory() int64 {
	// Check cgroups v2 memory.peak
	data, err := os.ReadFile("/sys/fs/cgroup/memory.peak")
	if err == nil {
		var bytes int64
		if _, err := fmt.Sscanf(string(data), "%d", &bytes); err == nil && bytes > 0 {
			return bytes / 1024
		}
	}
	return 16384 // Default 16MB estimate if cgroups not mounted
}
