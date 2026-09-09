export interface Transaction {
  id: number
  account_id: number
  amount: number | null
  timestamp: string
}

export interface TransactionListParams {
  limit?: number
  offset?: number
  account_id?: number
}

export interface CreateTransactionRequest {
  account_id: number
  amount: number
}
