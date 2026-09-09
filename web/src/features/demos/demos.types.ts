export interface DemoMeta {
  id: string
  name: string
  summary: string
  teaches: string
  expect: string
  tags: string[]
  estimated_seconds: number
  requires_vulnerable_mode: boolean
}

export interface DemoStep {
  seq: number
  elapsed_ms: number
  client_ip: string
  spoofed: boolean
  method: string
  path: string
  status: number
  blocked: boolean
  retry_after_sec: number
  duration_ms: number
  note?: string
  error?: string
}

interface DemoSummary {
  sent: number
  allowed: number
  blocked: number
  errors: number
  distinct_ips: number
  duration_ms: number
  verdict: string
}

export interface DemoResult {
  scenario_id: string
  started_at: string
  finished_at: string
  client_ip_mode: string
  rate_limit: number
  rate_window: string
  steps: DemoStep[]
  summary: DemoSummary
  teaches: string
  error?: string
}
