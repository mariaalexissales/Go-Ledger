import type { SecurityEvent, SecurityStats } from '@/features/security/security.types'

export function computeStats(events: readonly SecurityEvent[]): SecurityStats {
  const totals = { ALLOWED: 0, BLOCKED: 0 }
  const perIp = new Map<string, { total: number; blocked: number }>()
  const perBucket = new Map<string, { allowed: number; blocked: number }>()

  for (const event of events) {
    if (event.blocked) totals.BLOCKED++
    else totals.ALLOWED++

    const ip = perIp.get(event.ip_address) ?? { total: 0, blocked: 0 }
    ip.total++
    if (event.blocked) ip.blocked++
    perIp.set(event.ip_address, ip)

    const minute = new Date(event.timestamp)
    minute.setSeconds(0, 0)
    const key = minute.toISOString()

    const bucket = perBucket.get(key) ?? { allowed: 0, blocked: 0 }
    if (event.blocked) bucket.blocked++
    else bucket.allowed++
    perBucket.set(key, bucket)
  }

  const top_ips = [...perIp.entries()]
    .map(([ip_address, counts]) => ({ ip_address, ...counts }))
    .sort((a, b) => b.total - a.total || a.ip_address.localeCompare(b.ip_address))
    .slice(0, 10)

  const buckets = [...perBucket.entries()]
    .map(([bucket, counts]) => ({ bucket, ...counts }))
    .sort((a, b) => a.bucket.localeCompare(b.bucket))

  return {
    window: 'recorded run',
    totals,
    distinct_ips: perIp.size,
    top_ips,
    buckets,
    blocked_now: [],
  }
}
