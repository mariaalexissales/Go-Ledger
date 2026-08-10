import { api, type ListResponse } from '@/lib/http/client'
import type { Account, AccountListParams, CreateAccountRequest } from './accounts.types'
import type { Transaction } from '@/features/transactions/transactions.types'
import type { TransactionListParams } from '@/features/transactions/transactions.types'

/**
 * Transport only, no React and no cache. Components import the hooks in
 * accounts.queries.ts rather than this module directly.
 */
export const accountsApi = {
  list: (params: AccountListParams, signal?: AbortSignal) =>
    api.get<ListResponse<Account>>('/api/accounts', params, signal),

  get: (id: number, signal?: AbortSignal) => api.get<Account>(`/api/accounts/${id}`, undefined, signal),

  listTransactions: (id: number, params: TransactionListParams, signal?: AbortSignal) =>
    api.get<ListResponse<Transaction>>(`/api/accounts/${id}/transactions`, params, signal),

  create: (body: CreateAccountRequest) => api.post<Account>('/api/accounts', body),

  remove: (id: number) => api.delete<void>(`/api/accounts/${id}`),
}
