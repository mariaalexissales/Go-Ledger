import { Box } from '@mui/material'
import type { Column } from '@/components/data/DataTable'
import { formatDateTime, formatMoney } from '@/lib/format'
import type { Transaction } from './transactions.types'

export const idColumn: Column<Transaction> = { key: 'id', header: 'ID', width: 90 }

export const amountColumn: Column<Transaction> = {
  key: 'amount',
  header: 'Amount',
  width: 160,
  align: 'right',
  render: (row) => (
    <Box component="span" sx={{ color: (row.amount ?? 0) < 0 ? 'error.main' : 'success.main' }}>
      {formatMoney(row.amount)}
    </Box>
  ),
}

export const postedColumn: Column<Transaction> = {
  key: 'timestamp',
  header: 'Posted',
  format: (value: string) => formatDateTime(value),
}
