export type Language =
  | "python"
  | "javascript"
  | "typescript"
  | "c"
  | "cpp"
  | "go"
  | "java"

export interface TestCase {
  id: string
  input: string
  expected_output: string
}

export interface TestResult {
  test_case_id: string
  passed: boolean
  actual_output: string
  expected_output: string
  time_taken_ms: number
}

export interface ExecutionResult {
  stdout: string
  stderr: string
  exit_code: number
  time_taken_ms: number
  memory_used_kb: number
  compilation_time_ms?: number
  test_results?: TestResult[]
}

export type JobStatus =
  | "QUEUED"
  | "RUNNING"
  | "COMPLETED"
  | "FAILED"
  | "COMPILE_ERROR"
  | "TIMED_OUT"

export interface Job {
  id: string
  user_id?: string
  language: string
  code: string
  test_cases?: TestCase[]
  time_limit_ms: number
  memory_limit_mb: number
  status: JobStatus
  error?: string
  result?: ExecutionResult
  created_at: string
  started_at?: string
  completed_at?: string
}

export interface UserProfile {
  id: string
  username: string
  display_name: string
  bio: string
  avatar_url: string
  default_language: Language | string
  editor_theme: string
  company: string
  github_url: string
}

export interface SessionInfo {
  provider: string
  ip_address: string
  user_agent: string
  device_name: string
  expires_at: string
  last_rotated_at: string
  last_active_at: string
  created_at: string
}

export interface User {
  id: string
  email: string
  role: "admin" | "member"
  plan: "free" | "pro" | "enterprise"
  profile?: UserProfile
  created_at: string
}

export interface ApiKeyItem {
  id: string
  name: string
  key_prefix: string
  is_active: boolean
  rate_limit_rpm: number
  max_concurrency: number
  created_at: string
  revoked_at?: string
}

export interface CreatedApiKeyResponse {
  id: string
  name: string
  key_prefix: string
  raw_key: string
  rate_limit_rpm: number
  max_concurrency: number
  created_at: string
}

export interface SubmissionSummary {
  id: string
  language: string
  status: JobStatus
  exit_code: number
  time_taken_ms: number
  memory_used_kb: number
  total_test_cases: number
  passed_test_cases: number
  created_at: string
  completed_at?: string
}

export interface SubmissionDetail {
  id: string
  user_id: string
  api_key_id?: string
  language: string
  code: string
  status: JobStatus
  time_limit_ms: number
  memory_limit_mb: number
  exit_code: number
  stdout: string
  stderr: string
  error?: string
  time_taken_ms: number
  memory_used_kb: number
  test_results?: TestResult[]
  created_at: string
  started_at?: string
  completed_at?: string
}

export interface DashboardStats {
  total_submissions: number
  successful_runs: number
  failed_runs: number
  timeout_runs: number
  success_rate: number
  avg_time_ms: number
  total_compute_sec: number
  monthly_runs_used: number
  monthly_runs_limit: number
}
