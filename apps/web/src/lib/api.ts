import type {
  ApiKeyItem,
  CreatedApiKeyResponse,
  DashboardStats,
  Job,
  SubmissionDetail,
  SubmissionSummary,
  TestCase,
  User,
} from "../types"

const API_BASE = import.meta.env.VITE_API_BASE_URL || "/api"

async function request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {})
  headers.set("Content-Type", "application/json")

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
    credentials: "include",
  })

  let res: any
  try {
    res = await response.json()
  } catch {
    res = null
  }

  if (!response.ok) {
    let errorMsg = `HTTP Error ${response.status}`
    if (res && typeof res === "object" && res.error) {
      errorMsg = res.error
    }
    throw new Error(errorMsg)
  }

  // Handle standard envelope { request_id, success, data, error }
  if (res && typeof res === "object" && "success" in res) {
    if (!res.success) {
      throw new Error(res.error || `HTTP Error ${response.status}`)
    }
    return (res.data !== undefined ? res.data : res) as T
  }

  return res as T
}

export const api = {
  // Authentication (relies on HttpOnly SameSite cookie via credentials: "include")
  async register(email: string, password: string): Promise<{ user: User }> {
    return request<{ user: User }>("/auth/register", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    })
  },

  async login(email: string, password: string): Promise<{ user: User }> {
    return request<{ user: User }>("/auth/login", {
      method: "POST",
      body: JSON.stringify({ email, password }),
    })
  },

  async logout(): Promise<void> {
    try {
      await request("/auth/logout", { method: "POST" })
    } finally {
      // Purge any legacy token from earlier builds
      localStorage.removeItem("token")
    }
  },

  async getMe(): Promise<User> {
    return request<User>("/auth/me")
  },

  // API Keys
  async listKeys(): Promise<ApiKeyItem[]> {
    return request<ApiKeyItem[]>("/keys")
  },

  async createKey(name: string, rateLimitRpm?: number, maxConcurrency?: number): Promise<CreatedApiKeyResponse> {
    return request<CreatedApiKeyResponse>("/keys", {
      method: "POST",
      body: JSON.stringify({
        name,
        rate_limit_rpm: rateLimitRpm,
        max_concurrency: maxConcurrency,
      }),
    })
  },

  async revokeKey(id: string): Promise<void> {
    return request<void>(`/keys/${id}`, { method: "DELETE" })
  },

  // Submissions
  async listSubmissions(params: { page?: number; limit?: number; status?: string; language?: string } = {}): Promise<{
    items: SubmissionSummary[]
    total: number
    page: number
    limit: number
  }> {
    const q = new URLSearchParams()
    if (params.page) q.set("page", String(params.page))
    if (params.limit) q.set("limit", String(params.limit))
    if (params.status) q.set("status", params.status)
    if (params.language) q.set("language", params.language)
    return request(`/submissions?${q.toString()}`)
  },

  async getSubmission(id: string): Promise<SubmissionDetail> {
    return request<SubmissionDetail>(`/submissions/${id}`)
  },

  // Stats
  async getStats(): Promise<DashboardStats> {
    return request<DashboardStats>("/stats")
  },

  // Playground Execution
  async executeCode(payload: {
    language: string
    code: string
    test_cases: TestCase[]
    time_limit_ms: number
    memory_limit_mb: number
  }): Promise<Job> {
    return request<Job>("/playground/execute", {
      method: "POST",
      body: JSON.stringify(payload),
    })
  },
}
