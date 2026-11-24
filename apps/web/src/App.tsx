import { useState, useEffect } from "react"
import { Navbar } from "./components/Navbar"
import { Playground } from "./components/Playground"
import { ApiKeys } from "./components/ApiKeys"
import { Submissions } from "./components/Submissions"
import { Analytics } from "./components/Analytics"
import { Quickstart } from "./components/Quickstart"
import { AuthModal } from "./components/AuthModal"
import type { User } from "./types"
import { api } from "./lib/api"

export function App() {
  const [activeTab, setActiveTab] = useState("playground")
  const [user, setUser] = useState<User | null>(null)
  const [authModalOpen, setAuthModalOpen] = useState(false)

  // Verify authenticated session on load
  useEffect(() => {
    api
      .getMe()
      .then((u) => setUser(u))
      .catch(() => {
        // Not logged in or expired session
        setUser(null)
      })
  }, [])

  const handleLogout = async () => {
    await api.logout()
    setUser(null)
  }

  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col font-sans antialiased selection:bg-primary/20 selection:text-primary">
      {/* Navbar Header */}
      <Navbar
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        user={user}
        onOpenAuth={() => setAuthModalOpen(true)}
        onLogout={handleLogout}
      />

      {/* Main Content Area */}
      <main className="flex-1 mx-auto w-full max-w-7xl px-4 sm:px-6 pt-6">
        {activeTab === "playground" && (
          <Playground user={user} onOpenAuth={() => setAuthModalOpen(true)} />
        )}
        {activeTab === "keys" && (
          <ApiKeys user={user} onOpenAuth={() => setAuthModalOpen(true)} />
        )}
        {activeTab === "submissions" && (
          <Submissions user={user} onOpenAuth={() => setAuthModalOpen(true)} />
        )}
        {activeTab === "analytics" && (
          <Analytics user={user} onOpenAuth={() => setAuthModalOpen(true)} />
        )}
        {activeTab === "quickstart" && <Quickstart />}
      </main>

      {/* Auth Modal (Login / Register) */}
      <AuthModal
        isOpen={authModalOpen}
        onClose={() => setAuthModalOpen(false)}
        onSuccess={(newUser) => setUser(newUser)}
      />
    </div>
  )
}
