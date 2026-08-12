import type { Account } from '@/features/accounts/accounts.types'
import type { Transaction } from '@/features/transactions/transactions.types'
import type { DemoMeta, DemoResult } from '@/features/demos/demos.types'
import type { ClientIPMode, OpsConfig, SecurityEvent } from '@/features/security/security.types'

/** Written by cmd/record. See web/public/replay/. */
interface ReplayIndex {
  recorded_at: string
  modes: ClientIPMode[]
  /** Per-process fields are stripped by the recorder; the transport fills them in. */
  config: Omit<
    OpsConfig,
    'your_ip' | 'remote_addr' | 'mutable' | 'stream_subscribers' | 'dropped_events' | 'failed_events'
  >
  demos: DemoMeta[]
  accounts: Account[]
  transactions: Transaction[]
}

interface ReplayFixture {
  scenario_id: string
  client_ip_mode: ClientIPMode
  recorded_at: string
  result: DemoResult
  /** The real security_events rows the run produced, oldest first. */
  events: SecurityEvent[]
}

// BASE_URL is '/' locally and '/go-ledger/' on Pages, so fixtures resolve
// correctly under a project-page path without any extra configuration.
const ROOT = `${import.meta.env.BASE_URL}replay/`

let indexPromise: Promise<ReplayIndex> | null = null
const fixtureCache = new Map<string, Promise<ReplayFixture>>()

async function loadJSON<T>(name: string): Promise<T> {
  const res = await fetch(ROOT + name)
  if (!res.ok) {
    throw new Error(`missing replay fixture ${name}, run \`npm run record\``)
  }
  return (await res.json()) as T
}

export function loadIndex(): Promise<ReplayIndex> {
  indexPromise ??= loadJSON<ReplayIndex>('index.json')
  return indexPromise
}

/** Fixtures are fetched per scenario so the initial bundle stays small. */
export function loadFixture(scenarioId: string, mode: ClientIPMode): Promise<ReplayFixture> {
  const name = `${scenarioId}.${mode}.json`

  let cached = fixtureCache.get(name)
  if (!cached) {
    cached = loadJSON<ReplayFixture>(name)
    fixtureCache.set(name, cached)
  }
  return cached
}
