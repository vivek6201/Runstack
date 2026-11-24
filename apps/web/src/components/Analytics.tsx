import React, { useState, useEffect } from "react"
import {
  BarChart3,
  TrendingUp,
  Clock,
  Cpu,
  Zap,
  Shield,
  Layers,
  Loader2,
} from "lucide-react"
import type { DashboardStats, User } from "../types"
import { api } from "../lib/api"

interface AnalyticsProps {
  user: User | null
  onOpenAuth: () => void
}

export const Analytics: React.FC<AnalyticsProps> = ({ user, onOpenAuth }) => {
  const [stats, setStats] = useState<DashboardStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!user) return
    setLoading(true)
    api
      .getStats()
      .then((data) => setStats(data))
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [user])

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border p-12 text-center bg-card shadow-xs">
        <div className="mb-4 flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
          <BarChart3 className="size-6" />
        </div>
        <h3 className="text-lg font-bold text-foreground">Authentication Required</h3>
        <p className="mt-1 max-w-sm text-xs text-muted-foreground mb-6">
          Sign in to view real-time latency graphs, monthly quotas, and microVM compute utilization.
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

  const quotaPercent = stats
    ? Math.min(100, Math.round((stats.monthly_runs_used / stats.monthly_runs_limit) * 100))
    : 0

  return (
    <div className="space-y-6 pb-12">
      {/* Page Title */}
      <div>
        <h2 className="text-xl font-bold tracking-tight text-foreground">Platform Analytics</h2>
        <p className="text-xs text-muted-foreground mt-0.5">
          Real-time metrics, throughput counters, and microVM quota consumption.
        </p>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
          {error}
        </div>
      )}

      {loading ? (
        <div className="flex h-44 items-center justify-center text-muted-foreground">
          <Loader2 className="size-6 animate-spin text-primary" />
        </div>
      ) : (
        <>
          {/* Top 4 KPI Cards */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="rounded-xl border border-border bg-card p-5 shadow-xs">
              <div className="flex items-center justify-between text-muted-foreground mb-3">
                <span className="text-xs font-medium">Total Runs</span>
                <Zap className="size-4 text-amber-500" />
              </div>
              <div className="text-2xl font-extrabold text-foreground font-mono">
                {stats?.total_submissions ?? 0}
              </div>
              <p className="mt-1 text-[11px] text-muted-foreground">Lifetime microVM executions</p>
            </div>

            <div className="rounded-xl border border-border bg-card p-5 shadow-xs">
              <div className="flex items-center justify-between text-muted-foreground mb-3">
                <span className="text-xs font-medium">Success Rate</span>
                <TrendingUp className="size-4 text-emerald-500" />
              </div>
              <div className="text-2xl font-extrabold text-foreground font-mono">
                {stats ? `${stats.success_rate.toFixed(1)}%` : "100%"}
              </div>
              <p className="mt-1 text-[11px] text-emerald-600 dark:text-emerald-400 font-medium">
                {stats?.successful_runs ?? 0} successful completions
              </p>
            </div>

            <div className="rounded-xl border border-border bg-card p-5 shadow-xs">
              <div className="flex items-center justify-between text-muted-foreground mb-3">
                <span className="text-xs font-medium">Avg Execution Latency</span>
                <Clock className="size-4 text-primary" />
              </div>
              <div className="text-2xl font-extrabold text-foreground font-mono">
                {stats ? `${Math.round(stats.avg_time_ms)}ms` : "0ms"}
              </div>
              <p className="mt-1 text-[11px] text-muted-foreground">Sub-millisecond Redis dispatch</p>
            </div>

            <div className="rounded-xl border border-border bg-card p-5 shadow-xs">
              <div className="flex items-center justify-between text-muted-foreground mb-3">
                <span className="text-xs font-medium">Total Compute</span>
                <Cpu className="size-4 text-purple-500" />
              </div>
              <div className="text-2xl font-extrabold text-foreground font-mono">
                {stats ? `${stats.total_compute_sec.toFixed(2)}s` : "0.00s"}
              </div>
              <p className="mt-1 text-[11px] text-muted-foreground">Active vCPU execution time</p>
            </div>
          </div>

          {/* Monthly Quota Meter Card */}
          <div className="rounded-xl border border-border bg-card p-6 shadow-xs">
            <div className="flex items-center justify-between mb-4">
              <div>
                <h3 className="text-sm font-bold text-foreground">Monthly MicroVM Quota</h3>
                <p className="text-xs text-muted-foreground mt-0.5">
                  Plan tier: <span className="font-semibold text-primary uppercase">{user.plan}</span>
                </p>
              </div>
              <div className="text-right">
                <span className="text-base font-extrabold font-mono text-foreground">
                  {stats?.monthly_runs_used ?? 0}
                </span>
                <span className="text-xs text-muted-foreground font-mono">
                  {" "}/ {stats?.monthly_runs_limit ?? 1000} runs
                </span>
              </div>
            </div>

            {/* Progress bar */}
            <div className="h-3 w-full rounded-full bg-muted/60 overflow-hidden">
              <div
                className="h-full rounded-full bg-gradient-to-r from-emerald-500 to-primary transition-all duration-500"
                style={{ width: `${quotaPercent}%` }}
              />
            </div>
            <div className="mt-2 flex justify-between text-[11px] text-muted-foreground font-mono">
              <span>{quotaPercent}% consumed</span>
              <span>Resets on the 1st of every month</span>
            </div>
          </div>

          {/* Architectural Advantages */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <div className="rounded-xl border border-border bg-card p-4">
              <div className="flex items-center gap-2 mb-2 text-primary font-bold text-xs">
                <Shield className="size-4" />
                <span>Firecracker Isolation</span>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Every code submission executes inside a dedicated Linux KVM microVM with isolated network namespaces, cgroups v2 RAM ceilings, and non-root PID 1 init.
              </p>
            </div>

            <div className="rounded-xl border border-border bg-card p-4">
              <div className="flex items-center gap-2 mb-2 text-primary font-bold text-xs">
                <Layers className="size-4" />
                <span>Compile-Once, Run-N-Times</span>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Compiled languages (C, C++, Go, Java) are compiled once into memory or ephemeral tmpfs. All N test cases execute sequentially on that exact artifact without recompiling.
              </p>
            </div>

            <div className="rounded-xl border border-border bg-card p-4">
              <div className="flex items-center gap-2 mb-2 text-primary font-bold text-xs">
                <Zap className="size-4" />
                <span>Sub-3ms Queue Dispatch</span>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed">
                Submissions are validated in-memory via Redis SHA-256 key cache and placed in an in-memory execution queue in under 3 milliseconds before workers consume them.
              </p>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
