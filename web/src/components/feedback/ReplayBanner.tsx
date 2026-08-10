import { Alert, Box, Link as MuiLink, Typography } from '@mui/material'
import { REPLAY } from '@/replay/mode'

const REPO_URL = 'https://github.com/mariaalexissales/go-ledger'

export function ReplayBanner() {
  if (!REPLAY) return null

  return (
    <Alert severity="info" icon={false} sx={{ py: 0.5 }}>
      <Typography variant="body2">
        <strong>Recorded demo.</strong> There is no backend here, since GitHub Pages cannot run Go
        or Postgres. Every scenario below is a real run captured from the actual server, replayed at
        a watchable speed. Ledger edits live in this tab only. Run it for real with{' '}
        <Box component="span" sx={{ color: 'primary.main' }}>
          docker compose up
        </Box>{' '}
        from{' '}
        <MuiLink href={REPO_URL} target="_blank" rel="noreferrer">
          the repo
        </MuiLink>
        .
      </Typography>
    </Alert>
  )
}
