import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/lib/queryKeys'
import { accountsApi } from './accounts.api'
import type { AccountListParams, CreateAccountRequest } from './accounts.types'
import type { TransactionListParams } from '@/features/transactions/transactions.types'

export const accountsQueries = {
  list: (params: AccountListParams) =>
    queryOptions({
      queryKey: qk.accounts.list(params),
      queryFn: ({ signal }) => accountsApi.list(params, signal),
    }),

  detail: (id: number) =>
    queryOptions({
      queryKey: qk.accounts.detail(id),
      queryFn: ({ signal }) => accountsApi.get(id, signal),
    }),

  transactions: (id: number, params: TransactionListParams) =>
    queryOptions({
      queryKey: qk.accounts.transactions(id, params),
      queryFn: ({ signal }) => accountsApi.listTransactions(id, params, signal),
    }),
}

export function useCreateAccount() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (body: CreateAccountRequest) => accountsApi.create(body),
    // Not optimistic: the server assigns id, balance and created_at, so an
    // optimistic row would render with invented values and visibly flicker.
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.accounts.all })
    },
  })
}

export function useDeleteAccount() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => accountsApi.remove(id),
    onSuccess: () => {
      // Deleting an account cascades to its transactions.
      queryClient.invalidateQueries({ queryKey: qk.accounts.all })
      queryClient.invalidateQueries({ queryKey: qk.transactions.all })
    },
  })
}
