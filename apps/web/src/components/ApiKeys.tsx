import React, { useState, useEffect } from "react"
import {
  Key,
  Plus,
  Copy,
  Check,
  AlertTriangle,
  Loader2,
  Trash2,
  ShieldCheck,
  X,
} from "lucide-react"
import type { ApiKeyItem, CreatedApiKeyResponse, User } from "../types"
import { api } from "../lib/api"

interface ApiKeysProps {
  user: User | null
  onOpenAuth: () => void
}

export const ApiKeys: React.FC<ApiKeysProps> = ({ user, onOpenAuth }) => {
  const [keys, setKeys] = useState<ApiKeyItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Creation modal
  const [modalOpen, setModalOpen] = useState(false)
  const [keyName, setKeyName] = useState("")
  const [rateLimitRpm, setRateLimitRpm] = useState<number>(60)
  const [maxConcurrency, setMaxConcurrency] = useState<number>(2)
  const [createLoading, setCreateLoading] = useState(false)
  const [createdKeyData, setCreatedKeyData] = useState<CreatedApiKeyResponse | null>(null)
  const [copiedKey, setCopiedKey] = useState(false)

  const fetchKeys = async () => {
    if (!user) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listKeys()
      setKeys(data)
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError("Failed to load API keys")
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchKeys()
  }, [user])

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault()
    setCreateLoading(true)
    setError(null)
    try {
      const res = await api.createKey(keyName, rateLimitRpm, maxConcurrency)
      setCreatedKeyData(res)
      setKeyName("")
      fetchKeys()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError("Failed to create API key")
      }
    } finally {
      setCreateLoading(false)
    }
  }

  const handleRevoke = async (id: string) => {
    if (!confirm("Are you sure you want to revoke this API key? All applications using it will be blocked immediately.")) {
      return
    }
    try {
      await api.revokeKey(id)
      fetchKeys()
    } catch (err: unknown) {
      alert(err instanceof Error ? err.message : "Failed to revoke key")
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopiedKey(true)
    setTimeout(() => setCopiedKey(false), 2000)
  }

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center rounded-2xl border border-dashed border-border p-12 text-center bg-card shadow-xs">
        <div className="mb-4 flex size-12 items-center justify-center rounded-full bg-primary/10 text-primary">
          <Key className="size-6" />
        </div>
        <h3 className="text-lg font-bold text-foreground">Authentication Required</h3>
        <p className="mt-1 max-w-sm text-xs text-muted-foreground mb-6">
          Sign in or create an account to generate production API keys and configure microVM concurrency.
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
      {/* Top Banner */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="text-xl font-bold tracking-tight text-foreground">API Credentials</h2>
          <p className="text-xs text-muted-foreground mt-0.5">
            Authenticate automated workloads and SDK clients against the CodeRunner microVM engine.
          </p>
        </div>

        <button
          onClick={() => {
            setCreatedKeyData(null)
            setModalOpen(true)
          }}
          className="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-xs font-semibold text-primary-foreground shadow-xs hover:bg-primary/90 transition-colors"
        >
          <Plus className="size-3.5" />
          <span>Generate New Key</span>
        </button>
      </div>

      {error && (
        <div className="rounded-lg border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
          {error}
        </div>
      )}

      {/* Keys Table Card */}
      <div className="rounded-xl border border-border bg-card shadow-sm overflow-hidden">
        <div className="border-b border-border bg-muted/40 px-4 py-3 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ShieldCheck className="size-4 text-emerald-500" />
            <span className="text-xs font-bold text-foreground">Active Keys ({keys.length})</span>
          </div>
          <span className="text-[11px] text-muted-foreground">
            Keys are SHA-256 hashed and cached in Redis with zero plaintext storage.
          </span>
        </div>

        {loading ? (
          <div className="flex h-36 items-center justify-center text-muted-foreground">
            <Loader2 className="size-6 animate-spin text-primary" />
          </div>
        ) : keys.length === 0 ? (
          <div className="flex h-36 flex-col items-center justify-center text-center text-muted-foreground">
            <Key className="size-6 mb-2 opacity-40" />
            <p className="text-xs">No API keys created yet.</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs">
              <thead className="border-b border-border bg-muted/20 text-muted-foreground font-semibold">
                <tr>
                  <th className="px-4 py-2.5">Name</th>
                  <th className="px-4 py-2.5">Prefix</th>
                  <th className="px-4 py-2.5">Rate Limit</th>
                  <th className="px-4 py-2.5">Concurrency</th>
                  <th className="px-4 py-2.5">Created</th>
                  <th className="px-4 py-2.5">Status</th>
                  <th className="px-4 py-2.5 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {keys.map((k) => (
                  <tr key={k.id} className="hover:bg-muted/30 transition-colors">
                    <td className="px-4 py-3 font-semibold text-foreground">{k.name}</td>
                    <td className="px-4 py-3 font-mono text-muted-foreground">
                      <span className="rounded bg-muted px-1.5 py-0.5">{k.key_prefix}...</span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{k.rate_limit_rpm} req/min</td>
                    <td className="px-4 py-3 text-muted-foreground">{k.max_concurrency} microVMs</td>
                    <td className="px-4 py-3 text-muted-foreground font-mono">
                      {new Date(k.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3">
                      {k.is_active ? (
                        <span className="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-semibold text-emerald-600 dark:text-emerald-400">
                          Active
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 rounded-full bg-muted px-2 py-0.5 text-[10px] font-semibold text-muted-foreground">
                          Revoked
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-right">
                      {k.is_active && (
                        <button
                          onClick={() => handleRevoke(k.id)}
                          className="rounded p-1 text-muted-foreground hover:bg-destructive/10 hover:text-destructive transition-colors"
                          title="Revoke key immediately"
                        >
                          <Trash2 className="size-3.5" />
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Creation Modal */}
      {modalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs p-4">
          <div className="relative w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl animate-in fade-in zoom-in-95 duration-150">
            <button
              onClick={() => setModalOpen(false)}
              className="absolute right-4 top-4 text-muted-foreground hover:text-foreground"
            >
              <X className="size-5" />
            </button>

            {!createdKeyData ? (
              <>
                <h3 className="text-base font-bold text-foreground mb-1">Generate API Key</h3>
                <p className="text-xs text-muted-foreground mb-4">
                  Create a new credential to authenticate with the CodeRunner execution engine.
                </p>

                <form onSubmit={handleCreate} className="space-y-4">
                  <div>
                    <label className="block text-xs font-medium text-foreground mb-1">
                      Key Name / Identifier
                    </label>
                    <input
                      type="text"
                      required
                      value={keyName}
                      onChange={(e) => setKeyName(e.target.value)}
                      placeholder="e.g. Production Cluster Backend"
                      className="w-full rounded-md border border-input bg-background px-3 py-2 text-xs text-foreground focus:border-primary focus:outline-hidden"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-foreground mb-1">
                        Rate Limit (RPM)
                      </label>
                      <input
                        type="number"
                        min={10}
                        max={10000}
                        value={rateLimitRpm}
                        onChange={(e) => setRateLimitRpm(Number(e.target.value))}
                        className="w-full rounded-md border border-input bg-background px-3 py-2 text-xs text-foreground focus:border-primary focus:outline-hidden"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-foreground mb-1">
                        Max Concurrency
                      </label>
                      <input
                        type="number"
                        min={1}
                        max={100}
                        value={maxConcurrency}
                        onChange={(e) => setMaxConcurrency(Number(e.target.value))}
                        className="w-full rounded-md border border-input bg-background px-3 py-2 text-xs text-foreground focus:border-primary focus:outline-hidden"
                      />
                    </div>
                  </div>

                  <button
                    type="submit"
                    disabled={createLoading}
                    className="flex w-full items-center justify-center gap-2 rounded-md bg-primary py-2 text-xs font-semibold text-primary-foreground shadow-xs hover:bg-primary/90 disabled:opacity-50"
                  >
                    {createLoading && <Loader2 className="size-3.5 animate-spin" />}
                    <span>Create API Key</span>
                  </button>
                </form>
              </>
            ) : (
              <div className="space-y-4">
                <div className="flex items-center gap-2 text-emerald-500">
                  <ShieldCheck className="size-5" />
                  <h3 className="text-base font-bold text-foreground">API Key Generated</h3>
                </div>

                <div className="rounded-md border border-amber-500/20 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-400 flex items-start gap-2">
                  <AlertTriangle className="size-4 shrink-0 mt-0.5" />
                  <span>
                    Copy this key now. For security purposes, you will <strong>never</strong> be able to see it again.
                  </span>
                </div>

                <div>
                  <label className="block text-xs font-medium text-muted-foreground mb-1">Secret Key</label>
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      readOnly
                      value={createdKeyData.raw_key}
                      className="flex-1 rounded-md border border-input bg-muted/40 px-3 py-2 font-mono text-xs text-foreground select-all focus:outline-hidden"
                    />
                    <button
                      onClick={() => copyToClipboard(createdKeyData.raw_key)}
                      className="flex items-center gap-1 rounded-md bg-primary px-3 py-2 text-xs font-semibold text-primary-foreground hover:bg-primary/90"
                    >
                      {copiedKey ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
                      <span>{copiedKey ? "Copied" : "Copy"}</span>
                    </button>
                  </div>
                </div>

                <button
                  onClick={() => setModalOpen(false)}
                  className="w-full rounded-md border border-border py-2 text-xs font-semibold text-foreground hover:bg-muted"
                >
                  Done
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
