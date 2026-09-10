export interface SecurityEvent {
  id: number
  timestamp: string
  ip_address: string
  action_type: string
  method: string
  path: string
  flag_status: 'ALLOWED' | 'BLOCKED'
  blocked: boolean
}

export interface EventListParams {
  limit?: number
  offset?: number
  flag_status?: string
  ip_address?: string
  // Served by the Go handler and the replay transport; no UI sends it yet.
  action_type?: string
}

interface LimiterPolicy {
  limit: number
  window: string
  block_period: string
}

export type ClientIPMode = 'xff-trust-all' | 'remote-addr'

export interface OpsConfig {
  client_ip_mode: ClientIPMode
  rate_limit: LimiterPolicy
  your_ip: string
  remote_addr: string
  mutable: boolean
  stream_subscribers: number
  dropped_events: number
  failed_events: number
}

interface IpStat {
  ip_address: string
  total: number
  blocked: number
}

interface BucketStat {
  bucket: string
  allowed: number
  blocked: number
}

interface BlockedIp {
  ip_address: string
  until: string
}

export interface SecurityStats {
  window: string
  totals: { ALLOWED: number; BLOCKED: number }
  distinct_ips: number
  top_ips: IpStat[]
  buckets: BucketStat[]
  blocked_now: BlockedIp[]
}
