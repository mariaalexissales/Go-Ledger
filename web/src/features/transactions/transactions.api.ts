import { api, type ListResponse } from '@/lib/http/client'
import type {
  CreateTransactionRequest,
  Transaction,
  TransactionListParams,
} from './transactions.types'

export const transactionsApi = {
  list: (params: TransactionListParams, signal?: AbortSignal) =>
    api.get<ListResponse<Transaction>>('/api/transactions', params, signal),

  create: (body: CreateTransactionRequest) => api.post<Transaction>('/api/transactions', body),

  remove: (id: number) => api.delete<void>(`/api/transactions/${id}`),
}
