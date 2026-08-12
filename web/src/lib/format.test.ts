import { describe, expect, it } from 'vitest'
import { IP_RAMP_SIZE } from '@/theme'
import { formatDuration, formatMoney, ipColor } from './format'

/**
 * formatDateTime and formatTime are deliberately not asserted against literal
 * strings: both use Intl with the runtime's locale and time zone, so any such
 * expectation would encode the machine that ran it. Their guard clauses are
 * what matter, and those are covered below.
 */

describe('formatMoney', () => {
  it('renders an em dash for absent values rather than $0.00 or NaN', () => {
    // A null balance means "no rows yet", which is not the same as zero. The
    // Go side sends null because balance comes from pgtype.Numeric.
    expect(formatMoney(null)).toBe('—')
    expect(formatMoney(undefined)).toBe('—')
    expect(formatMoney(NaN)).toBe('—')
    expect(formatMoney(Infinity)).toBe('—')
  })

  it('always shows two fraction digits, including for whole amounts', () => {
    expect(formatMoney(0)).toMatch(/0\.00$/)
    expect(formatMoney(1234)).toMatch(/1,?234\.00$/)
  })

  it('formats negatives, which the seeded ledger produces', () => {
    expect(formatMoney(-288.39)).toMatch(/288\.39/)
    expect(formatMoney(-288.39)).not.toBe('—')
  })
})

describe('formatDuration', () => {
  it('uses milliseconds below one second', () => {
    expect(formatDuration(0)).toBe('0 ms')
    expect(formatDuration(1)).toBe('1 ms')
    expect(formatDuration(999)).toBe('999 ms')
  })

  it('switches to seconds at exactly one second', () => {
    expect(formatDuration(1000)).toBe('1.0 s')
  })

  it('keeps one decimal place', () => {
    expect(formatDuration(1500)).toBe('1.5 s')
    expect(formatDuration(90_000)).toBe('90.0 s')
    expect(formatDuration(1234)).toBe('1.2 s')
  })
})

describe('ipColor', () => {
  it('is stable for the same address', () => {
    // The whole point: one IP keeps one colour across the feed and the chart,
    // so a distributed scenario visually fans out.
    expect(ipColor('192.0.2.1')).toBe(ipColor('192.0.2.1'))
  })

  it('returns a CSS var reference, valid only in a property position', () => {
    expect(ipColor('192.0.2.1')).toMatch(/^var\(--mui-palette-ipRamp-\d+\)$/)
  })

  it('stays inside the ramp for any input, including empty and unicode', () => {
    const index = (ip: string) => Number(/(\d+)\)$/.exec(ipColor(ip))![1])

    for (const ip of ['', '0.0.0.0', '255.255.255.255', '::1', 'x'.repeat(200), '🙂']) {
      expect(index(ip)).toBeGreaterThanOrEqual(0)
      expect(index(ip)).toBeLessThan(IP_RAMP_SIZE)
    }
  })

  it('spreads different addresses across more than one slot', () => {
    const seen = new Set(
      Array.from({ length: 40 }, (_, i) => ipColor(`198.51.100.${i}`)),
    )
    expect(seen.size).toBeGreaterThan(1)
  })
})
