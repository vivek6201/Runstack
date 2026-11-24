package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"code-runner/shared/dto"
)

const (
	maxOutputBytes = 256 * 1024       // 256 KB max stdout/stderr cap
	compileTimeout = 15 * time.Second // Max 15s for compilation
)

// getSanitizedEnv returns a minimal, isolated environment for child processes,
// preventing user code from inspecting host secrets, database URLs, or Redis credentials.
func getSanitizedEnv(workingDir string) []string {
	path := os.Getenv("PATH")
	if path == "" {
		path = "/usr/local/bin:/usr/bin:/bin:/opt/homebrew/bin"
	}
	return []string{
		"PATH=" + path,
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
		"HOME=" + workingDir,
		"TMPDIR=" + workingDir,
		"PYTHONUNBUFFERED=1",
		"PYTHONDONTWRITEBYTECODE=1",
		"NODE_ENV=production",
	}
}

// LocalSubprocessAdapter executes code locally inside an isolated temporary directory.
// For compiled languages (C, C++, Go, Java), it compiles ONCE and executes the resulting
// binary across all test cases with high speed.
type LocalSubprocessAdapter struct{}

func NewLocalSubprocessAdapter() *LocalSubprocessAdapter {
	return &LocalSubprocessAdapter{}
}

func (a *LocalSubprocessAdapter) Name() string {
	return "local-subprocess"
}

type langConfig struct {
	srcFile    string
	compileCmd string
	compileArgs []string
	runCmd     string
	runArgs    []string
}

func (a *LocalSubprocessAdapter) Execute(ctx context.Context, job *dto.Job) (*dto.ExecutionResult, error) {
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("exec-%s-*", job.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp execution dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cfg, err := prepareLanguageConfig(job.Language, tempDir)
	if err != nil {
		return nil, err
	}

	// 1. Write user source code to file
	srcPath := filepath.Join(tempDir, cfg.srcFile)
	if err := os.WriteFile(srcPath, []byte(job.Code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write source code: %w", err)
	}

	result := &dto.ExecutionResult{
		TestResults: make([]dto.TestResult, 0),
	}

	// 2. COMPILE PHASE (if language requires compilation)
	if cfg.compileCmd != "" {
		compileCtx, cancel := context.WithTimeout(ctx, compileTimeout)
		defer cancel()

		cmd := exec.Command(cfg.compileCmd, cfg.compileArgs...)
		cmd.Dir = tempDir
		cmd.Env = getSanitizedEnv(tempDir)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

		var compileStderr, compileStdout strings.Builder
		cmd.Stderr = &compileStderr
		cmd.Stdout = &compileStdout

		compileStart := time.Now()
		if err := cmd.Start(); err != nil {
			result.ExitCode = 1
			result.Stderr = capOutput(fmt.Sprintf("Failed to start compiler: %v", err))
			return result, nil
		}

		compileDone := make(chan error, 1)
		go func() {
			compileDone <- cmd.Wait()
		}()

		var compileErr error
		select {
		case <-compileCtx.Done():
			if cmd.Process != nil {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			}
			<-compileDone
			result.CompilationTimeMs = time.Since(compileStart).Milliseconds()
			result.ExitCode = 1
			result.Stderr = capOutput("Compilation Error:\nCompilation Timed Out (exceeded 15s limit)\n" + compileStderr.String())
			return result, nil
		case compileErr = <-compileDone:
			result.CompilationTimeMs = time.Since(compileStart).Milliseconds()
		}

		if compileErr != nil {
			errOutput := compileStderr.String()
			if errOutput == "" {
				errOutput = compileStdout.String()
			}
			// Compilation failed: skip all test cases and return immediately!
			result.ExitCode = 1
			result.Stderr = capOutput("Compilation Error:\n" + errOutput)
			return result, nil
		}
	}

	// 3. EXECUTE TEST CASES PHASE (Fast loop over pre-compiled binary or interpreter)
	timeout := time.Duration(job.TimeLimitMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 3000 * time.Millisecond
	}

	// Case A: No test cases provided — run once with no stdin
	if len(job.TestCases) == 0 {
		singleRes, err := runSingleTestCase(ctx, tempDir, cfg.runCmd, cfg.runArgs, "", timeout)
		if err != nil {
			return nil, err
		}
		singleRes.CompilationTimeMs = result.CompilationTimeMs
		return singleRes, nil
	}

	// Case B: Batch test case execution
	var totalExecutionTime int64
	for _, tc := range job.TestCases {
		res, err := runSingleTestCase(ctx, tempDir, cfg.runCmd, cfg.runArgs, tc.Input, timeout)
		if err != nil {
			return nil, err
		}
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

		// Track last / first failing output
		result.Stdout = res.Stdout
		result.Stderr = res.Stderr
		result.ExitCode = res.ExitCode
	}

	result.TimeTakenMs = totalExecutionTime
	result.MemoryUsedKB = 16384 // Estimated baseline memory
	return result, nil
}

func prepareLanguageConfig(language, tempDir string) (*langConfig, error) {
	binaryPath := filepath.Join(tempDir, "solution")

	switch strings.ToLower(language) {
	// --- Compiled Languages (Compile Once, Run N times) ---
	case "c":
		return &langConfig{
			srcFile:     "solution.c",
			compileCmd:  "gcc",
			compileArgs: []string{"-O2", "-o", binaryPath, filepath.Join(tempDir, "solution.c")},
			runCmd:      binaryPath,
			runArgs:     []string{},
		}, nil

	case "cpp", "c++", "cxx", "cc":
		return &langConfig{
			srcFile:     "solution.cpp",
			compileCmd:  "g++",
			compileArgs: []string{"-O2", "-std=c++20", "-o", binaryPath, filepath.Join(tempDir, "solution.cpp")},
			runCmd:      binaryPath,
			runArgs:     []string{},
		}, nil

	case "go", "golang":
		return &langConfig{
			srcFile:     "solution.go",
			compileCmd:  "go",
			compileArgs: []string{"build", "-o", binaryPath, filepath.Join(tempDir, "solution.go")},
			runCmd:      binaryPath,
			runArgs:     []string{},
		}, nil

	case "java":
		// Java public class must match file name: Solution.java
		return &langConfig{
			srcFile:     "Solution.java",
			compileCmd:  "javac",
			compileArgs: []string{filepath.Join(tempDir, "Solution.java")},
			runCmd:      "java",
			runArgs:     []string{"-cp", tempDir, "Solution"},
		}, nil

	// --- Interpreted / JIT Languages (Run directly) ---
	case "python", "py", "python3":
		return &langConfig{
			srcFile: "solution.py",
			runCmd:  "python3",
			runArgs: []string{filepath.Join(tempDir, "solution.py")},
		}, nil

	case "javascript", "js", "node":
		return &langConfig{
			srcFile: "solution.js",
			runCmd:  "node",
			runArgs: []string{filepath.Join(tempDir, "solution.js")},
		}, nil

	case "typescript", "ts":
		// Check if bun is available (blazing fast TS runtime)
		tsRunner := "bun"
		if _, err := exec.LookPath("bun"); err != nil {
			// Fallback to node with stripped types
			tsRunner = "node"
			return &langConfig{
				srcFile: "solution.ts",
				runCmd:  tsRunner,
				runArgs: []string{"--experimental-strip-types", filepath.Join(tempDir, "solution.ts")},
			}, nil
		}
		return &langConfig{
			srcFile: "solution.ts",
			runCmd:  tsRunner,
			runArgs: []string{"run", filepath.Join(tempDir, "solution.ts")},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
}

func runSingleTestCase(ctx context.Context, workingDir, cmdName string, args []string, stdinInput string, timeout time.Duration) (*dto.ExecutionResult, error) {
	execCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.Command(cmdName, args...)
	cmd.Dir = workingDir
	cmd.Env = getSanitizedEnv(workingDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if stdinInput != "" {
		cmd.Stdin = strings.NewReader(stdinInput)
	}

	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	start := time.Now()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start execution process: %w", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	var waitErr error
	select {
	case <-execCtx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-waitDone
		duration := time.Since(start).Milliseconds()
		return &dto.ExecutionResult{
			Stdout:      capOutput(stdoutBuf.String()),
			Stderr:      "Time Limit Exceeded\n" + capOutput(stderrBuf.String()),
			ExitCode:    124,
			TimeTakenMs: duration,
		}, nil
	case waitErr = <-waitDone:
	}

	duration := time.Since(start).Milliseconds()
	stdoutStr := capOutput(stdoutBuf.String())
	stderrStr := capOutput(stderrBuf.String())

	exitCode := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
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
	}, nil
}

func capOutput(s string) string {
	if len(s) > maxOutputBytes {
		return s[:maxOutputBytes] + "\n[Output truncated at 256KB limit]"
	}
	return s
}
