import React from "react"
import {
  Code2,
  Key,
  History,
  BarChart3,
  BookOpen,
  Sun,
  Moon,
  User as UserIcon,
  LogOut,
  Zap,
} from "lucide-react"
import type { User } from "../types"
import { useTheme } from "./theme-provider"

interface NavbarProps {
  activeTab: string
  setActiveTab: (tab: string) => void
  user: User | null
  onOpenAuth: () => void
  onLogout: () => void
}

export const Navbar: React.FC<NavbarProps> = ({
  activeTab,
  setActiveTab,
  user,
  onOpenAuth,
  onLogout,
}) => {
  const { theme, setTheme } = useTheme()

  const tabs = [
    { id: "playground", label: "Playground", icon: Code2 },
    { id: "keys", label: "API Keys", icon: Key },
    { id: "submissions", label: "Submissions", icon: History },
    { id: "analytics", label: "Analytics", icon: BarChart3 },
    { id: "quickstart", label: "Docs / API", icon: BookOpen },
  ]

  return (
    <header className="sticky top-0 z-40 w-full border-b border-border/60 bg-background/95 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6">
        {/* Brand / Logo */}
        <div className="flex items-center gap-6">
          <div
            onClick={() => setActiveTab("playground")}
            className="flex cursor-pointer items-center gap-2.5 font-bold tracking-tight text-foreground transition-opacity hover:opacity-90"
          >
            <div className="flex size-9 items-center justify-center rounded-lg bg-gradient-to-tr from-amber-500 to-orange-500 text-white shadow-md shadow-orange-500/20">
              <Zap className="size-5 fill-current" />
            </div>
            <div className="flex flex-col">
              <span className="text-base font-extrabold tracking-tight">CodeRunner</span>
              <span className="text-[10px] font-medium tracking-wider text-muted-foreground uppercase">
                MicroVM Cloud
              </span>
            </div>
          </div>

          {/* Navigation Tabs */}
          <nav className="hidden md:flex items-center gap-1 rounded-lg border border-border/50 bg-muted/40 p-1 text-sm font-medium">
            {tabs.map((tab) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.id
              return (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id)}
                  className={`flex items-center gap-2 rounded-md px-3 py-1.5 transition-all ${
                    isActive
                      ? "bg-background text-foreground shadow-xs"
                      : "text-muted-foreground hover:bg-background/50 hover:text-foreground"
                  }`}
                >
                  <Icon className="size-4" />
                  <span>{tab.label}</span>
                </button>
              )
            })}
          </nav>
        </div>

        {/* Right side controls */}
        <div className="flex items-center gap-3">
          {/* MicroVM Engine indicator */}
          <div className="hidden sm:flex items-center gap-1.5 rounded-full border border-emerald-500/20 bg-emerald-500/10 px-2.5 py-1 text-xs font-medium text-emerald-600 dark:text-emerald-400">
            <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
            <span>Engine Ready</span>
          </div>

          {/* Theme Switcher */}
          <button
            onClick={() => setTheme(theme === "dark" ? "light" : "dark")}
            className="flex size-9 items-center justify-center rounded-md border border-border text-muted-foreground hover:bg-muted hover:text-foreground"
            title="Toggle theme (Press 'd')"
          >
            {theme === "dark" ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </button>

          {/* User Auth Profile */}
          {user ? (
            <div className="flex items-center gap-2">
              <div className="flex items-center gap-2 rounded-md border border-border bg-muted/30 px-3 py-1.5 text-xs">
                <UserIcon className="size-3.5 text-muted-foreground" />
                <span className="font-medium text-foreground">{user.email}</span>
                <span className="rounded bg-primary/10 px-1.5 py-0.5 font-mono text-[10px] font-semibold text-primary uppercase">
                  {user.plan}
                </span>
              </div>
              <button
                onClick={onLogout}
                className="flex size-8 items-center justify-center rounded-md border border-border text-muted-foreground hover:bg-destructive/10 hover:text-destructive"
                title="Log out"
              >
                <LogOut className="size-3.5" />
              </button>
            </div>
          ) : (
            <button
              onClick={onOpenAuth}
              className="rounded-md bg-primary px-3.5 py-1.5 text-xs font-semibold text-primary-foreground shadow-xs hover:bg-primary/90 transition-colors"
            >
              Sign In
            </button>
          )}
        </div>
      </div>
    </header>
  )
}
