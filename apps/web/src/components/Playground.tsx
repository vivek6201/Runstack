import React, { useState, useEffect, useCallback } from "react"
import {
  Play,
  RotateCcw,
  Plus,
  Trash2,
  CheckCircle2,
  XCircle,
  Clock,
  Cpu,
  Layers,
  Terminal,
  AlertTriangle,
  Loader2,
  Copy,
  Check,
} from "lucide-react"
import type { Language, TestCase, Job, User } from "../types"
import { api } from "../lib/api"

interface PlaygroundProps {
  user: User | null
  onOpenAuth: () => void
}

const DEFAULT_TEMPLATES: Record<Language, { code: string; testCases: TestCase[] }> = {
  python: {
    code: `import sys

def main():
    lines = sys.stdin.read().split()
    if not lines:
        return
    if len(lines) >= 2:
        a = int(lines[0])
        b = int(lines[1])
        print(a + b)
    else:
        print(f"Hello, {lines[0]}!")

if __name__ == "__main__":
    main()
`,
    testCases: [
      { id: "1", input: "5 10", expected_output: "15" },
      { id: "2", input: "123 456", expected_output: "579" },
      { id: "3", input: "-10 10", expected_output: "0" },
    ],
  },
  cpp: {
    code: `#include <iostream>

using namespace std;

int main() {
    int a, b;
    if (cin >> a >> b) {
        cout << (a + b) << endl;
    } else {
        cout << "Hello from C++!" << endl;
    }
    return 0;
}
`,
    testCases: [
      { id: "1", input: "5 10", expected_output: "15" },
      { id: "2", input: "123 456", expected_output: "579" },
      { id: "3", input: "-10 10", expected_output: "0" },
    ],
  },
  c: {
    code: `#include <stdio.h>

int main() {
    int a, b;
    if (scanf("%d %d", &a, &b) == 2) {
        printf("%d\\n", a + b);
    } else {
        printf("Hello from C!\\n");
    }
    return 0;
}
`,
    testCases: [
      { id: "1", input: "5 10", expected_output: "15" },
      { id: "2", input: "123 456", expected_output: "579" },
      { id: "3", input: "-10 10", expected_output: "0" },
    ],
  },
  go: {
    code: `package main

import (
	"fmt"
)

func main() {
	var a, b int
	if _, err := fmt.Scan(&a, &b); err == nil {
		fmt.Println(a + b)
	} else {
		fmt.Println("Hello from Go!")
	}
}
`,
    testCases: [
      { id: "1", input: "5 10", expected_output: "15" },
      { id: "2", input: "123 456", expected_output: "579" },
      { id: "3", input: "-10 10", expected_output: "0" },
    ],
  },
  java: {
    code: `import java.util.Scanner;

public class Main {
    public static void main(String[] args) {
        Scanner scanner = new Scanner(System.in);
        if (scanner.hasNextInt()) {
            int a = scanner.nextInt();
            int b = scanner.nextInt();
            System.out.println(a + b);
        } else {
            System.out.println("Hello from Java!");
        }
    }
}
`,
    testCases: [
      { id: "1", input: "5 10", expected_output: "15" },
      { id: "2", input: "123 456", expected_output: "579" },
      { id: "3", input: "-10 10", expected_output: "0" },
    ],
  },
  javascript: {
    code: `const fs = require('fs');

function main() {
    const input = fs.readFileSync(0, 'utf-8').trim();
    if (!input) return;
    const parts = input.split(/\\s+/);
    if (parts.length >= 2) {
        console.log(Number(parts[0]) + Number(parts[1]));
    } else {
        console.log(\`Hello, \${parts[0]}!\`);
    }
}

main();
`,
    testCases: [
      { id: "1", input: "5 10", expected_output: "15" },
      { id: "2", input: "123 456", expected_output: "579" },
      { id: "3", input: "-10 10", expected_output: "0" },
    ],
  },
  typescript: {
    code: `import * as fs from 'fs';

function main(): void {
    const input: string = fs.readFileSync(0, 'utf-8').trim();
    if (!input) return;
    const parts: string[] = input.split(/\\s+/);
    if (parts.length >= 2) {
        const sum: number = Number(parts[0]) + Number(parts[1]);
        console.log(sum);
    } else {
        console.log(\`Hello from TypeScript: \${parts[0]}\`);
    }
}

main();
`,
    testCases: [
      { id: "1", input: "5 10", expected_output: "15" },
      { id: "2", input: "123 456", expected_output: "579" },
      { id: "3", input: "-10 10", expected_output: "0" },
    ],
  },
}

export const Playground: React.FC<PlaygroundProps> = ({ user, onOpenAuth }) => {
  const [language, setLanguage] = useState<Language>("python")
  const [code, setCode] = useState<string>(DEFAULT_TEMPLATES.python.code)
  const [testCases, setTestCases] = useState<TestCase[]>(DEFAULT_TEMPLATES.python.testCases)
  const [timeLimitMs, setTimeLimitMs] = useState<number>(3000)
  const [memoryLimitMB, setMemoryLimitMB] = useState<number>(128)

  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<Job | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [activeResultTab, setActiveResultTab] = useState<"tests" | "stdout" | "stderr">("tests")
  const [copied, setCopied] = useState(false)

  // Language switch
  const handleLanguageChange = (newLang: Language) => {
    setLanguage(newLang)
    setCode(DEFAULT_TEMPLATES[newLang].code)
    setTestCases(DEFAULT_TEMPLATES[newLang].testCases)
    setResult(null)
    setError(null)
  }

  // Reset current template
  const handleResetTemplate = () => {
    setCode(DEFAULT_TEMPLATES[language].code)
    setTestCases(DEFAULT_TEMPLATES[language].testCases)
    setResult(null)
    setError(null)
  }

  // Add test case
  const handleAddTestCase = () => {
    const newId = String(testCases.length + 1)
    setTestCases([...testCases, { id: newId, input: "", expected_output: "" }])
  }

  // Remove test case
  const handleRemoveTestCase = (index: number) => {
    setTestCases(testCases.filter((_, i) => i !== index))
  }

  // Update test case
  const handleUpdateTestCase = (index: number, field: "input" | "expected_output", val: string) => {
    const updated = [...testCases]
    updated[index] = { ...updated[index], [field]: val }
    setTestCases(updated)
  }

  // Execute Code
  const handleRun = useCallback(async () => {
    if (!user) {
      onOpenAuth()
      return
    }

    setLoading(true)
    setError(null)

    try {
      const job = await api.executeCode({
        language,
        code,
        test_cases: testCases,
        time_limit_ms: timeLimitMs,
        memory_limit_mb: memoryLimitMB,
      })
      setResult(job)
      if (job.result?.stderr && !job.result?.stdout) {
        setActiveResultTab("stderr")
      } else {
        setActiveResultTab("tests")
      }
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError("Failed to execute code")
      }
    } finally {
      setLoading(false)
    }
  }, [user, language, code, testCases, timeLimitMs, memoryLimitMB, onOpenAuth])

  // Cmd+Enter or Ctrl+Enter to run
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
        e.preventDefault()
        handleRun()
      }
    }
    window.addEventListener("keydown", handleKeyDown)
    return () => window.removeEventListener("keydown", handleKeyDown)
  }, [handleRun])

  const copyCode = () => {
    navigator.clipboard.writeText(code)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const passedCount = result?.result?.test_results?.filter((t) => t.passed).length ?? 0
  const totalCount = result?.result?.test_results?.length ?? 0

  return (
    <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 pb-12">
      {/* Left Column: Code Editor & Controls (7 cols) */}
      <div className="lg:col-span-7 flex flex-col gap-4">
        {/* Editor Toolbar */}
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-border bg-card p-3 shadow-xs">
          {/* Language Selector */}
          <div className="flex items-center gap-2">
            <span className="text-xs font-semibold uppercase text-muted-foreground tracking-wider">
              Language:
            </span>
            <select
              value={language}
              onChange={(e) => handleLanguageChange(e.target.value as Language)}
              className="rounded-md border border-input bg-background px-3 py-1.5 text-xs font-semibold text-foreground focus:border-primary focus:outline-hidden"
            >
              <option value="python">Python 3.12</option>
              <option value="cpp">C++ (g++ 13)</option>
              <option value="c">C (gcc 13)</option>
              <option value="go">Go 1.26</option>
              <option value="java">Java 17 (OpenJDK)</option>
              <option value="javascript">JavaScript (Node.js 20)</option>
              <option value="typescript">TypeScript (Bun/Node)</option>
            </select>
          </div>

          {/* Quick Controls */}
          <div className="flex items-center gap-2">
            <button
              onClick={copyCode}
              className="flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
              title="Copy source code"
            >
              {copied ? <Check className="size-3.5 text-emerald-500" /> : <Copy className="size-3.5" />}
              <span>{copied ? "Copied" : "Copy"}</span>
            </button>
            <button
              onClick={handleResetTemplate}
              className="flex items-center gap-1.5 rounded-md border border-border px-2.5 py-1.5 text-xs font-medium text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
              title="Reset starter template"
            >
              <RotateCcw className="size-3.5" />
              <span>Reset</span>
            </button>

            {/* Run Button */}
            <button
              onClick={handleRun}
              disabled={loading}
              className="flex items-center gap-2 rounded-md bg-gradient-to-r from-emerald-600 to-teal-600 px-4 py-1.5 text-xs font-bold text-white shadow-sm hover:from-emerald-500 hover:to-teal-500 disabled:opacity-50 transition-all cursor-pointer"
            >
              {loading ? <Loader2 className="size-3.5 animate-spin" /> : <Play className="size-3.5 fill-current" />}
              <span>Run Code</span>
              <span className="hidden sm:inline-block rounded bg-black/20 px-1.5 py-0.5 text-[10px] font-mono">
                ⌘↵
              </span>
            </button>
          </div>
        </div>

        {/* Code Editor Box */}
        <div className="relative rounded-xl border border-border bg-card shadow-sm overflow-hidden flex flex-col min-h-[440px]">
          <div className="flex items-center justify-between border-b border-border bg-muted/40 px-4 py-2 text-xs font-mono text-muted-foreground">
            <span>solution.{language === "cpp" ? "cpp" : language === "c" ? "c" : language === "go" ? "go" : language === "java" ? "java" : language === "javascript" ? "js" : language === "typescript" ? "ts" : "py"}</span>
            <span>UTF-8 • Compile-Once, Run-N-Times</span>
          </div>
          <div className="relative flex-1 flex">
            {/* Line numbers dummy gutter */}
            <div className="select-none bg-muted/20 border-r border-border/40 px-3 py-3 text-right font-mono text-xs text-muted-foreground/50 leading-relaxed">
              {Array.from({ length: Math.max(16, code.split("\n").length) }, (_, i) => (
                <div key={i}>{i + 1}</div>
              ))}
            </div>
            {/* Source code textarea */}
            <textarea
              value={code}
              onChange={(e) => setCode(e.target.value)}
              spellCheck={false}
              className="flex-1 w-full resize-none bg-transparent p-3 font-mono text-xs text-foreground focus:outline-hidden leading-relaxed tab-2"
              rows={18}
            />
          </div>
        </div>

        {/* Resource Limits Configuration */}
        <div className="grid grid-cols-2 gap-4 rounded-xl border border-border bg-card p-4 shadow-xs">
          <div>
            <div className="flex justify-between text-xs font-medium mb-1.5">
              <span className="text-muted-foreground">Execution Timeout</span>
              <span className="font-mono text-foreground font-semibold">{timeLimitMs}ms</span>
            </div>
            <input
              type="range"
              min={500}
              max={10000}
              step={500}
              value={timeLimitMs}
              onChange={(e) => setTimeLimitMs(Number(e.target.value))}
              className="w-full accent-primary cursor-pointer"
            />
          </div>
          <div>
            <div className="flex justify-between text-xs font-medium mb-1.5">
              <span className="text-muted-foreground">Memory Ceiling</span>
              <span className="font-mono text-foreground font-semibold">{memoryLimitMB}MB</span>
            </div>
            <input
              type="range"
              min={32}
              max={512}
              step={32}
              value={memoryLimitMB}
              onChange={(e) => setMemoryLimitMB(Number(e.target.value))}
              className="w-full accent-primary cursor-pointer"
            />
          </div>
        </div>
      </div>

      {/* Right Column: Test Cases & Execution Results (5 cols) */}
      <div className="lg:col-span-5 flex flex-col gap-4">
        {/* Test Cases Editor Card */}
        <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden flex flex-col">
          <div className="flex items-center justify-between border-b border-border bg-muted/40 px-4 py-2.5">
            <div className="flex items-center gap-2">
              <Layers className="size-4 text-muted-foreground" />
              <span className="text-xs font-bold text-foreground">
                Test Cases ({testCases.length})
              </span>
            </div>
            <button
              onClick={handleAddTestCase}
              className="flex items-center gap-1 rounded border border-border bg-background px-2 py-1 text-[11px] font-semibold text-foreground hover:bg-muted"
            >
              <Plus className="size-3" />
              <span>Add Case</span>
            </button>
          </div>

          <div className="p-4 space-y-3 max-h-[220px] overflow-y-auto">
            {testCases.map((tc, idx) => (
              <div
                key={tc.id || idx}
                className="group relative rounded-lg border border-border bg-background p-2.5 shadow-2xs"
              >
                <div className="flex items-center justify-between mb-2">
                  <span className="text-[11px] font-bold text-muted-foreground font-mono">
                    Case #{idx + 1}
                  </span>
                  {testCases.length > 1 && (
                    <button
                      onClick={() => handleRemoveTestCase(idx)}
                      className="text-muted-foreground opacity-60 hover:opacity-100 hover:text-destructive"
                      title="Remove test case"
                    >
                      <Trash2 className="size-3.5" />
                    </button>
                  )}
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div>
                    <label className="block text-[10px] font-medium text-muted-foreground uppercase mb-0.5">
                      Input (stdin)
                    </label>
                    <input
                      type="text"
                      value={tc.input}
                      onChange={(e) => handleUpdateTestCase(idx, "input", e.target.value)}
                      placeholder="e.g. 5 10"
                      className="w-full rounded border border-input bg-card px-2 py-1 font-mono text-xs text-foreground focus:border-primary focus:outline-hidden"
                    />
                  </div>
                  <div>
                    <label className="block text-[10px] font-medium text-muted-foreground uppercase mb-0.5">
                      Expected Output
                    </label>
                    <input
                      type="text"
                      value={tc.expected_output}
                      onChange={(e) =>
                        handleUpdateTestCase(idx, "expected_output", e.target.value)
                      }
                      placeholder="e.g. 15"
                      className="w-full rounded border border-input bg-card px-2 py-1 font-mono text-xs text-foreground focus:border-primary focus:outline-hidden"
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Results Panel */}
        <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden flex flex-col flex-1 min-h-[300px]">
          {/* Status Header */}
          <div className="flex items-center justify-between border-b border-border bg-muted/40 px-4 py-2.5">
            <div className="flex items-center gap-2">
              <Terminal className="size-4 text-muted-foreground" />
              <span className="text-xs font-bold text-foreground">Execution Output</span>
            </div>

            {result && (
              <div className="flex items-center gap-1.5">
                {result.status === "COMPLETED" && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2.5 py-0.5 text-xs font-semibold text-emerald-600 dark:text-emerald-400">
                    <CheckCircle2 className="size-3" />
                    {passedCount === totalCount ? "All Passed" : `${passedCount}/${totalCount} Passed`}
                  </span>
                )}
                {result.status === "FAILED" && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-destructive/15 px-2.5 py-0.5 text-xs font-semibold text-destructive">
                    <XCircle className="size-3" />
                    Failed
                  </span>
                )}
                {result.status === "COMPILE_ERROR" && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2.5 py-0.5 text-xs font-semibold text-amber-600 dark:text-amber-400">
                    <AlertTriangle className="size-3" />
                    Compile Error
                  </span>
                )}
                {result.status === "TIMED_OUT" && (
                  <span className="inline-flex items-center gap-1 rounded-full bg-purple-500/15 px-2.5 py-0.5 text-xs font-semibold text-purple-600 dark:text-purple-400">
                    <Clock className="size-3" />
                    Timed Out
                  </span>
                )}
              </div>
            )}
          </div>

          {/* Metrics bar */}
          {result?.result && (
            <div className="grid grid-cols-4 border-b border-border/60 bg-muted/20 px-3 py-2 text-[11px] font-mono">
              <div>
                <span className="text-muted-foreground">Total Time: </span>
                <span className="font-semibold text-foreground">{result.result.time_taken_ms}ms</span>
              </div>
              <div>
                <span className="text-muted-foreground">Compile: </span>
                <span className="font-semibold text-foreground">
                  {result.result.compilation_time_ms ? `${result.result.compilation_time_ms}ms` : "N/A"}
                </span>
              </div>
              <div>
                <span className="text-muted-foreground">Peak RAM: </span>
                <span className="font-semibold text-foreground">{result.result.memory_used_kb} KB</span>
              </div>
              <div>
                <span className="text-muted-foreground">Exit: </span>
                <span className={`font-semibold ${result.result.exit_code === 0 ? "text-emerald-500" : "text-destructive"}`}>
                  {result.result.exit_code}
                </span>
              </div>
            </div>
          )}

          {/* Result Tabs */}
          <div className="flex border-b border-border bg-background px-4">
            <button
              onClick={() => setActiveResultTab("tests")}
              className={`border-b-2 py-2 px-3 text-xs font-semibold transition-colors ${
                activeResultTab === "tests"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              }`}
            >
              Test Results
            </button>
            <button
              onClick={() => setActiveResultTab("stdout")}
              className={`border-b-2 py-2 px-3 text-xs font-semibold transition-colors ${
                activeResultTab === "stdout"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              }`}
            >
              Stdout
            </button>
            <button
              onClick={() => setActiveResultTab("stderr")}
              className={`border-b-2 py-2 px-3 text-xs font-semibold transition-colors ${
                activeResultTab === "stderr"
                  ? "border-primary text-primary"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              }`}
            >
              Stderr / Error
            </button>
          </div>

          {/* Results Tab Content */}
          <div className="flex-1 p-4 overflow-y-auto">
            {error && (
              <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
                {error}
              </div>
            )}

            {!result && !error && !loading && (
              <div className="flex h-44 flex-col items-center justify-center text-center text-muted-foreground">
                <Cpu className="size-8 mb-2 stroke-1 opacity-50" />
                <p className="text-xs">Click &quot;Run Code&quot; or press ⌘+Enter to execute.</p>
                <p className="text-[11px] text-muted-foreground/60 mt-1">
                  Executes inside isolated, secure Firecracker microVM.
                </p>
              </div>
            )}

            {loading && (
              <div className="flex h-44 flex-col items-center justify-center text-center text-muted-foreground">
                <Loader2 className="size-8 animate-spin text-primary mb-2" />
                <p className="text-xs font-medium">Booting microVM & executing code...</p>
                <p className="text-[11px] text-muted-foreground/60 mt-1">
                  Compile once, run {testCases.length} test cases
                </p>
              </div>
            )}

            {result && activeResultTab === "tests" && (
              <div className="space-y-2.5">
                {result.result?.test_results && result.result.test_results.length > 0 ? (
                  result.result.test_results.map((tr, i) => (
                    <div
                      key={tr.test_case_id || i}
                      className={`rounded-lg border p-3 font-mono text-xs transition-colors ${
                        tr.passed
                          ? "border-emerald-500/20 bg-emerald-500/5 dark:bg-emerald-500/10"
                          : "border-destructive/30 bg-destructive/5 dark:bg-destructive/10"
                      }`}
                    >
                      <div className="flex items-center justify-between mb-1.5">
                        <div className="flex items-center gap-1.5 font-bold">
                          {tr.passed ? (
                            <CheckCircle2 className="size-3.5 text-emerald-500" />
                          ) : (
                            <XCircle className="size-3.5 text-destructive" />
                          )}
                          <span className={tr.passed ? "text-emerald-600 dark:text-emerald-400" : "text-destructive"}>
                            Test Case #{i + 1} • {tr.passed ? "Passed" : "Failed"}
                          </span>
                        </div>
                        <span className="text-[10px] text-muted-foreground">{tr.time_taken_ms}ms</span>
                      </div>
                      <div className="grid grid-cols-2 gap-2 text-[11px] pt-1 border-t border-border/40">
                        <div>
                          <span className="text-muted-foreground">Expected: </span>
                          <span className="font-semibold text-foreground">{JSON.stringify(tr.expected_output)}</span>
                        </div>
                        <div>
                          <span className="text-muted-foreground">Actual: </span>
                          <span className={`font-semibold ${tr.passed ? "text-emerald-600 dark:text-emerald-400" : "text-destructive"}`}>
                            {JSON.stringify(tr.actual_output)}
                          </span>
                        </div>
                      </div>
                    </div>
                  ))
                ) : (
                  <div className="text-xs text-muted-foreground py-4 text-center">
                    No discrete test case results generated. Check Stdout/Stderr tabs.
                  </div>
                )}
              </div>
            )}

            {result && activeResultTab === "stdout" && (
              <pre className="rounded-lg bg-black/80 p-3 font-mono text-xs text-emerald-400 whitespace-pre-wrap break-all min-h-[140px]">
                {result.result?.stdout || "(empty stdout)"}
              </pre>
            )}

            {result && activeResultTab === "stderr" && (
              <pre className="rounded-lg bg-black/80 p-3 font-mono text-xs text-rose-400 whitespace-pre-wrap break-all min-h-[140px]">
                {result.error || result.result?.stderr || "(empty stderr)"}
              </pre>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
