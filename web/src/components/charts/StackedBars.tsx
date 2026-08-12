import { useEffect, useId, useRef, useState } from 'react'
import Box from '@mui/material/Box'
import Stack from '@mui/material/Stack'
import Typography from '@mui/material/Typography'
import { visuallyHidden } from '@mui/utils'

/**
 * A stacked bar chart, in about a page of SVG.
 */

interface Band {
  label: string
  /** One entry per series, in the same order. */
  values: readonly number[]
}

interface Serie {
  label: string
  /** Any CSS color. A `var(...)` reference is fine, see the note below. */
  color: string
}

const PAD = { top: 8, right: 8, bottom: 22, left: 36 }
/** Past this the bars are narrower than the gaps between them and it is noise. */
const MAX_BANDS = 200

/** Rounds up to 1, 2 or 5 × 10ⁿ so the gridline labels are readable numbers. */
function niceMax(value: number): number {
  if (value <= 0) return 1

  const base = 10 ** Math.floor(Math.log10(value))
  const frac = value / base
  return (frac <= 1 ? 1 : frac <= 2 ? 2 : frac <= 5 ? 5 : 10) * base
}

function useWidth(ref: React.RefObject<HTMLDivElement | null>): number {
  const [width, setWidth] = useState(0)

  useEffect(() => {
    const node = ref.current
    if (!node) return

    const observer = new ResizeObserver(([entry]) => setWidth(entry.contentRect.width))
    observer.observe(node)
    return () => observer.disconnect()
  }, [ref])

  return width
}

export function StackedBars({
  bands,
  series,
  height = 260,
  label,
}: {
  bands: readonly Band[]
  series: readonly Serie[]
  height?: number
  /** Accessible name. Required, since a bare SVG is invisible to screen readers. */
  label: string
}) {
  const ref = useRef<HTMLDivElement>(null)
  const width = useWidth(ref)
  const id = useId()

  if (bands.length > MAX_BANDS && import.meta.env.DEV) {
    console.warn(
      `StackedBars: ${bands.length} bands exceeds the ${MAX_BANDS} cap and will be truncated. ` +
        'Widen the bucket size rather than the window.',
    )
  }
  const shown = bands.slice(-MAX_BANDS)

  const totals = shown.map((b) => b.values.reduce((sum, v) => sum + v, 0))
  const max = niceMax(Math.max(0, ...totals))

  const plotW = Math.max(0, width - PAD.left - PAD.right)
  const plotH = Math.max(0, height - PAD.top - PAD.bottom)
  const bandW = shown.length > 0 ? plotW / shown.length : 0
  const barW = Math.max(1, Math.min(bandW * 0.7, 28))

  // Thin the x labels to whatever fits at ~52px each.
  const every = Math.max(1, Math.ceil(shown.length / Math.max(2, Math.floor(plotW / 52))))

  const seriesTotals = series.map((_, s) => shown.reduce((sum, b) => sum + (b.values[s] ?? 0), 0))
  const peak = Math.max(0, ...totals)
  const peakBand = shown[totals.indexOf(peak)]

  const summary = [
    series.map((serie, s) => `${seriesTotals[s]} ${serie.label.toLowerCase()}`).join(', '),
    `across ${shown.length} ${shown.length === 1 ? 'bucket' : 'buckets'}.`,
    peakBand ? `Peak ${peak} requests at ${peakBand.label}.` : '',
  ]
    .filter(Boolean)
    .join(' ')

  return (
    <Box>
      <Box ref={ref} sx={{ width: '100%' }}>
        {width > 0 && (
          <svg
            width={width}
            height={height}
            viewBox={`0 0 ${width} ${height}`}
            role="img"
            aria-labelledby={`${id}-title ${id}-desc`}
            style={{ display: 'block', fontVariantNumeric: 'tabular-nums' }}
          >
            <title id={`${id}-title`}>{label}</title>
            <desc id={`${id}-desc`}>{summary}</desc>

            {[0, 0.5, 1].map((frac) => {
              const y = PAD.top + plotH * (1 - frac)
              return (
                <g key={frac}>
                  <line
                    x1={PAD.left}
                    x2={width - PAD.right}
                    y1={y}
                    y2={y}
                    style={{ stroke: 'var(--mui-palette-estral-hairline)' }}
                    strokeWidth={1}
                  />
                  <text
                    x={PAD.left - 6}
                    y={y + 3}
                    textAnchor="end"
                    fontSize={10}
                    style={{ fill: 'var(--mui-palette-text-secondary)' }}
                  >
                    {Math.round(max * frac)}
                  </text>
                </g>
              )
            })}

            {shown.map((band, i) => {
              const cx = PAD.left + bandW * i + bandW / 2
              const total = totals[i]

              // Stack from the bottom up, so the last series lands on top.
              let cursor = PAD.top + plotH

              return (
                <g key={`${band.label}-${i}`}>
                  <title>
                    {`${band.label}: `}
                    {series
                      .map((serie, s) => `${band.values[s] ?? 0} ${serie.label.toLowerCase()}`)
                      .join(', ')}
                  </title>

                  {total === 0 ? (
                    // A gap would read as "no data". The server emits explicit
                    // zeros for quiet minutes, so this has to read as
                    // "measured, and it was nothing".
                    <line
                      x1={cx - barW / 2}
                      x2={cx + barW / 2}
                      y1={PAD.top + plotH}
                      y2={PAD.top + plotH}
                      style={{ stroke: 'var(--mui-palette-text-disabled)' }}
                      strokeWidth={1}
                    />
                  ) : (
                    series.map((serie, s) => {
                      const value = band.values[s] ?? 0
                      if (value <= 0) return null

                      const h = (value / max) * plotH
                      cursor -= h

                      return (
                        <rect
                          key={serie.label}
                          x={cx - barW / 2}
                          y={cursor}
                          width={barW}
                          height={h}
                          // SVG presentation attributes do not resolve var().
                          // `fill={serie.color}` renders black; this does not.
                          style={{ fill: serie.color }}
                        />
                      )
                    })
                  )}

                  {i % every === 0 && (
                    <text
                      x={cx}
                      y={height - 6}
                      textAnchor="middle"
                      fontSize={10}
                      style={{ fill: 'var(--mui-palette-text-secondary)' }}
                    >
                      {band.label}
                    </text>
                  )}
                </g>
              )
            })}
          </svg>
        )}
        {width === 0 && <Box sx={{ height }} />}
      </Box>

      <Stack direction="row" spacing={2} useFlexGap sx={{ flexWrap: 'wrap', mt: 1 }}>
        {series.map((serie) => (
          <Stack key={serie.label} direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
            <Box sx={{ width: 10, height: 10, bgcolor: serie.color }} />
            <Typography variant="caption" color="text.secondary">
              {serie.label}
            </Typography>
          </Stack>
        ))}
      </Stack>

      {/*
        role="img" makes the SVG opaque to screen readers, so the numbers have
        to exist somewhere reachable. The chart this replaced had no equivalent.
      */}
      <Box component="table" sx={visuallyHidden}>
        <caption>{label}</caption>
        <thead>
          <tr>
            <th scope="col">Time</th>
            {series.map((serie) => (
              <th key={serie.label} scope="col">
                {serie.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {shown.map((band, i) => (
            <tr key={`${band.label}-${i}`}>
              <th scope="row">{band.label}</th>
              {series.map((serie, s) => (
                <td key={serie.label}>{band.values[s] ?? 0}</td>
              ))}
            </tr>
          ))}
        </tbody>
      </Box>
    </Box>
  )
}
