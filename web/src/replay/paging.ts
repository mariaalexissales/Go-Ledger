import type { ListResponse } from '@/lib/http/client'

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
