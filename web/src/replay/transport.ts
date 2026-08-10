import { ApiError } from '@/lib/http/errors'
import type { ListResponse, QueryParams } from '@/lib/http/client'
import type { ClientIPMode, OpsConfig, SecurityEvent, SecurityStats } from '@/features/security/security.types'
import type { DemoResult } from '@/features/demos/demos.types'
import { loadFixture, loadIndex } from './fixtures'
import { replayBus } from './bus'
import { replayLedger } from './ledger'

/**
 * The replay transport mirrors the Go router's route table and returns the same
 * JSON shapes. Everything above it stays identical in both builds: the feature
 * api modules, the query hooks and the components.
 */

let mode: ClientIPMode | null = null
let replayedEvents: SecurityEvent[] = []

type Params = Record<string, string | number | boolean | undefined | null>

export async function replayRequest<T>(
  method: string,
  path: string,
  params?: QueryParams,
  body?: unknown,
): Promise<T> {
  const query = (params ?? {}) as Params
  return (await route(method, path, query, body)) as T
}

async function route(method: string, path: string, query: Params, body: unknown): Promise<unknown> {

  if (path === '/api/accounts' && method === 'GET') {
    const rows = await replayLedger.listAccounts(str(query.q))
    return paginate(rows, query, 25, 100)
  }

  if (path === '/api/accounts' && method === 'POST') {
    const name = str((body as { name?: string } | undefined)?.name)
    if (!name.trim()) throw new ApiError(400, 'name is required')
    return replayLedger.createAccount(name)
  }

  const accountTransactions = /^\/api\/accounts\/(\d+)\/transactions$/.exec(path)
  if (accountTransactions && method === 'GET') {
    const rows = await replayLedger.listTransactions(Number(accountTransactions[1]))
    return paginate(rows, query, 25, 100)
  }

  const account = /^\/api\/accounts\/(\d+)$/.exec(path)
  if (account) {
    const id = Number(account[1])

    if (method === 'GET') {
      const found = await replayLedger.getAccount(id)
      if (!found) throw new ApiError(404, 'account not found')
      return found
    }

    if (method === 'DELETE') {
      if (!(await replayLedger.deleteAccount(id))) throw new ApiError(404, 'account not found')
      return undefined
    }
  }

  if (path === '/api/transactions' && method === 'GET') {
    const accountId = query.account_id === undefined ? undefined : Number(query.account_id)
    const rows = await replayLedger.listTransactions(accountId)
    return paginate(rows, query, 25, 100)
  }

  if (path === '/api/transactions' && method === 'POST') {
    const payload = body as { account_id?: number; amount?: number } | undefined
    const created = await replayLedger.createTransaction(
      Number(payload?.account_id),
      Number(payload?.amount),
    )
    if (!created) throw new ApiError(404, 'account not found')
    return created
  }

  const transaction = /^\/api\/transactions\/(\d+)$/.exec(path)
  if (transaction) {
    const id = Number(transaction[1])

    if (method === 'GET') {
      const found = await replayLedger.getTransaction(id)
      if (!found) throw new ApiError(404, 'transaction not found')
      return found
    }

    if (method === 'DELETE') {
      if (!(await replayLedger.deleteTransaction(id))) {
        throw new ApiError(404, 'transaction not found')
      }
      return undefined
    }
  }

  if (path === '/ops/config' && method === 'GET') return currentConfig()

  if (path === '/ops/config/client-ip-mode' && method === 'PUT') {
    const next = (body as { mode?: ClientIPMode } | undefined)?.mode
    if (next !== 'xff-trust-all' && next !== 'remote-addr') {
      throw new ApiError(400, `unknown client IP mode "${String(next)}"`)
    }

    mode = next
    reset()
    return currentConfig()
  }

  if (path === '/ops/config/limiter-policy' && method === 'PUT') {
    throw new ApiError(
      409,
      'The limiter policy is fixed in the recorded demo. Clone the repo and run it locally to change it.',
    )
  }

  if (path === '/ops/events' && method === 'GET') {
    let rows = replayedEvents
    const flag = str(query.flag_status)
    if (flag) rows = rows.filter((event) => event.flag_status === flag)

    const ips = str(query.ip_address)
      .split(',')
      .map((ip) => ip.trim())
      .filter(Boolean)
    if (ips.length > 0) rows = rows.filter((event) => ips.includes(event.ip_address))

    const action = str(query.action_type).toLowerCase()
    if (action) {
      rows = rows.filter((event) => event.action_type.toLowerCase().includes(action))
    }

    // Newest first, matching the server's ORDER BY.
    return paginate([...rows].reverse(), query, 50, 500)
  }

  if (path === '/ops/events/reset' && method === 'POST') {
    reset()
    return undefined
  }

  if (path === '/ops/stats' && method === 'GET') return computeStats()

  if (path === '/ops/demos' && method === 'GET') {
    const index = await loadIndex()
    return list(index.demos, index.demos.length, index.demos.length, 0)
  }

  const demoRun = /^\/ops\/demos\/([\w-]+)\/run$/.exec(path)
  if (demoRun && method === 'POST') {
    return runScenario(demoRun[1])
  }

  throw new ApiError(404, 'not found')
}

/**
 * Replays a recorded run. Events go out on the bus at their recorded cadence
 * and the promise resolves only once the last one lands, so the progress bar
 * and pending state behave exactly as they do against the live server.
 */
async function runScenario(scenarioId: string): Promise<DemoResult> {
  const fixture = await loadFixture(scenarioId, await currentMode())

  reset()

  const pacing = await replayBus.schedule(fixture.events, (event) => {
    replayedEvents.push(event)
  })

  return {
    ...fixture.result,
    summary: {
      ...fixture.result.summary,
      // Report the recorded duration rather than the stretched replay window.
      // The recorded one is the real measurement.
      duration_ms: pacing.recordedMs || fixture.result.summary.duration_ms,
    },
  }
}

function reset() {
  replayBus.cancel()
  replayedEvents = []
}

async function currentMode(): Promise<ClientIPMode> {
  mode ??= (await loadIndex()).config.client_ip_mode
  return mode
}

async function currentConfig(): Promise<OpsConfig> {
  const index = await loadIndex()

  return {
    ...index.config,
    client_ip_mode: await currentMode(),
    // Per-process facts the recorder strips, since they mean nothing in a
    // recording. `mutable` is true because both modes were recorded, so the
    // vulnerable/hardened toggle genuinely works here.
    mutable: true,
    your_ip: 'recorded',
    remote_addr: 'recorded',
    stream_subscribers: 1,
    dropped_events: 0,
    failed_events: 0,
  }
}

/** Derived from what has replayed so far, so the tiles move during a run. */
function computeStats(): SecurityStats {
  const totals = { ALLOWED: 0, BLOCKED: 0 }
  const perIp = new Map<string, { total: number; blocked: number }>()
  const perBucket = new Map<string, { allowed: number; blocked: number }>()

  for (const event of replayedEvents) {
    if (event.blocked) totals.BLOCKED++
    else totals.ALLOWED++

    const ip = perIp.get(event.ip_address) ?? { total: 0, blocked: 0 }
    ip.total++
    if (event.blocked) ip.blocked++
    perIp.set(event.ip_address, ip)

    // Bucket by the recorded timestamp so the chart reflects the real run shape.
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

function paginate<T>(rows: T[], query: Params, fallback: number, max: number): ListResponse<T> {
  const limit = clampInt(query.limit, fallback, max)
  const offset = Math.max(0, Number(query.offset) || 0)

  return list(rows.slice(offset, offset + limit), rows.length, limit, offset)
}

function list<T>(data: T[], total: number, limit: number, offset: number): ListResponse<T> {
  return { data, total, limit, offset }
}

function clampInt(value: unknown, fallback: number, max: number): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 1) return fallback
  return Math.min(Math.trunc(parsed), max)
}

function str(value: unknown): string {
  return typeof value === 'string' ? value : ''
}
