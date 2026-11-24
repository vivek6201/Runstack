import React, { useState } from "react"
import { X, Lock, Mail, Loader2, Zap } from "lucide-react"
import { api } from "../lib/api"
import type { User } from "../types"

interface AuthModalProps {
  isOpen: boolean
  onClose: () => void
  onSuccess: (user: User) => void
}

export const AuthModal: React.FC<AuthModalProps> = ({ isOpen, onClose, onSuccess }) => {
  const [isLogin, setIsLogin] = useState(true)
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  if (!isOpen) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      if (isLogin) {
        const res = await api.login(email, password)
        onSuccess(res.user)
      } else {
        const res = await api.register(email, password)
        onSuccess(res.user)
      }
      onClose()
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message)
      } else {
        setError("Authentication failed")
      }
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-xs p-4">
      <div className="relative w-full max-w-md rounded-xl border border-border bg-card p-6 shadow-2xl animate-in fade-in zoom-in-95 duration-150">
        <button
          onClick={onClose}
          className="absolute right-4 top-4 text-muted-foreground hover:text-foreground"
        >
          <X className="size-5" />
        </button>

        <div className="mb-6 text-center">
          <div className="mx-auto mb-3 flex size-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <Zap className="size-6 fill-current" />
          </div>
          <h2 className="text-xl font-bold tracking-tight text-foreground">
            {isLogin ? "Welcome back" : "Create your account"}
          </h2>
          <p className="mt-1 text-xs text-muted-foreground">
            {isLogin
              ? "Sign in to access your API keys, playground, and submissions"
              : "Get instant access to fast, sandboxed microVM code execution"}
          </p>
        </div>

        {error && (
          <div className="mb-4 rounded-md border border-destructive/20 bg-destructive/10 p-3 text-xs text-destructive">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-xs font-medium text-foreground mb-1">Email Address</label>
            <div className="relative">
              <Mail className="absolute left-3 top-2.5 size-4 text-muted-foreground" />
              <input
                type="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@domain.com"
                className="w-full rounded-md border border-input bg-background py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground/60 focus:border-primary focus:outline-hidden focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-medium text-foreground mb-1">Password</label>
            <div className="relative">
              <Lock className="absolute left-3 top-2.5 size-4 text-muted-foreground" />
              <input
                type="password"
                required
                minLength={6}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
                className="w-full rounded-md border border-input bg-background py-2 pl-9 pr-3 text-sm text-foreground placeholder:text-muted-foreground/60 focus:border-primary focus:outline-hidden focus:ring-1 focus:ring-primary"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="flex w-full items-center justify-center gap-2 rounded-md bg-primary py-2.5 text-sm font-semibold text-primary-foreground shadow-xs hover:bg-primary/90 disabled:opacity-50 transition-colors"
          >
            {loading && <Loader2 className="size-4 animate-spin" />}
            <span>{isLogin ? "Sign In" : "Create Account"}</span>
          </button>
        </form>

        <div className="mt-6 text-center text-xs text-muted-foreground">
          {isLogin ? "Don't have an account yet?" : "Already have an account?"}{" "}
          <button
            type="button"
            onClick={() => {
              setIsLogin(!isLogin)
              setError(null)
            }}
            className="font-medium text-primary hover:underline"
          >
            {isLogin ? "Sign up" : "Sign in"}
          </button>
        </div>
      </div>
    </div>
  )
}
