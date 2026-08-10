import { Card, CardContent, Stack, Typography } from '@mui/material'
import type { ReactNode } from 'react'

export function StatCard({
  label,
  value,
  hint,
  color,
  icon,
}: {
  label: string
  value: ReactNode
  hint?: string
  color?: string
  icon?: ReactNode
}) {
  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 0.5 }}>
          {icon}
          <Typography variant="overline" color="text.secondary" sx={{ lineHeight: 1.2 }}>
            {label}
          </Typography>
        </Stack>

        <Typography variant="h1" sx={{ color, fontVariantNumeric: 'tabular-nums' }}>
          {value}
        </Typography>

        {hint && (
          <Typography variant="caption" color="text.secondary">
            {hint}
          </Typography>
        )}
      </CardContent>
    </Card>
  )
}
