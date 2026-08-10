import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Divider,
  Stack,
  Typography,
} from '@mui/material'
import { StackedBars } from '@/components/charts/StackedBars'
import ShieldIcon from '@mui/icons-material/Shield'
import BlockIcon from '@mui/icons-material/Block'
import PublicIcon from '@mui/icons-material/Public'
import AccountBalanceIcon from '@mui/icons-material/AccountBalance'
import { securityQueries } from '@/features/security/security.queries'
import { accountsQueries } from '@/features/accounts/accounts.queries'
import { StatCard } from '@/components/StatCard'
import { EventFeed } from '@/features/security/components/EventFeed'
import { useEventStream } from '@/features/security/useEventStream'
import { ErrorState, LoadingRows } from '@/components/feedback/States'
import { ipColor } from '@/lib/format'

export const Route = createFileRoute('/')({ component: DashboardPage })

const STATS_WINDOW = '30m'

function DashboardPage() {
  const stats = useQuery(securityQueries.stats(STATS_WINDOW))
  const config = useQuery(securityQueries.config())
  const accounts = useQuery(accountsQueries.list({ limit: 1 }))
  const { events } = useEventStream()

  if (stats.isPending) return <LoadingRows rows={4} />
  if (stats.isError) return <ErrorState error={stats.error} onRetry={() => stats.refetch()} />

  const { totals, distinct_ips, top_ips, buckets, blocked_now } = stats.data

  return (
    <Stack spacing={3}>
      <Box>
        <Typography variant="h1" gutterBottom>
          Dashboard
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Everything the security guard has seen in the last {stats.data.window}.
        </Typography>
      </Box>

      <Box
        sx={{
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(4, 1fr)' },
        }}
      >
        <StatCard
          label="Allowed"
          value={totals.ALLOWED.toLocaleString()}
          color="success.main"
          icon={<ShieldIcon fontSize="small" color="success" />}
        />
        <StatCard
          label="Blocked"
          value={totals.BLOCKED.toLocaleString()}
          color="error.main"
          icon={<BlockIcon fontSize="small" color="error" />}
          hint={blocked_now.length > 0 ? `${blocked_now.length} IP(s) blocked right now` : undefined}
        />
        <StatCard
          label="Source IPs"
          value={distinct_ips.toLocaleString()}
          icon={<PublicIcon fontSize="small" />}
        />
        <StatCard
          label="Accounts"
          value={accounts.data?.total.toLocaleString() ?? '—'}
          icon={<AccountBalanceIcon fontSize="small" />}
        />
      </Box>

      {config.data && (
        <Card>
          <CardContent>
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ alignItems: { md: 'center' } }}>
              <Chip
                label={`client IP mode: ${config.data.client_ip_mode}`}
                color={config.data.client_ip_mode === 'xff-trust-all' ? 'warning' : 'success'}
                variant="outlined"
              />
              <Typography variant="body2" color="text.secondary">
                Rate limit {config.data.rate_limit.limit} per {config.data.rate_limit.window}, block{' '}
                {config.data.rate_limit.block_period}. The guard sees this browser as{' '}
                {/* The whole UI is mono, so inline code needs color to stand
                    out of prose rather than a font family switch. */}
                <Box component="span" sx={{ color: 'primary.main' }}>
                  {config.data.your_ip}
                </Box>
                .
              </Typography>
              <Box sx={{ flex: 1 }} />
              <Button component={Link} to="/demos" variant="contained" size="small">
                Run a demo
              </Button>
            </Stack>
          </CardContent>
        </Card>
      )}

      <Box
        sx={{
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr', lg: '1.3fr 1fr' },
          alignItems: 'start',
        }}
      >
        <Card>
          <CardHeader title="Requests per minute" subheader={`Last ${stats.data.window}`} />
          <Divider />
          <CardContent>
            {buckets.length > 0 ? (
              <StackedBars
                height={260}
                label={`Requests per minute, last ${stats.data.window}`}
                series={[
                  // Cyan is the readout color: an allowed request is the guard
                  // reporting nominal. Blocked is the fault.
                  { label: 'Allowed', color: 'var(--mui-palette-success-main)' },
                  { label: 'Blocked', color: 'var(--mui-palette-error-main)' },
                ]}
                bands={buckets.map((b) => ({
                  label: new Date(b.bucket).toLocaleTimeString(undefined, {
                    hour: '2-digit',
                    minute: '2-digit',
                  }),
                  values: [b.allowed, b.blocked],
                }))}
              />
            ) : (
              <Typography variant="body2" color="text.secondary">
                No traffic in this window yet.
              </Typography>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader title="Live feed" subheader="Streaming as requests arrive" />
          <Divider />
          <EventFeed events={events} height={320} />
        </Card>
      </Box>

      <Card>
        <CardHeader title="Busiest source IPs" subheader={`Last ${stats.data.window}`} />
        <Divider />
        <CardContent>
          {top_ips.length === 0 ? (
            <Typography variant="body2" color="text.secondary">
              Nothing recorded yet.
            </Typography>
          ) : (
            <Stack spacing={1}>
              {top_ips.map((ip) => (
                <Stack key={ip.ip_address} direction="row" spacing={2} sx={{ alignItems: 'center' }}>
                  <Box component="span" sx={{ minWidth: 150, color: ipColor(ip.ip_address) }}>
                    {ip.ip_address}
                  </Box>
                  <Typography variant="body2" color="text.secondary">
                    {ip.total} requests
                  </Typography>
                  {ip.blocked > 0 && (
                    <Chip size="small" color="error" label={`${ip.blocked} blocked`} />
                  )}
                </Stack>
              ))}
            </Stack>
          )}
        </CardContent>
      </Card>
    </Stack>
  )
}
