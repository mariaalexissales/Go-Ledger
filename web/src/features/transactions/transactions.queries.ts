import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/lib/queryKeys'
import { transactionsApi } from './transactions.api'
import type { CreateTransactionRequest, TransactionListParams } from './transactions.types'

export const transactionsQueries = {
  list: (params: TransactionListParams) =>
    queryOptions({
      queryKey: qk.transactions.list(params),
      queryFn: ({ signal }) => transactionsApi.list(params, signal),
    }),
}

export function useCreateTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (body: CreateTransactionRequest) => transactionsApi.create(body),
    onSuccess: () => {
      // A transaction moves the account balance, so both trees are stale.
      queryClient.invalidateQueries({ queryKey: qk.transactions.all })
      queryClient.invalidateQueries({ queryKey: qk.accounts.all })
    },
  })
}

export function useDeleteTransaction() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: number) => transactionsApi.remove(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.transactions.all })
      queryClient.invalidateQueries({ queryKey: qk.accounts.all })
    },
  })
}
