import { useState } from 'react'
import { createFileRoute, useNavigate } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { DataTable, type Column, type Pagination } from '@/components/data/DataTable'
import { Box, Button, Card, IconButton, Stack, TextField, Tooltip, Typography } from '@mui/material'
import AddIcon from '@mui/icons-material/Add'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutlined'
import { accountsQueries, useDeleteAccount } from '@/features/accounts/accounts.queries'
import { CreateAccountDialog } from '@/features/accounts/components/CreateAccountDialog'
import { ErrorState } from '@/components/feedback/States'
import { useIsRateLimited } from '@/components/feedback/RateLimitBanner'
import { formatDateTime, formatMoney } from '@/lib/format'
import type { Account } from '@/features/accounts/accounts.types'

export const Route = createFileRoute('/accounts/')({ component: AccountsPage })

function AccountsPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [pagination, setPagination] = useState<Pagination>({ page: 0, pageSize: 25 })

  const rateLimited = useIsRateLimited()
  const deleteAccount = useDeleteAccount()

  const params = {
    limit: pagination.pageSize,
    offset: pagination.page * pagination.pageSize,
    q: search || undefined,
  }
  const accounts = useQuery(accountsQueries.list(params))

  const columns: Column<Account>[] = [
    { key: 'id', header: 'ID', width: 80 },
    { key: 'name', header: 'Name' },
    {
      key: 'balance',
      header: 'Balance',
      width: 150,
      align: 'right',
      render: (row) => (
        <Box component="span" sx={{ color: (row.balance ?? 0) < 0 ? 'error.main' : 'inherit' }}>
          {formatMoney(row.balance)}
        </Box>
      ),
    },
    {
      key: 'created_at',
      header: 'Created',
      width: 180,
      format: (value: string) => formatDateTime(value),
    },
    {
      key: 'actions',
      header: '',
      width: 60,
      render: (row) => (
        <Tooltip title={rateLimited ? 'Rate limited' : 'Delete account'}>
          <span>
            <IconButton
              size="small"
              disabled={rateLimited || deleteAccount.isPending}
              onClick={(e) => {
                e.stopPropagation()
                deleteAccount.mutate(row.id)
              }}
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
          Accounts
        </Typography>

        <TextField
          size="small"
          label="Search by name"
          value={search}
          onChange={(e) => {
            setSearch(e.target.value)
            setPagination((p) => ({ ...p, page: 0 }))
          }}
        />

        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setDialogOpen(true)}
          disabled={rateLimited}
        >
          New account
        </Button>
      </Stack>

      {accounts.isError && <ErrorState error={accounts.error} onRetry={() => accounts.refetch()} />}
      {deleteAccount.isError && <ErrorState error={deleteAccount.error} />}

      <Card>
        <DataTable
          label="Accounts"
          rows={accounts.data?.data ?? []}
          columns={columns}
          loading={accounts.isPending}
          rowCount={accounts.data?.total ?? 0}
          pagination={pagination}
          onPaginationChange={setPagination}
          onRowClick={(row) =>
            navigate({ to: '/accounts/$accountId', params: { accountId: String(row.id) } })
          }
          emptyTitle="No accounts"
          emptyDescription={
            search ? 'Nothing matches that name.' : 'Seed the ledger to get started.'
          }
          minWidth={640}
        />
      </Card>

      <CreateAccountDialog open={dialogOpen} onClose={() => setDialogOpen(false)} />
    </Stack>
  )
}
