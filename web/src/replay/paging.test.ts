import { describe, expect, it } from 'vitest'
import { EVENTS_PAGE, LEDGER_PAGE, clampInt, paginate } from './paging'

describe('clampInt', () => {
  it('returns the fallback when the value is absent or unparseable', () => {
    expect(clampInt(undefined, 25, 100)).toBe(25)
    expect(clampInt(null, 25, 100)).toBe(25)
    expect(clampInt('', 25, 100)).toBe(25)
    expect(clampInt('abc', 25, 100)).toBe(25)
    expect(clampInt(NaN, 25, 100)).toBe(25)
    expect(clampInt(Infinity, 25, 100)).toBe(25)
  })

  it('rejects zero and negatives in favour of the fallback', () => {
    expect(clampInt(0, 25, 100)).toBe(25)
    expect(clampInt(-5, 25, 100)).toBe(25)
  })

  it('clamps to the max instead of erroring', () => {
    expect(clampInt(5000, 25, 100)).toBe(100)
    expect(clampInt(100, 25, 100)).toBe(100)
    expect(clampInt(99, 25, 100)).toBe(99)
  })

  it('truncates fractions rather than rounding', () => {
    expect(clampInt(10.9, 25, 100)).toBe(10)
  })

  it('accepts numeric strings, since query values arrive as strings', () => {
    expect(clampInt('10', 25, 100)).toBe(10)
  })
})

describe('paginate', () => {
  const rows = Array.from({ length: 30 }, (_, i) => i)

  it('reports the unpaginated total alongside the page', () => {
    const page = paginate(rows, { limit: 10 }, 25, 100)
    expect(page.data).toEqual([0, 1, 2, 3, 4, 5, 6, 7, 8, 9])
    expect(page.total).toBe(30)
    expect(page.limit).toBe(10)
    expect(page.offset).toBe(0)
  })

  it('applies the offset', () => {
    const page = paginate(rows, { limit: 5, offset: 25 }, 25, 100)
    expect(page.data).toEqual([25, 26, 27, 28, 29])
    expect(page.offset).toBe(25)
  })

  it('returns an empty page past the end rather than throwing', () => {
    const page = paginate(rows, { limit: 10, offset: 999 }, 25, 100)
    expect(page.data).toEqual([])
    expect(page.total).toBe(30)
  })

  it('treats a negative or unparseable offset as the first page', () => {
    expect(paginate(rows, { offset: -10 }, 25, 100).offset).toBe(0)
    expect(paginate(rows, { offset: 'soon' }, 25, 100).offset).toBe(0)
  })

  it('falls back to the default limit with no query', () => {
    const page = paginate(rows, {}, 25, 100)
    expect(page.limit).toBe(25)
    expect(page.data).toHaveLength(25)
  })
})

describe('the constants mirroring the Go server', () => {
  it('matches internal/api/params.go for ledger endpoints', () => {
    expect(LEDGER_PAGE).toEqual({ fallback: 25, max: 100 })
  })

  it('matches internal/ops/console.go for the event list', () => {
    expect(EVENTS_PAGE).toEqual({ fallback: 50, max: 500 })
  })
})
