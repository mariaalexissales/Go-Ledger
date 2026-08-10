/** Mirrors demo.Meta. */
export interface DemoMeta {
  id: string
  name: string
  summary: string
  /** The point of the scenario, in plain English. */
  teaches: string
  /** What to watch for while it runs. */
  expect: string
  tags: string[]
  estimated_seconds: number
  requires_vulnerable_mode: boolean
}

/** Mirrors demo.Step. A step with no method is an annotation, not a request. */
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

/** Mirrors demo.Summary. */
export interface DemoSummary {
  sent: number
  allowed: number
  blocked: number
  errors: number
  distinct_ips: number
  duration_ms: number
  verdict: string
}

/** Mirrors demo.Result. */
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
