import { useState } from 'react'
import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { DataTable, type Column, type Pagination } from '@/components/data/DataTable'
import { Box, Button, Card, IconButton, Stack, TextField, Tooltip, Typography } from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutlined'
import {
  transactionsQueries,
  useDeleteTransaction,
} from '@/features/transactions/transactions.queries'
import { CreateTransactionDialog } from '@/features/transactions/components/CreateTransactionDialog'
import { ErrorState } from '@/components/feedback/States'
import { useIsRateLimited } from '@/components/feedback/RateLimitBanner'
import { formatDateTime, formatMoney } from '@/lib/format'
import type { Transaction } from '@/features/transactions/transactions.types'

export const Route = createFileRoute('/transactions/')({ component: TransactionsPage })

function TransactionsPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [accountFilter, setAccountFilter] = useState('')
  const [pagination, setPagination] = useState<Pagination>({ page: 0, pageSize: 25 })

  const rateLimited = useIsRateLimited()
  const deleteTransaction = useDeleteTransaction()

  const accountId = Number(accountFilter)
  const transactions = useQuery(
    transactionsQueries.list({
      limit: pagination.pageSize,
      offset: pagination.page * pagination.pageSize,
      account_id: Number.isInteger(accountId) && accountId > 0 ? accountId : undefined,
    }),
  )

  const columns: Column<Transaction>[] = [
    { key: 'id', header: 'ID', width: 90 },
    {
      key: 'account_id',
      header: 'Account',
      width: 110,
      render: (row) => (
        <Link to="/accounts/$accountId" params={{ accountId: String(row.account_id) }}>
          #{row.account_id}
        </Link>
      ),
    },
    {
      key: 'amount',
      header: 'Amount',
      width: 160,
      align: 'right',
      render: (row) => (
        <Box component="span" sx={{ color: (row.amount ?? 0) < 0 ? 'error.main' : 'success.main' }}>
          {formatMoney(row.amount)}
        </Box>
      ),
    },
    {
      key: 'timestamp',
      header: 'Posted',
      format: (value: string) => formatDateTime(value),
    },
    {
      key: 'actions',
      header: '',
      width: 60,
      render: (row) => (
        <Tooltip title={rateLimited ? 'Rate limited' : 'Delete transaction'}>
          <span>
            <IconButton
              size="small"
              disabled={rateLimited || deleteTransaction.isPending}
              onClick={() => deleteTransaction.mutate(row.id)}
            >
              <DeleteOutlineIcon fontSize="small" />
            </IconButton>
          </span>
        </Tooltip>
      ),
    },
  ]

  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
        <Typography variant="h1" sx={{ flex: 1 }}>
          Transactions
        </Typography>

        <TextField
          size="small"
          label="Filter by account ID"
          type="number"
          value={accountFilter}
          onChange={(e) => {
            setAccountFilter(e.target.value)
            setPagination((p) => ({ ...p, page: 0 }))
          }}
        />

        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setDialogOpen(true)}
          disabled={rateLimited}
        >
          New transaction
        </Button>
      </Stack>

      {transactions.isError && (
        <ErrorState error={transactions.error} onRetry={() => transactions.refetch()} />
      )}
      {deleteTransaction.isError && <ErrorState error={deleteTransaction.error} />}

      <Card>
        <DataTable
          label="Transactions"
          rows={transactions.data?.data ?? []}
          columns={columns}
          loading={transactions.isPending}
          rowCount={transactions.data?.total ?? 0}
          pagination={pagination}
          onPaginationChange={setPagination}
          emptyTitle="No transactions"
          emptyDescription={
            accountFilter ? 'Nothing posted to that account.' : 'Post one to get started.'
          }
          minWidth={640}
        />
      </Card>

      <CreateTransactionDialog open={dialogOpen} onClose={() => setDialogOpen(false)} />
    </Stack>
  )
}
