import type { ListResponse } from '@/lib/http/client'

/**
 * Pagination for the replay transport, kept in its own module because it
 * duplicates the Go server and that duplication needs somewhere to be tested.
 *
 * These constants mirror two places on the server. Nothing enforces the match,
 * so if either side changes, the replay build silently paginates differently
 * from the live one:
 *
 *   LEDGER  internal/api/params.go        defaultLimit 25, maxLimit 100
 *   EVENTS  internal/ops/console.go       50, 500, inline in listEvents
 *
 * Behaviour also mirrors parsePageParams deliberately: a malformed or
 * out-of-range value falls back to the default rather than erroring, because a
 * list endpoint returning 400 for a stray query param is more annoying than
 * useful.
 */
export const LEDGER_PAGE = { fallback: 25, max: 100 } as const
export const EVENTS_PAGE = { fallback: 50, max: 500 } as const

type PageQuery = { limit?: unknown; offset?: unknown }

export function clampInt(value: unknown, fallback: number, max: number): number {
  const parsed = Number(value)
  if (!Number.isFinite(parsed) || parsed < 1) return fallback
  return Math.min(Math.trunc(parsed), max)
}

export function paginate<T>(
  rows: readonly T[],
  query: PageQuery,
  fallback: number,
  max: number,
): ListResponse<T> {
  const limit = clampInt(query.limit, fallback, max)
  const offset = Math.max(0, Number(query.offset) || 0)

  return list(rows.slice(offset, offset + limit), rows.length, limit, offset)
}

export function list<T>(data: T[], total: number, limit: number, offset: number): ListResponse<T> {
  return { data, total, limit, offset }
}
