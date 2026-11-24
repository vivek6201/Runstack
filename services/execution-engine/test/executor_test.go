package test

import (
	"context"
	"strings"
	"testing"
	"time"

	"code-runner/execution-engine/internal/adapters/executor"
	"code-runner/shared/dto"
)

func TestLocalExecutor_Python(t *testing.T) {
	adapter := executor.NewLocalSubprocessAdapter()

	job := &dto.Job{
		ID:          "test-py",
		Language:    "python",
		Code:        "import sys\nprint(int(sys.stdin.read().strip()) * 3)",
		TimeLimitMs: 3000,
		TestCases: []dto.TestCase{
			{ID: "1", Input: "5\n", ExpectedOutput: "15\n"},
			{ID: "2", Input: "12\n", ExpectedOutput: "36\n"},
		},
	}

	res, err := adapter.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(res.TestResults) != 2 || !res.TestResults[0].Passed || !res.TestResults[1].Passed {
		t.Fatalf("expected all test cases to pass: %+v", res.TestResults)
	}
}

func TestLocalExecutor_JavaScript(t *testing.T) {
	adapter := executor.NewLocalSubprocessAdapter()

	job := &dto.Job{
		ID:       "test-js",
		Language: "javascript",
		Code: `const fs = require('fs');
const num = parseInt(fs.readFileSync(0, 'utf-8').trim(), 10);
console.log(num * 10);
`,
		TimeLimitMs: 3000,
		TestCases: []dto.TestCase{
			{ID: "1", Input: "7\n", ExpectedOutput: "70\n"},
		},
	}

	res, err := adapter.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(res.TestResults) != 1 || !res.TestResults[0].Passed {
		t.Fatalf("expected test case to pass: %+v", res.TestResults)
	}
}

func TestLocalExecutor_TypeScript(t *testing.T) {
	adapter := executor.NewLocalSubprocessAdapter()

	job := &dto.Job{
		ID:       "test-ts",
		Language: "typescript",
		Code: `const val: number = 42;
console.log(val + 8);
`,
		TimeLimitMs: 3000,
	}

	res, err := adapter.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if res.ExitCode != 0 || strings.TrimSpace(res.Stdout) != "50" {
		t.Fatalf("expected stdout 50, got: %q (stderr: %s)", res.Stdout, res.Stderr)
	}
}

func TestLocalExecutor_C_CompileOnce(t *testing.T) {
	adapter := executor.NewLocalSubprocessAdapter()

	job := &dto.Job{
		ID:       "test-c",
		Language: "c",
		Code: `#include <stdio.h>
int main() {
    int x;
    if (scanf("%d", &x) == 1) {
        printf("%d\n", x * 2);
    }
    return 0;
}
`,
		TimeLimitMs: 3000,
		TestCases: []dto.TestCase{
			{ID: "1", Input: "10\n", ExpectedOutput: "20\n"},
			{ID: "2", Input: "50\n", ExpectedOutput: "100\n"},
			{ID: "3", Input: "7\n", ExpectedOutput: "14\n"},
		},
	}

	res, err := adapter.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if res.CompilationTimeMs <= 0 {
		t.Logf("Notice: C compilation took %d ms", res.CompilationTimeMs)
	}

	if len(res.TestResults) != 3 {
		t.Fatalf("expected 3 test results, got %d", len(res.TestResults))
	}

	for _, tr := range res.TestResults {
		if !tr.Passed {
			t.Errorf("test case %s failed: actual %q, expected %q", tr.TestCaseID, tr.ActualOutput, tr.ExpectedOutput)
		}
	}
}

func TestLocalExecutor_CPP_CompileOnce(t *testing.T) {
	adapter := executor.NewLocalSubprocessAdapter()

	job := &dto.Job{
		ID:       "test-cpp",
		Language: "cpp",
		Code: `#include <iostream>
int main() {
    int a, b;
    if (std::cin >> a >> b) {
        std::cout << (a + b) << std::endl;
    }
    return 0;
}
`,
		TimeLimitMs: 3000,
		TestCases: []dto.TestCase{
			{ID: "1", Input: "3 4\n", ExpectedOutput: "7\n"},
			{ID: "2", Input: "100 250\n", ExpectedOutput: "350\n"},
		},
	}

	res, err := adapter.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(res.TestResults) != 2 || !res.TestResults[0].Passed || !res.TestResults[1].Passed {
		t.Fatalf("expected all test cases to pass: %+v", res.TestResults)
	}
}

func TestLocalExecutor_Go_CompileOnce(t *testing.T) {
	adapter := executor.NewLocalSubprocessAdapter()

	job := &dto.Job{
		ID:       "test-go",
		Language: "go",
		Code: `package main
import "fmt"
func main() {
    var s string
    fmt.Scan(&s)
    fmt.Println("Echo:" + s)
}
`,
		TimeLimitMs: 3000,
		TestCases: []dto.TestCase{
			{ID: "1", Input: "Gopher\n", ExpectedOutput: "Echo:Gopher\n"},
			{ID: "2", Input: "Firecracker\n", ExpectedOutput: "Echo:Firecracker\n"},
		},
	}

	res, err := adapter.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if len(res.TestResults) != 2 || !res.TestResults[0].Passed || !res.TestResults[1].Passed {
		t.Fatalf("expected all test cases to pass: %+v", res.TestResults)
	}
}

func TestLocalExecutor_CompileError_SkipsTestCases(t *testing.T) {
	adapter := executor.NewLocalSubprocessAdapter()

	// C++ with deliberate syntax error
	job := &dto.Job{
		ID:       "test-syntax-error",
		Language: "cpp",
		Code: `int main() { this is invalid syntax ;;; }`,
		TestCases: []dto.TestCase{
			{ID: "1", Input: "test\n", ExpectedOutput: "test\n"},
		},
	}

	res, err := adapter.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("execution error: %v", err)
	}

	if res.ExitCode == 0 {
		t.Errorf("expected non-zero exit code on compilation failure")
	}

	if !strings.Contains(res.Stderr, "Compilation Error:") {
		t.Errorf("expected compilation error in stderr, got: %s", res.Stderr)
	}

	// Crucial check: 0 test cases should have been executed!
	if len(res.TestResults) != 0 {
		t.Errorf("expected 0 test cases to run on compilation failure, got %d", len(res.TestResults))
	}
}

func TestLocalExecutor_Timeout(t *testing.T) {
	adapter := executor.NewLocalSubprocessAdapter()

	job := &dto.Job{
		ID:          "test-timeout",
		Language:    "python",
		Code:        "import time\ntime.sleep(5)",
		TimeLimitMs: 400,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := adapter.Execute(ctx, job)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ExitCode != 124 {
		t.Errorf("expected exit code 124 for timeout, got %d", res.ExitCode)
	}
}
