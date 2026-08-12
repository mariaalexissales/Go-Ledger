import { Box, Chip, Stack, Typography } from '@mui/material'
import { formatTime, ipColor } from '@/lib/format'
import type { SecurityEvent } from '../security.types'
import { EmptyState } from '@/components/feedback/States'

/**
 * The live tail of the security log. Rows are keyed by the database id, which
 * the server assigns before publishing, so React never sees a duplicate key.
 */
export function EventFeed({
  events,
  height = 420,
  emptyHint,
}: {
  events: SecurityEvent[]
  height?: number | string
  emptyHint?: string
}) {
  if (events.length === 0) {
    return (
      <EmptyState
        title="No events yet"
        description={emptyHint ?? 'Traffic to the ledger API shows up here the moment it arrives.'}
      />
    )
  }

  return (
    <Box sx={{ height, overflowY: 'auto' }}>
      <Stack spacing={0.5} sx={{ p: 1 }}>
        {events.map((event) => (
          <EventRow key={event.id} event={event} />
        ))}
      </Stack>
    </Box>
  )
}

function EventRow({ event }: { event: SecurityEvent }) {
  return (
    <Stack
      direction="row"
      spacing={1}
      sx={{
        alignItems: 'center',
        px: 1,
        py: 0.5,
        borderRadius: 1,
        fontSize: 13,
        // A tinted row rather than a saturated block, so a screen full of
        // blocked events stays readable.
        bgcolor: event.blocked ? 'estral.blockedRow' : 'transparent',
      }}
    >
      <Typography component="span" sx={{ fontSize: 12, color: 'text.secondary' }}>
        {formatTime(event.timestamp)}
      </Typography>

      <Chip
        size="small"
        label={event.ip_address}
        sx={{
          fontSize: 11,
          height: 20,
          bgcolor: 'transparent',
          border: '1px solid',
          borderColor: ipColor(event.ip_address),
          color: ipColor(event.ip_address),
        }}
      />

      <Typography component="span" sx={{ fontSize: 13, flex: 1, minWidth: 0 }} noWrap>
        <Box component="span" sx={{ fontWeight: 600, mr: 0.75 }}>
          {event.method}
        </Box>
        {event.path}
      </Typography>

      <Chip
        size="small"
        color={event.blocked ? 'error' : 'success'}
        // Aberration needs a dark ground. Split over contrastText on a
        // saturated fill just reads as mud.
        variant={event.blocked ? 'glitch' : 'outlined'}
        label={event.flag_status}
        sx={{ fontSize: 10, height: 20 }}
      />
    </Stack>
  )
}
