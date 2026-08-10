import { IP_RAMP_SIZE } from '@/theme'

/**
 * `balance` and `amount` arrive as bare JSON numbers because the Go side uses
 * pgtype.Numeric, which marshals that way. The column is NUMERIC(15,2); the top
 * of that range sits at the edge of what a float64 represents exactly, which is
 * fine for a seeded demo ledger but would need a decimal type for real money.
 */
const money = new Intl.NumberFormat(undefined, {
  style: 'currency',
  currency: 'USD',
  minimumFractionDigits: 2,
})

export function formatMoney(value: number | null | undefined): string {
  if (value === null || value === undefined || !Number.isFinite(value)) return '—'
  return money.format(value)
}

const dateTime = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short',
})

export function formatDateTime(value: string | null | undefined): string {
  if (!value) return '—'

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'

  if (date.getUTCFullYear() < 1900) return '—'

  return dateTime.format(date)
}

export function formatTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'

  return date.toLocaleTimeString(undefined, {
    hour12: false,
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

/**
 * Stable color per IP so a distributed scenario visually fans out in the feed.
 *
 * Indexes into the palette's `ipRamp`: 12 hues from blue-violet to hot across
 * two lightness tiers, defined per color scheme in theme/tokens.ts.
 *
 * Returns a CSS `var()` reference, so it is only valid in a property position.
 * Note that SVG presentation attributes are *not* one: `<rect fill={ipColor(…)}>`
 * renders black. Use `style={{ fill: … }}` there.
 */
export function ipColor(ip: string): string {
  let hash = 0
  for (let i = 0; i < ip.length; i++) {
    hash = (hash * 31 + ip.charCodeAt(i)) | 0
  }
  return `var(--mui-palette-ipRamp-${Math.abs(hash) % IP_RAMP_SIZE})`
}
