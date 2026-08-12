import { Button, Card, CardActions, CardContent, Chip, Stack, Typography } from '@mui/material'
import PlayArrowIcon from '@mui/icons-material/PlayArrow'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import type { DemoMeta } from '../demos.types'

export function ScenarioCard({
  meta,
  onRun,
  running,
  disabled,
  vulnerableMode,
}: {
  meta: DemoMeta
  onRun: () => void
  running: boolean
  disabled: boolean
  vulnerableMode: boolean
}) {
  // The spoof scenario still runs in hardened mode. Running it twice is the
  // whole point, so this is a note rather than a block.
  const modeMismatch = meta.requires_vulnerable_mode && !vulnerableMode

  return (
    <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <CardContent sx={{ flex: 1 }}>
        <Stack direction="row" spacing={1} sx={{ mb: 1, flexWrap: 'wrap', gap: 0.5 }}>
          {meta.tags.map((tag) => (
            <Chip key={tag} label={tag} size="small" variant="outlined" />
          ))}
          {meta.requires_vulnerable_mode && (
            <Chip
              icon={<WarningAmberIcon />}
              label="tests the XFF hole"
              size="small"
              color="warning"
              variant="outlined"
            />
          )}
        </Stack>

        <Typography variant="h3" gutterBottom>
          {meta.name}
        </Typography>

        <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
          {meta.summary}
        </Typography>

        <Typography variant="caption" color="text.secondary" component="p" sx={{ mb: 1 }}>
          <strong>What it proves:</strong> {meta.teaches}
        </Typography>

        <Typography variant="caption" color="text.secondary" component="p">
          <strong>Watch for:</strong> {meta.expect}
        </Typography>

        {modeMismatch && (
          <Typography variant="caption" color="warning.main" component="p" sx={{ mt: 1 }}>
            Client IP mode is hardened. Run this to see the spoofing fail.
          </Typography>
        )}
      </CardContent>

      <CardActions sx={{ px: 2, pb: 2, justifyContent: 'space-between' }}>
        <Typography variant="caption" color="text.secondary">
          ~{meta.estimated_seconds}s
        </Typography>

        <Button
          variant="contained"
          size="small"
          startIcon={<PlayArrowIcon />}
          onClick={onRun}
          disabled={disabled}
        >
          {running ? 'Running…' : 'Run'}
        </Button>
      </CardActions>
    </Card>
  )
}
