import { createRootRoute, Link, Outlet, useRouterState } from '@tanstack/react-router'
import {
  AppBar,
  Box,
  Chip,
  Container,
  IconButton,
  Stack,
  Tab,
  Tabs,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material'
import { useColorScheme } from '@mui/material/styles'
import DarkModeIcon from '@mui/icons-material/DarkMode'
import LightModeIcon from '@mui/icons-material/LightMode'
import CircleIcon from '@mui/icons-material/Circle'
import { EventStreamProvider, useEventStream } from '@/features/security/useEventStream'
import { RateLimitBanner } from '@/components/feedback/RateLimitBanner'
import { ReplayBanner } from '@/components/feedback/ReplayBanner'
import { REPLAY } from '@/replay/mode'

const NAV = [
  { to: '/', label: 'Dashboard' },
  { to: '/accounts', label: 'Accounts' },
  { to: '/transactions', label: 'Transactions' },
  { to: '/security', label: 'Security log' },
  { to: '/demos', label: 'Demos' },
] as const

export const Route = createRootRoute({ component: RootLayout })

function RootLayout() {
  return (
    <EventStreamProvider>
      <AppShell />
    </EventStreamProvider>
  )
}

function AppShell() {
  const pathname = useRouterState({ select: (s) => s.location.pathname })

  // Longest matching prefix, so /accounts/3 still highlights Accounts.
  const active =
    NAV.filter((item) => item.to === '/' ? pathname === '/' : pathname.startsWith(item.to))
      .sort((a, b) => b.to.length - a.to.length)[0]?.to ?? '/'

  return (
    // No background here on purpose. The page ground, texture and radial
    // washes, is painted on body, and an opaque shell would cover it.
    <Box sx={{ minHeight: '100vh' }}>
      <ReplayBanner />

      <AppBar position="sticky" color="default" elevation={0} sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Toolbar sx={{ gap: 2 }}>
          <Typography variant="h3" component="span" sx={{ mr: 1 }}>
            go-ledger
          </Typography>

          <Tabs value={active} sx={{ flex: 1, minHeight: 48 }}>
            {NAV.map((item) => (
              <Tab
                key={item.to}
                value={item.to}
                label={item.label}
                component={Link}
                to={item.to}
              />
            ))}
          </Tabs>

          <StreamIndicator />
          <ThemeToggle />
        </Toolbar>
      </AppBar>

      <RateLimitBanner />

      <Container maxWidth="xl" sx={{ py: 3 }}>
        <Outlet />
      </Container>
    </Box>
  )
}

function StreamIndicator() {
  const { status, events } = useEventStream()

  const connected = REPLAY ? 'replay' : 'live'
  const label =
    status === 'open'
      ? `${connected} · ${events.length}`
      : status === 'connecting'
        ? 'connecting'
        : 'offline'

  return (
    <Tooltip
      title={
        REPLAY
          ? 'Recorded events, replayed locally'
          : 'Live security event stream (SSE)'
      }
    >
      <Chip
        size="small"
        variant="outlined"
        icon={<CircleIcon sx={{ fontSize: 10 }} />}
        label={label}
        color={status === 'open' ? 'success' : status === 'connecting' ? 'default' : 'error'}
      />
    </Tooltip>
  )
}

function ThemeToggle() {
  const { mode, setMode } = useColorScheme()

  return (
    <Stack direction="row">
      <IconButton
        onClick={() => setMode(mode === 'dark' ? 'light' : 'dark')}
        aria-label="Toggle color scheme"
      >
        {mode === 'dark' ? <LightModeIcon /> : <DarkModeIcon />}
      </IconButton>
    </Stack>
  )
}
