import React, { useState } from "react"
import { Copy, Check, Terminal, Globe } from "lucide-react"

export const Quickstart: React.FC = () => {
  const [activeLang, setActiveLang] = useState<"curl" | "typescript" | "python" | "go">("curl")
  const [copied, setCopied] = useState(false)

  const SNIPPETS = {
    curl: `# 1. Submit execution job
curl -X POST http://localhost:8081/v1/submit \\
  -H "Content-Type: application/json" \\
  -H "X-API-Key: YOUR_API_KEY" \\
  -d '{
    "language": "python",
    "code": "import sys; print(int(sys.stdin.read()) * 2)",
    "test_cases": [
      {"id": "1", "input": "21", "expected_output": "42\\n"}
    ],
    "time_limit_ms": 2000,
    "memory_limit_mb": 128
  }'

# Response (< 3ms):
# {"job_id": "9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d", "status": "QUEUED"}

# 2. Poll job status until COMPLETED:
curl -X GET http://localhost:8081/v1/status/9b1deb4d-3b7d-4bad-9bdd-2b0d7b3dcb6d \\
  -H "X-API-Key: YOUR_API_KEY"`,

    typescript: `import axios from 'axios';

const API_KEY = 'cr_live_your_secret_key';
const ENGINE_URL = 'http://localhost:8081';

async function executeCode() {
  // 1. Submit code to queue
  const { data: job } = await axios.post(
    \`\${ENGINE_URL}/v1/submit\`,
    {
      language: 'cpp',
      code: \`
        #include <iostream>
        int main() {
          int n; std::cin >> n;
          std::cout << n * n << std::endl;
          return 0;
        }
      \`,
      test_cases: [
        { id: '1', input: '7', expected_output: '49\\n' },
        { id: '2', input: '12', expected_output: '144\\n' }
      ],
      time_limit_ms: 2000,
      memory_limit_mb: 128
    },
    { headers: { 'X-API-Key': API_KEY } }
  );

  console.log('Submitted Job ID:', job.job_id);

  // 2. Poll for status
  while (true) {
    await new Promise((r) => setTimeout(r, 100));
    const { data: status } = await axios.get(
      \`\${ENGINE_URL}/v1/status/\${job.job_id}\`,
      { headers: { 'X-API-Key': API_KEY } }
    );

    if (status.status === 'COMPLETED' || status.status === 'FAILED') {
      console.log('Execution Finished:', status.result);
      break;
    }
  }
}

executeCode();`,

    python: `import time
import requests

API_KEY = "cr_live_your_secret_key"
ENGINE_URL = "http://localhost:8081"

# 1. Enqueue job
submit_resp = requests.post(
    f"{ENGINE_URL}/v1/submit",
    headers={"X-API-Key": API_KEY},
    json={
        "language": "go",
        "code": """package main
import "fmt"
func main() {
    var x int
    fmt.Scan(&x)
    fmt.Println(x * 10)
}""",
        "test_cases": [{"id": "1", "input": "5", "expected_output": "50\\n"}],
        "time_limit_ms": 2000,
        "memory_limit_mb": 128,
    },
)

job_id = submit_resp.json()["job_id"]
print(f"Job queued: {job_id}")

# 2. Poll for result
while True:
    time.sleep(0.1)
    status_resp = requests.get(
        f"{ENGINE_URL}/v1/status/{job_id}",
        headers={"X-API-Key": API_KEY},
    )
    data = status_resp.json()
    if data["status"] in ["COMPLETED", "FAILED", "COMPILE_ERROR"]:
        print("Results:", data["result"])
        break`,

    go: `package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func main() {
	apiKey := "cr_live_your_secret_key"
	engineURL := "http://localhost:8081"

	payload := map[string]interface{}{
		"language": "python",
		"code":     "import sys\\nprint('Hello ' + sys.stdin.read().strip())",
		"test_cases": []map[string]string{
			{"id": "1", "input": "World", "expected_output": "Hello World\\n"},
		},
		"time_limit_ms":   2000,
		"memory_limit_mb": 128,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", engineURL+"/v1/submit", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := http.DefaultClient.Do(req)
	var submitResp struct {
		JobID string \`json:"job_id"\`
	}
	_ = json.NewDecoder(resp.Body).Decode(&submitResp)
	fmt.Println("Job ID:", submitResp.JobID)

	// Poll until done
	for {
		time.Sleep(100 * time.Millisecond)
		pollReq, _ := http.NewRequest("GET", engineURL+"/v1/status/"+submitResp.JobID, nil)
		pollReq.Header.Set("X-API-Key", apiKey)
		pResp, _ := http.DefaultClient.Do(pollReq)
		var state map[string]interface{}
		_ = json.NewDecoder(pResp.Body).Decode(&state)

		status := state["status"].(string)
		if status == "COMPLETED" || status == "FAILED" {
			fmt.Println("Done:", state["result"])
			break
		}
	}
}`,
  }

  const copySnippet = () => {
    navigator.clipboard.writeText(SNIPPETS[activeLang])
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-6 pb-12">
      <div>
        <h2 className="text-xl font-bold tracking-tight text-foreground">API Integration Guide</h2>
        <p className="text-xs text-muted-foreground mt-0.5">
          Send remote compilation and execution jobs to the CodeRunner microVM engine in any language.
        </p>
      </div>

      <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden">
        {/* Language Tabs */}
        <div className="flex items-center justify-between border-b border-border bg-muted/40 px-4 py-2">
          <div className="flex items-center gap-1">
            <button
              onClick={() => setActiveLang("curl")}
              className={`rounded-md px-3 py-1.5 text-xs font-semibold transition-colors ${
                activeLang === "curl" ? "bg-background text-foreground shadow-2xs" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              cURL
            </button>
            <button
              onClick={() => setActiveLang("typescript")}
              className={`rounded-md px-3 py-1.5 text-xs font-semibold transition-colors ${
                activeLang === "typescript" ? "bg-background text-foreground shadow-2xs" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              TypeScript / Node
            </button>
            <button
              onClick={() => setActiveLang("python")}
              className={`rounded-md px-3 py-1.5 text-xs font-semibold transition-colors ${
                activeLang === "python" ? "bg-background text-foreground shadow-2xs" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              Python
            </button>
            <button
              onClick={() => setActiveLang("go")}
              className={`rounded-md px-3 py-1.5 text-xs font-semibold transition-colors ${
                activeLang === "go" ? "bg-background text-foreground shadow-2xs" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              Go
            </button>
          </div>

          <button
            onClick={copySnippet}
            className="flex items-center gap-1.5 rounded-md border border-border bg-background px-2.5 py-1 text-xs font-medium text-foreground hover:bg-muted"
          >
            {copied ? <Check className="size-3.5 text-emerald-500" /> : <Copy className="size-3.5" />}
            <span>{copied ? "Copied" : "Copy Code"}</span>
          </button>
        </div>

        {/* Code Snippet Box */}
        <div className="bg-black/90 p-5">
          <pre className="font-mono text-xs text-emerald-400 overflow-x-auto whitespace-pre leading-relaxed">
            {SNIPPETS[activeLang]}
          </pre>
        </div>
      </div>

      {/* API Reference summary */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 mb-2 text-foreground font-bold text-xs">
            <Terminal className="size-4 text-primary" />
            <span>POST /v1/submit</span>
          </div>
          <p className="text-xs text-muted-foreground mb-3">
            Submits code for async execution in a microVM. Returns in &lt; 3ms with a unique <code className="text-primary font-mono">job_id</code> and state <code className="text-primary font-mono">QUEUED</code>.
          </p>
          <div className="rounded bg-muted/30 p-2 font-mono text-[11px] text-muted-foreground">
            Requires header: <code>X-API-Key: cr_live_...</code>
          </div>
        </div>

        <div className="rounded-xl border border-border bg-card p-4">
          <div className="flex items-center gap-2 mb-2 text-foreground font-bold text-xs">
            <Globe className="size-4 text-primary" />
            <span>GET /v1/status/:id</span>
          </div>
          <p className="text-xs text-muted-foreground mb-3">
            Retrieves current execution state and full results array including test case outputs, peak RAM (KB), compile time, and stdout/stderr.
          </p>
          <div className="rounded bg-muted/30 p-2 font-mono text-[11px] text-muted-foreground">
            Cached in Redis for instant query turnaround
          </div>
        </div>
      </div>
    </div>
  )
}
