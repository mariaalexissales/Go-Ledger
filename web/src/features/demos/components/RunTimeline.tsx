import { Box, Chip, Divider, Stack, Typography } from '@mui/material'
import { formatDuration, ipColor } from '@/lib/format'
import type { DemoResult, DemoStep } from '../demos.types'

export function RunTimeline({ result }: { result: DemoResult }) {
  return (
    <Box>
      <Stack direction="row" spacing={3} useFlexGap sx={{ px: 2, py: 1.5, flexWrap: 'wrap' }}>
        <Counter label="Sent" value={result.summary.sent} />
        <Counter label="Allowed" value={result.summary.allowed} color="success.main" />
        <Counter label="Blocked" value={result.summary.blocked} color="error.main" />
        <Counter label="Source IPs" value={result.summary.distinct_ips} />
        <Counter label="Took" value={formatDuration(result.summary.duration_ms)} />
      </Stack>

      <Divider />

      <Box sx={{ maxHeight: 460, overflowY: 'auto' }}>
        <Stack spacing={0.25} sx={{ p: 1 }}>
          {result.steps.map((step) => (
            <StepRow key={step.seq} step={step} />
          ))}
        </Stack>
      </Box>
    </Box>
  )
}

function Counter({ label, value, color }: { label: string; value: string | number; color?: string }) {
  return (
    <Box>
      <Typography variant="overline" color="text.secondary" sx={{ display: 'block', lineHeight: 1.2 }}>
        {label}
      </Typography>
      <Typography variant="h2" sx={{ color, fontVariantNumeric: 'tabular-nums' }}>
        {value}
      </Typography>
    </Box>
  )
}

function StepRow({ step }: { step: DemoStep }) {
  // A step with no method is an annotation the scenario wrote, not a request.
  if (!step.method) {
    return (
      <Typography
        variant="caption"
        color="text.disabled"
        sx={{ px: 1, py: 0.5 }}
      >
        # {step.note}
      </Typography>
    )
  }

  return (
    <Stack
      direction="row"
      spacing={1}
      sx={{
        alignItems: 'center',
        px: 1,
        py: 0.25,
        borderRadius: 1,
        fontSize: 12.5,
        bgcolor: step.blocked ? 'estral.blockedRow' : 'transparent',
      }}
    >
      <Box component="span" sx={{ width: 34, color: 'text.disabled', textAlign: 'right' }}>
        {step.seq}
      </Box>

      <Box component="span" sx={{ width: 56, color: 'text.secondary', textAlign: 'right' }}>
        +{step.elapsed_ms}ms
      </Box>

      <Chip
        size="small"
        label={step.client_ip}
        title={step.spoofed ? 'Claimed via X-Forwarded-For' : 'Trusted demo identity'}
        sx={{
            fontSize: 11,
          height: 19,
          bgcolor: 'transparent',
          border: '1px solid',
          borderStyle: step.spoofed ? 'dashed' : 'solid',
          borderColor: ipColor(step.client_ip),
          color: ipColor(step.client_ip),
        }}
      />

      <Box component="span" sx={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}>
        <strong>{step.method}</strong> {step.path}
      </Box>

      {step.error ? (
        <Chip size="small" color="warning" label={step.error} sx={{ fontSize: 10, height: 19 }} />
      ) : (
        <Chip
          size="small"
          color={step.blocked ? 'error' : step.status < 400 ? 'success' : 'default'}
          variant={step.blocked ? 'glitch' : 'outlined'}
          label={step.blocked ? `429 · retry ${step.retry_after_sec}s` : step.status}
          sx={{ fontSize: 10, height: 19 }}
        />
      )}
    </Stack>
  )
}
