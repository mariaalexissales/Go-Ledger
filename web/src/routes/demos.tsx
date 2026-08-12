import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  Alert,
  AlertTitle,
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Divider,
  FormControlLabel,
  LinearProgress,
  Paper,
  Stack,
  Switch,
  Typography,
} from '@mui/material'
import RestartAltIcon from '@mui/icons-material/RestartAlt'
import { demosQueries, useRunDemo } from '@/features/demos/demos.queries'
import {
  securityQueries,
  useResetEvents,
  useSetClientIPMode,
} from '@/features/security/security.queries'
import { ScenarioCard } from '@/features/demos/components/ScenarioCard'
import { RunTimeline } from '@/features/demos/components/RunTimeline'
import { EventFeed } from '@/features/security/components/EventFeed'
import { useEventStream } from '@/features/security/useEventStream'
import { ErrorState, LoadingRows } from '@/components/feedback/States'
import { REPLAY } from '@/replay/mode'
import type { DemoResult } from '@/features/demos/demos.types'

export const Route = createFileRoute('/demos')({ component: DemosPage })

function DemosPage() {
  const demos = useQuery(demosQueries.list())
  const config = useQuery(securityQueries.config())
  const runDemo = useRunDemo()
  const resetEvents = useResetEvents()
  const setMode = useSetClientIPMode()
  const { events, clear } = useEventStream()

  const [result, setResult] = useState<DemoResult | null>(null)
  const [runningId, setRunningId] = useState<string | null>(null)

  const vulnerableMode = config.data?.client_ip_mode === 'xff-trust-all'

  const handleRun = (id: string) => {
    setRunningId(id)
    setResult(null)
    clear()

    runDemo.mutate(id, {
      onSuccess: setResult,
      onSettled: () => setRunningId(null),
    })
  }

  if (demos.isPending) return <LoadingRows rows={3} />
  if (demos.isError) return <ErrorState error={demos.error} onRetry={() => demos.refetch()} />

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h1" gutterBottom>
          Security demos
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 780 }}>
          {REPLAY
            ? 'Each scenario fired a scripted traffic pattern at the ledger API using synthetic source IPs, and the guard’s decisions were captured as they happened. Watch the run timeline and the event feed together. A step turning red on the left is the same request appearing as BLOCKED on the right.'
            : 'Each scenario fires a scripted traffic pattern at the ledger API from the server itself, using synthetic source IPs. Watch the run timeline and the live event feed together. A step turning red on the left is the same request appearing as BLOCKED on the right.'}
        </Typography>
      </Box>

      <Paper variant="outlined" sx={{ p: 2 }}>
        <Stack
          direction={{ xs: 'column', md: 'row' }}
          spacing={2}
          sx={{ alignItems: { md: 'center' } }}
        >
          <FormControlLabel
            control={
              <Switch
                checked={Boolean(vulnerableMode)}
                disabled={!config.data?.mutable || setMode.isPending || Boolean(runningId)}
                onChange={(e) => setMode.mutate(e.target.checked ? 'xff-trust-all' : 'remote-addr')}
              />
            }
            label={
              <Box>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>
                  Trust X-Forwarded-For {vulnerableMode ? '(vulnerable)' : '(hardened)'}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {vulnerableMode
                    ? 'The guard believes whatever source IP a client claims.'
                    : 'The guard ignores forwarding headers and uses the real socket peer.'}
                </Typography>
              </Box>
            }
          />

          <Box sx={{ flex: 1 }} />

          <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
            <Typography variant="caption" color="text.secondary">
              Limit {config.data?.rate_limit.limit} per {config.data?.rate_limit.window} · block{' '}
              {config.data?.rate_limit.block_period}
              {REPLAY && ' · fixed in this recording'}
            </Typography>

            <Button
              size="small"
              startIcon={<RestartAltIcon />}
              onClick={() => {
                resetEvents.mutate()
                clear()
                setResult(null)
              }}
              disabled={resetEvents.isPending || Boolean(runningId)}
            >
              {REPLAY ? 'Clear feed' : 'Clear log'}
            </Button>
          </Stack>
        </Stack>
      </Paper>

      <Box
        sx={{
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)', xl: 'repeat(3, 1fr)' },
        }}
      >
        {demos.data.data.map((meta) => (
          <ScenarioCard
            key={meta.id}
            meta={meta}
            vulnerableMode={Boolean(vulnerableMode)}
            running={runningId === meta.id}
            disabled={Boolean(runningId)}
            onRun={() => handleRun(meta.id)}
          />
        ))}
      </Box>

      {runningId && <LinearProgress />}

      {runDemo.isError && <ErrorState error={runDemo.error} />}

      <Box
        sx={{
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr', lg: '1.4fr 1fr' },
          alignItems: 'start',
        }}
      >
        <Card>
          <CardHeader
            title="Run timeline"
            subheader={
              result
                ? `${result.scenario_id} · mode ${result.client_ip_mode}`
                : 'Run a scenario to see every request'
            }
          />
          <Divider />
          {result ? (
            <>
              <RunTimeline result={result} />
              <Divider />
              <CardContent>
                {/*
                  The one gold element in the app. Gold is capped at once per
                  screen, so it goes to the single most useful thing on the
                  page: the sentence saying what the run proved. It is a
                  variant rather than a severity because no MUI severity is
                  gold, and leaving it as info or warning let it drift.
                */}
                <Alert variant="verdict" icon={false}>
                  <AlertTitle>Verdict</AlertTitle>
                  {result.summary.verdict}
                  <Typography variant="caption" component="p" sx={{ mt: 1 }}>
                    {result.teaches}
                  </Typography>
                  {REPLAY && (
                    <Typography variant="caption" component="p" sx={{ mt: 1, opacity: 0.8 }}>
                      The server handled this in {result.summary.duration_ms} ms. The replay above
                      is stretched out so it can be watched.
                    </Typography>
                  )}
                </Alert>
              </CardContent>
            </>
          ) : (
            <CardContent>
              <Typography variant="body2" color="text.secondary">
                No run yet.
              </Typography>
            </CardContent>
          )}
        </Card>

        <Card>
          <CardHeader
            title="Live security log"
            subheader="Streaming over SSE, straight from the guard"
          />
          <Divider />
          <EventFeed
            events={events}
            height={520}
            emptyHint="Run a scenario and the guard's decisions appear here in real time."
          />
        </Card>
      </Box>
    </Stack>
  )
}
