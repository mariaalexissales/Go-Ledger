import { useState } from 'react'
import { createFileRoute, Link } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Divider,
  Stack,
  Typography,
} from '@mui/material'
import { DataTable, type Column, type Pagination } from '@/components/data/DataTable'
import AddIcon from '@mui/icons-material/Add'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import { accountsQueries } from '@/features/accounts/accounts.queries'
import { CreateTransactionDialog } from '@/features/transactions/components/CreateTransactionDialog'
import { ErrorState, LoadingRows } from '@/components/feedback/States'
import { useIsRateLimited } from '@/components/feedback/RateLimitBanner'
import { formatDateTime, formatMoney } from '@/lib/format'
import type { Transaction } from '@/features/transactions/transactions.types'

export const Route = createFileRoute('/accounts/$accountId')({ component: AccountDetailPage })

function AccountDetailPage() {
  const { accountId } = Route.useParams()
  const id = Number(accountId)

  const [dialogOpen, setDialogOpen] = useState(false)
  const [pagination, setPagination] = useState<Pagination>({ page: 0, pageSize: 25 })
  const rateLimited = useIsRateLimited()

  const account = useQuery(accountsQueries.detail(id))
  const transactions = useQuery(
    accountsQueries.transactions(id, {
      limit: pagination.pageSize,
      offset: pagination.page * pagination.pageSize,
    }),
  )

  const columns: Column<Transaction>[] = [
    { key: 'id', header: 'ID', width: 90 },
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
  ]

  if (account.isPending) return <LoadingRows rows={3} />
  if (account.isError) return <ErrorState error={account.error} onRetry={() => account.refetch()} />

  return (
    <Stack spacing={2}>
      <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
        <Button component={Link} to="/accounts" startIcon={<ArrowBackIcon />} size="small">
          Accounts
        </Button>
        <Box sx={{ flex: 1 }} />
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setDialogOpen(true)}
          disabled={rateLimited}
        >
          New transaction
        </Button>
      </Stack>

      <Card>
        <CardContent>
          <Typography variant="overline" color="text.secondary">
            Account #{account.data.id}
          </Typography>
          <Typography variant="h1" gutterBottom>
            {account.data.name}
          </Typography>

          <Stack direction="row" spacing={4} sx={{ mt: 2 }}>
            <Box>
              <Typography variant="overline" color="text.secondary" sx={{ display: 'block' }}>
                Balance
              </Typography>
              <Typography
                variant="h2"
                sx={{ color: (account.data.balance ?? 0) < 0 ? 'error.main' : 'inherit' }}
              >
                {formatMoney(account.data.balance)}
              </Typography>
            </Box>

            <Box>
              <Typography variant="overline" color="text.secondary" sx={{ display: 'block' }}>
                Created
              </Typography>
              <Typography variant="h2">{formatDateTime(account.data.created_at)}</Typography>
            </Box>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardHeader
          title="Transactions"
          subheader={`${transactions.data?.total ?? 0} on this account`}
        />
        <Divider />
        {transactions.isError ? (
          <CardContent>
            <ErrorState error={transactions.error} onRetry={() => transactions.refetch()} />
          </CardContent>
        ) : (
          <DataTable
            label="Transactions on this account"
            rows={transactions.data?.data ?? []}
            columns={columns}
            loading={transactions.isPending}
            rowCount={transactions.data?.total ?? 0}
            pagination={pagination}
            onPaginationChange={setPagination}
            pageSizeOptions={[10, 25, 50]}
            emptyTitle="No transactions"
            emptyDescription="Nothing has posted to this account yet."
            minWidth={520}
          />
        )}
      </Card>

      <CreateTransactionDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        accountId={id}
      />
    </Stack>
  )
}
