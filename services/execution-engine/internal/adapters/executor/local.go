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
type LocalSubprocessAdapter struct{}

func NewLocalSubprocessAdapter() *LocalSubprocessAdapter {
	return &LocalSubprocessAdapter{}
}

func (a *LocalSubprocessAdapter) Name() string {
	return "local-subprocess"
}

type langConfig struct {
	srcFile     string
	compileCmd  string
	compileArgs []string
	runCmd      string
	runArgs     []string
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

	srcPath := filepath.Join(tempDir, cfg.srcFile)
	if err := os.WriteFile(srcPath, []byte(job.Code), 0600); err != nil {
		return nil, fmt.Errorf("failed to write source code: %w", err)
	}

	compileTimeMs, compileFailure := compileSource(ctx, tempDir, cfg)
	if compileFailure != nil {
		return compileFailure, nil
	}

	return executeTestCases(ctx, tempDir, cfg, job, compileTimeMs)
}

func compileSource(ctx context.Context, tempDir string, cfg *langConfig) (int64, *dto.ExecutionResult) {
	if cfg.compileCmd == "" {
		return 0, nil
	}

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
		return 0, &dto.ExecutionResult{
			TestResults: make([]dto.TestResult, 0),
			ExitCode:    1,
			Stderr:      capOutput(fmt.Sprintf("Failed to start compiler: %v", err)),
		}
	}

	compileDone := make(chan error, 1)
	go func() {
		compileDone <- cmd.Wait()
	}()

	select {
	case <-compileCtx.Done():
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-compileDone
		return 0, &dto.ExecutionResult{
			TestResults:       make([]dto.TestResult, 0),
			CompilationTimeMs: time.Since(compileStart).Milliseconds(),
			ExitCode:          1,
			Stderr:            capOutput("Compilation Error:\nCompilation Timed Out (exceeded 15s limit)\n" + compileStderr.String()),
		}
	case compileErr := <-compileDone:
		elapsed := time.Since(compileStart).Milliseconds()
		if compileErr == nil {
			return elapsed, nil
		}
		errOutput := compileStderr.String()
		if errOutput == "" {
			errOutput = compileStdout.String()
		}
		return elapsed, &dto.ExecutionResult{
			TestResults:       make([]dto.TestResult, 0),
			CompilationTimeMs: elapsed,
			ExitCode:          1,
			Stderr:            capOutput("Compilation Error:\n" + errOutput),
		}
	}
}

func executeTestCases(ctx context.Context, tempDir string, cfg *langConfig, job *dto.Job, compileTimeMs int64) (*dto.ExecutionResult, error) {
	timeout := time.Duration(job.TimeLimitMs) * time.Millisecond
	if timeout <= 0 {
		timeout = 3000 * time.Millisecond
	}

	if len(job.TestCases) == 0 {
		singleRes, err := runSingleTestCase(ctx, tempDir, cfg.runCmd, cfg.runArgs, "", timeout)
		if err != nil {
			return nil, err
		}
		singleRes.CompilationTimeMs = compileTimeMs
		return singleRes, nil
	}

	result := &dto.ExecutionResult{
		CompilationTimeMs: compileTimeMs,
		TestResults:       make([]dto.TestResult, 0, len(job.TestCases)),
		MemoryUsedKB:      16384,
	}

	for _, tc := range job.TestCases {
		res, err := runSingleTestCase(ctx, tempDir, cfg.runCmd, cfg.runArgs, tc.Input, timeout)
		if err != nil {
			return nil, err
		}
		result.TimeTakenMs += res.TimeTakenMs

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

	return result, nil
}

func prepareLanguageConfig(language, tempDir string) (*langConfig, error) {
	binaryPath := filepath.Join(tempDir, "solution")

	switch strings.ToLower(language) {
	case "c":
		return &langConfig{
			srcFile:     "solution.c",
			compileCmd:  "gcc",
			compileArgs: []string{"-O2", "-o", binaryPath, filepath.Join(tempDir, "solution.c")},
			runCmd:      binaryPath,
		}, nil

	case "cpp", "c++", "cxx", "cc":
		return &langConfig{
			srcFile:     "solution.cpp",
			compileCmd:  "g++",
			compileArgs: []string{"-O2", "-std=c++20", "-o", binaryPath, filepath.Join(tempDir, "solution.cpp")},
			runCmd:      binaryPath,
		}, nil

	case "go", "golang":
		return &langConfig{
			srcFile:     "solution.go",
			compileCmd:  "go",
			compileArgs: []string{"build", "-o", binaryPath, filepath.Join(tempDir, "solution.go")},
			runCmd:      binaryPath,
		}, nil

	case "java":
		return &langConfig{
			srcFile:     "Solution.java",
			compileCmd:  "javac",
			compileArgs: []string{filepath.Join(tempDir, "Solution.java")},
			runCmd:      "java",
			runArgs:     []string{"-cp", tempDir, "Solution"},
		}, nil

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
		if _, err := exec.LookPath("bun"); err == nil {
			return &langConfig{
				srcFile: "solution.ts",
				runCmd:  "bun",
				runArgs: []string{"run", filepath.Join(tempDir, "solution.ts")},
			}, nil
		}
		return &langConfig{
			srcFile: "solution.ts",
			runCmd:  "node",
			runArgs: []string{"--experimental-strip-types", filepath.Join(tempDir, "solution.ts")},
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
		return &dto.ExecutionResult{
			Stdout:      capOutput(stdoutBuf.String()),
			Stderr:      "Time Limit Exceeded\n" + capOutput(stderrBuf.String()),
			ExitCode:    124,
			TimeTakenMs: time.Since(start).Milliseconds(),
		}, nil
	case waitErr = <-waitDone:
	}

	return &dto.ExecutionResult{
		Stdout:      capOutput(stdoutBuf.String()),
		Stderr:      capOutput(stderrBuf.String()),
		ExitCode:    extractExitCode(waitErr),
		TimeTakenMs: time.Since(start).Milliseconds(),
	}, nil
}

func extractExitCode(waitErr error) int {
	if waitErr == nil {
		return 0
	}
	if exitErr, ok := waitErr.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return 1
}

func capOutput(s string) string {
	if len(s) > maxOutputBytes {
		return s[:maxOutputBytes] + "\n[Output truncated at 256KB limit]"
	}
	return s
}
