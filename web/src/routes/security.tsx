import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { DataTable, type Column, type Pagination } from '@/components/data/DataTable'
import {
  Alert,
  Box,
  Card,
  CardHeader,
  Chip,
  Divider,
  MenuItem,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { securityQueries } from '@/features/security/security.queries'
import { EventFeed } from '@/features/security/components/EventFeed'
import { useEventStream } from '@/features/security/useEventStream'
import { ErrorState } from '@/components/feedback/States'
import { formatDateTime, ipColor } from '@/lib/format'
import type { SecurityEvent } from '@/features/security/security.types'

export const Route = createFileRoute('/security')({ component: SecurityPage })

function SecurityPage() {
  const [flagStatus, setFlagStatus] = useState('')
  const [ipFilter, setIpFilter] = useState('')
  const [pagination, setPagination] = useState<Pagination>({ page: 0, pageSize: 50 })

  const { events, status, dropped } = useEventStream()
  const config = useQuery(securityQueries.config())

  const history = useQuery(
    securityQueries.events({
      limit: pagination.pageSize,
      offset: pagination.page * pagination.pageSize,
      flag_status: flagStatus || undefined,
      ip_address: ipFilter || undefined,
    }),
  )

  const columns: Column<SecurityEvent>[] = [
    {
      key: 'timestamp',
      header: 'When',
      width: 190,
      format: (value: string) => formatDateTime(value),
    },
    {
      key: 'ip_address',
      header: 'Source IP',
      width: 160,
      render: (row) => (
        <Box component="span" sx={{ color: ipColor(row.ip_address) }}>
          {row.ip_address}
        </Box>
      ),
    },
    { key: 'method', header: 'Method', width: 90 },
    // The one column without a width: it takes whatever is left.
    { key: 'path', header: 'Path' },
    {
      key: 'flag_status',
      header: 'Decision',
      width: 130,
      render: (row) => (
        <Chip
          size="small"
          label={row.flag_status}
          color={row.blocked ? 'error' : 'success'}
          variant={row.blocked ? 'glitch' : 'outlined'}
        />
      ),
    },
  ]

  return (
    <Stack spacing={2}>
      <Box>
        <Typography variant="h1" gutterBottom>
          Security log
        </Typography>
        <Typography variant="body2" color="text.secondary">
          Every request the guard evaluated. The console reads this plane outside the guard, so
          browsing here never pollutes the log or trips the limiter.
        </Typography>
      </Box>

      {dropped > 0 && (
        <Alert severity="warning">
          {dropped} live events were dropped because this browser fell behind. The feed below is a
          live tail, not a complete record. The table has everything.
        </Alert>
      )}

      {config.data && (
        <Stack direction="row" spacing={1} useFlexGap sx={{ flexWrap: 'wrap' }}>
          <Chip
            size="small"
            variant="outlined"
            color={config.data.client_ip_mode === 'xff-trust-all' ? 'warning' : 'success'}
            label={`IP mode: ${config.data.client_ip_mode}`}
          />
          <Chip
            size="small"
            variant="outlined"
            label={`limit ${config.data.rate_limit.limit}/${config.data.rate_limit.window}`}
          />
          <Chip size="small" variant="outlined" label={`stream: ${status}`} />
          {config.data.dropped_events > 0 && (
            <Chip
              size="small"
              color="warning"
              label={`${config.data.dropped_events} events dropped server-side`}
            />
          )}
        </Stack>
      )}

      <Box
        sx={{
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr', xl: '1.5fr 1fr' },
          alignItems: 'start',
        }}
      >
        <Stack spacing={2}>
          <Stack direction="row" spacing={2}>
            <TextField
              select
              size="small"
              label="Decision"
              value={flagStatus}
              onChange={(e) => {
                setFlagStatus(e.target.value)
                setPagination((p) => ({ ...p, page: 0 }))
              }}
              sx={{ minWidth: 160 }}
            >
              <MenuItem value="">All</MenuItem>
              <MenuItem value="ALLOWED">Allowed</MenuItem>
              <MenuItem value="BLOCKED">Blocked</MenuItem>
            </TextField>

            <TextField
              size="small"
              label="Source IP"
              placeholder="203.0.113.7"
              value={ipFilter}
              onChange={(e) => {
                setIpFilter(e.target.value)
                setPagination((p) => ({ ...p, page: 0 }))
              }}
            />
          </Stack>

          {history.isError && (
            <ErrorState error={history.error} onRetry={() => history.refetch()} />
          )}

          <Card>
            <DataTable
              label="Security event log"
              rows={history.data?.data ?? []}
              columns={columns}
              loading={history.isPending}
              rowCount={history.data?.total ?? 0}
              pagination={pagination}
              onPaginationChange={setPagination}
              pageSizeOptions={[25, 50, 100]}
              emptyTitle="No matching events"
              emptyDescription="Nothing the guard evaluated matches these filters."
              minWidth={760}
              maxHeight={640}
            />
          </Card>
        </Stack>

        <Card>
          <CardHeader title="Live tail" subheader="Server-sent events, no polling" />
          <Divider />
          <EventFeed events={events} height={640} />
        </Card>
      </Box>
    </Stack>
  )
}
