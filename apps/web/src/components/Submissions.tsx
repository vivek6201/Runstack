import React, { useState, useEffect } from "react"
import {
  History,
  CheckCircle2,
  XCircle,
  Clock,
  Loader2,
  X,
  FileCode,
  Filter,
  ArrowRight,
} from "lucide-react"
import type { SubmissionSummary, SubmissionDetail, User } from "../types"
import { api } from "../lib/api"

interface SubmissionsProps {
  user: User | null
  onOpenAuth: () => void
}

export const Submissions: React.FC<SubmissionsProps> = ({ user, onOpenAuth }) => {
  const [submissions, setSubmissions] = useState<SubmissionSummary[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Filters
  const [filterLang, setFilterLang] = useState("")
  const [filterStatus, setFilterStatus] = useState("")

  // Detail Drawer
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [detail, setDetail] = useState<SubmissionDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const fetchSubmissions = async () => {
    if (!user) return
    setLoading(true)
    setError(null)
    try {
      const res = await api.listSubmissions({
        page,
        limit: 15,
        language: filterLang || undefined,
        status: filterStatus || undefined,
      })
      setSubmissions(res.items || [])
      setTotal(res.total || 0)
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError("Failed to fetch submissions")
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchSubmissions()
  }, [user, page, filterLang, filterStatus])

  const handleInspect = async (id: string) => {
    setSelectedId(id)
    setDetailLoading(true)
    try {
      const data = await api.getSubmission(id)
      setDetail(data)
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to load submission details")
      setSelectedId(null)
    } finally {
      setDetailLoading(false)
    }
  }

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border p-12 text-center bg-card shadow-xs">
        <div className="mb-4 flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
          <History className="size-6" />
        </div>
        <h3 className="text-lg font-bold text-foreground">Authentication Required</h3>
        <p className="mt-1 max-w-sm text-xs text-muted-foreground mb-6">
          Sign in to view your complete permanent execution history, logs, and test metrics.
        </p>
        <button
          onClick={onOpenAuth}
          className="rounded-md bg-primary px-5 py-2 text-xs font-semibold text-primary-foreground shadow-xs hover:bg-primary/90"
        >
          Sign In / Register
        </button>
      </div>
    )
  }

  return (
    <div className="space-y-6 pb-12">
      {/* Header and Filters */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-foreground">Submission History</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Real-time audit log of all microVM code execution runs ({total} total).
          </p>
        </div>

        {/* Filter Toolbar */}
        <div className="flex items-center gap-2">
          <Filter className="size-3.5 text-muted-foreground" />
          <select
            value={filterLang}
            onChange={(e) => {
              setFilterLang(e.target.value)
              setPage(1)
            }}
            className="rounded-md border border-input bg-card px-2.5 py-1.5 text-xs font-medium text-foreground focus:outline-hidden"
          >
            <option value="">All Languages</option>
            <option value="python">Python</option>
            <option value="cpp">C++</option>
            <option value="c">C</option>
            <option value="go">Go</option>
            <option value="java">Java</option>
            <option value="javascript">JavaScript</option>
            <option value="typescript">TypeScript</option>
          </select>

          <select
            value={filterStatus}
            onChange={(e) => {
              setFilterStatus(e.target.value)
              setPage(1)
            }}
            className="rounded-md border border-input bg-card px-2.5 py-1.5 text-xs font-medium text-foreground focus:outline-hidden"
          >
            <option value="">All Statuses</option>
            <option value="COMPLETED">Completed</option>
            <option value="FAILED">Failed</option>
            <option value="TIMED_OUT">Timed Out</option>
          </select>
        </div>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
          {error}
        </div>
      )}

      {/* Submissions Table Card */}
      <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden">
        {loading ? (
          <div className="flex h-44 items-center justify-center text-muted-foreground">
            <Loader2 className="size-6 animate-spin text-primary" />
          </div>
        ) : submissions.length === 0 ? (
          <div className="flex h-44 flex-col items-center justify-center text-center text-muted-foreground">
            <History className="size-6 mb-2 opacity-40" />
            <p className="text-xs">No execution history found matching the filters.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-border bg-muted/20 text-muted-foreground font-semibold">
                <tr>
                  <th className="px-4 py-2.5">Job ID</th>
                  <th className="px-4 py-2.5">Language</th>
                  <th className="px-4 py-2.5">Status</th>
                  <th className="px-4 py-2.5">Test Results</th>
                  <th className="px-4 py-2.5">Execution Time</th>
                  <th className="px-4 py-2.5">Peak RAM</th>
                  <th className="px-4 py-2.5">Timestamp</th>
                  <th className="px-4 py-2.5 text-right">Inspect</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {submissions.map((sub) => (
                  <tr
                    key={sub.id}
                    onClick={() => handleInspect(sub.id)}
                    className="hover:bg-muted/40 cursor-pointer transition-colors"
                  >
                    <td className="px-4 py-3 font-mono text-[11px] text-muted-foreground">
                      {sub.id.substring(0, 8)}...
                    </td>
                    <td className="px-4 py-3 font-semibold text-foreground capitalize">
                      {sub.language}
                    </td>
                    <td className="px-4 py-3">
                      {sub.status === "COMPLETED" && (
                        <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">
                          <CheckCircle2 className="size-3" />
                          Passed
                        </span>
                      )}
                      {sub.status === "FAILED" && (
                        <span className="inline-flex items-center gap-1 rounded-full bg-destructive/15 px-2 py-0.5 text-[10px] font-semibold text-destructive">
                          <XCircle className="size-3" />
                          Failed
                        </span>
                      )}
                      {sub.status === "TIMED_OUT" && (
                        <span className="inline-flex items-center gap-1 rounded-full bg-purple-500/15 px-2 py-0.5 text-[10px] font-semibold text-purple-600 dark:text-purple-400">
                          <Clock className="size-3" />
                          Timed Out
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 font-mono text-muted-foreground">
                      {sub.total_test_cases > 0
                        ? `${sub.passed_test_cases}/${sub.total_test_cases} passed`
                        : "Direct Run"}
                    </td>
                    <td className="px-4 py-3 font-mono text-muted-foreground">{sub.time_taken_ms}ms</td>
                    <td className="px-4 py-3 font-mono text-muted-foreground">{sub.memory_used_kb} KB</td>
                    <td className="px-4 py-3 font-mono text-muted-foreground">
                      {new Date(sub.created_at).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit", second: "2-digit" })}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <span className="inline-flex items-center gap-1 text-[11px] font-semibold text-primary hover:underline">
                        Details <ArrowRight className="size-3" />
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Pagination controls */}
        {total > 15 && (
          <div className="flex items-center justify-between border-t border-border px-4 py-2.5 text-xs text-muted-foreground">
            <span>
              Page {page} of {Math.ceil(total / 15)}
            </span>
            <div className="flex gap-2">
              <button
                disabled={page <= 1}
                onClick={() => setPage(page - 1)}
                className="rounded border border-border px-2.5 py-1 disabled:opacity-40 hover:bg-muted"
              >
                Previous
              </button>
              <button
                disabled={page >= Math.ceil(total / 15)}
                onClick={() => setPage(page + 1)}
                className="rounded border border-border px-2.5 py-1 disabled:opacity-40 hover:bg-muted"
              >
                Next
              </button>
            </div>
          </div>
        )}
      </div>

      {/* Inspect Detail Modal */}
      {selectedId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs p-4">
          <div className="relative w-full max-w-2xl max-h-[90vh] flex flex-col rounded-xl border border-border bg-card shadow-2xl overflow-hidden animate-in fade-in zoom-in-95 duration-150">
            <div className="flex items-center justify-between border-b border-border bg-muted/40 px-5 py-3">
              <div className="flex items-center gap-2">
                <FileCode className="size-4 text-primary" />
                <h3 className="text-sm font-bold text-foreground">
                  Submission Details • {selectedId.substring(0, 13)}
                </h3>
              </div>
              <button
                onClick={() => {
                  setSelectedId(null)
                  setDetail(null)
                }}
                className="text-muted-foreground hover:text-foreground"
              >
                <X className="size-5" />
              </button>
            </div>

            <div className="flex-1 overflow-y-auto p-5 space-y-4">
              {detailLoading || !detail ? (
                <div className="flex h-44 items-center justify-center">
                  <Loader2 className="size-6 animate-spin text-primary" />
                </div>
              ) : (
                <>
                  {/* Summary Bar */}
                  <div className="grid grid-cols-4 gap-2 rounded-lg border border-border bg-muted/20 p-3 text-xs font-mono">
                    <div>
                      <span className="text-muted-foreground block text-[10px] uppercase">Language</span>
                      <span className="font-semibold text-foreground uppercase">{detail.language}</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[10px] uppercase">Duration</span>
                      <span className="font-semibold text-foreground">{detail.time_taken_ms}ms</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[10px] uppercase">Peak RAM</span>
                      <span className="font-semibold text-foreground">{detail.memory_used_kb} KB</span>
                    </div>
                    <div>
                      <span className="text-muted-foreground block text-[10px] uppercase">Status</span>
                      <span className="font-semibold text-emerald-500">{detail.status}</span>
                    </div>
                  </div>

                  {/* Source Code */}
                  <div>
                    <label className="block text-xs font-bold text-muted-foreground uppercase mb-1">
                      Submitted Code
                    </label>
                    <pre className="rounded-lg bg-black/80 p-3 font-mono text-xs text-foreground overflow-x-auto max-h-48">
                      {detail.code}
                    </pre>
                  </div>

                  {/* Test Cases */}
                  {detail.test_results && detail.test_results.length > 0 && (
                    <div>
                      <label className="block text-xs font-bold text-muted-foreground uppercase mb-1">
                        Test Cases ({detail.test_results.length})
                      </label>
                      <div className="space-y-2">
                        {detail.test_results.map((tr, i) => (
                          <div
                            key={i}
                            className={`rounded-md border p-2.5 font-mono text-xs ${
                              tr.passed ? "border-emerald-500/20 bg-emerald-500/5" : "border-destructive/20 bg-destructive/5"
                            }`}
                          >
                            <div className="flex justify-between font-semibold mb-1">
                              <span className={tr.passed ? "text-emerald-500" : "text-destructive"}>
                                #{i + 1} {tr.passed ? "PASSED" : "FAILED"}
                              </span>
                              <span className="text-muted-foreground">{tr.time_taken_ms}ms</span>
                            </div>
                            <div className="grid grid-cols-2 gap-2 text-[11px] text-muted-foreground">
                              <div>Expected: {JSON.stringify(tr.expected_output)}</div>
                              <div>Actual: {JSON.stringify(tr.actual_output)}</div>
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Stdout / Stderr */}
                  {detail.stdout && (
                    <div>
                      <label className="block text-xs font-bold text-muted-foreground uppercase mb-1">
                        Stdout
                      </label>
                      <pre className="rounded-lg bg-black/80 p-3 font-mono text-xs text-emerald-400 overflow-x-auto max-h-36">
                        {detail.stdout}
                      </pre>
                    </div>
                  )}
                  {detail.stderr && (
                    <div>
                      <label className="block text-xs font-bold text-muted-foreground uppercase mb-1">
                        Stderr
                      </label>
                      <pre className="rounded-lg bg-black/80 p-3 font-mono text-xs text-rose-400 overflow-x-auto max-h-36">
                        {detail.stderr}
                      </pre>
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
