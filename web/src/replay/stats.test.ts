import { describe, expect, it } from 'vitest'
import type { SecurityEvent } from '@/features/security/security.types'
import { computeStats } from './stats'

function event(partial: Partial<SecurityEvent>): SecurityEvent {
  return {
    id: 1,
    timestamp: '2026-08-11T10:00:00.000Z',
    ip_address: '192.0.2.1',
    action_type: 'GET /api/accounts',
    method: 'GET',
    path: '/api/accounts',
    flag_status: 'ALLOWED',
    blocked: false,
    ...partial,
  }
}

describe('computeStats', () => {
  it('returns zeroed totals for no events', () => {
    // The dashboard reads totals.BLOCKED.toLocaleString() unconditionally, so
    // both keys must exist even with nothing recorded.
    const stats = computeStats([])
    expect(stats.totals).toEqual({ ALLOWED: 0, BLOCKED: 0 })
    expect(stats.distinct_ips).toBe(0)
    expect(stats.top_ips).toEqual([])
    expect(stats.buckets).toEqual([])
  })

  it('splits totals on the blocked flag', () => {
    const stats = computeStats([
      event({ blocked: false }),
      event({ blocked: true }),
      event({ blocked: true }),
    ])
    expect(stats.totals).toEqual({ ALLOWED: 1, BLOCKED: 2 })
  })

  it('counts distinct source IPs', () => {
    const stats = computeStats([
      event({ ip_address: '192.0.2.1' }),
      event({ ip_address: '192.0.2.1' }),
      event({ ip_address: '203.0.113.9' }),
    ])
    expect(stats.distinct_ips).toBe(2)
  })

  it('orders top IPs by total descending', () => {
    const stats = computeStats([
      event({ ip_address: 'a' }),
      event({ ip_address: 'b' }),
      event({ ip_address: 'b' }),
      event({ ip_address: 'c' }),
      event({ ip_address: 'c' }),
      event({ ip_address: 'c' }),
    ])
    expect(stats.top_ips.map((i) => i.ip_address)).toEqual(['c', 'b', 'a'])
    expect(stats.top_ips[0]).toEqual({ ip_address: 'c', total: 3, blocked: 0 })
  })

  it('breaks ties on ip_address ascending, matching the SQL ORDER BY', () => {
    // internal/ops/console.go orders by `total DESC, ip_address`. Without the
    // second key the order would follow Map insertion and differ from the server.
    const stats = computeStats([
      event({ ip_address: '203.0.113.9' }),
      event({ ip_address: '192.0.2.1' }),
    ])
    expect(stats.top_ips.map((i) => i.ip_address)).toEqual(['192.0.2.1', '203.0.113.9'])
  })

  it('caps top IPs at ten', () => {
    const events = Array.from({ length: 25 }, (_, i) => event({ ip_address: `10.0.0.${i}` }))
    expect(computeStats(events).top_ips).toHaveLength(10)
  })

  it('counts blocked per IP separately from the total', () => {
    const stats = computeStats([
      event({ ip_address: 'x', blocked: false }),
      event({ ip_address: 'x', blocked: true }),
      event({ ip_address: 'x', blocked: true }),
    ])
    expect(stats.top_ips[0]).toEqual({ ip_address: 'x', total: 3, blocked: 2 })
  })

  it('truncates buckets to the minute, matching date_trunc', () => {
    const stats = computeStats([
      event({ timestamp: '2026-08-11T10:00:05.000Z' }),
      event({ timestamp: '2026-08-11T10:00:59.999Z' }),
      event({ timestamp: '2026-08-11T10:01:00.000Z', blocked: true }),
    ])

    expect(stats.buckets).toHaveLength(2)
    expect(stats.buckets[0]).toEqual({
      bucket: '2026-08-11T10:00:00.000Z',
      allowed: 2,
      blocked: 0,
    })
    expect(stats.buckets[1]).toEqual({
      bucket: '2026-08-11T10:01:00.000Z',
      allowed: 0,
      blocked: 1,
    })
  })

  it('sorts buckets ascending so the chart reads left to right', () => {
    const stats = computeStats([
      event({ timestamp: '2026-08-11T10:05:00.000Z' }),
      event({ timestamp: '2026-08-11T10:01:00.000Z' }),
      event({ timestamp: '2026-08-11T10:03:00.000Z' }),
    ])
    expect(stats.buckets.map((b) => b.bucket)).toEqual([
      '2026-08-11T10:01:00.000Z',
      '2026-08-11T10:03:00.000Z',
      '2026-08-11T10:05:00.000Z',
    ])
  })

  it('describes itself as a recorded run rather than a time window', () => {
    // The live server reports the requested window; replay has no window, and
    // the dashboard prints this string verbatim.
    expect(computeStats([]).window).toBe('recorded run')
  })
})
