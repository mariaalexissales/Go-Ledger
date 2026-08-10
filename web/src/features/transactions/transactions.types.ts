/** Mirrors api.Transaction in internal/api/models.go. */
export interface Transaction {
  id: number
  account_id: number
  /** pgtype.Numeric marshals as a bare JSON number. */
  amount: number | null
  timestamp: string
}

export interface TransactionListParams {
  limit?: number
  offset?: number
  account_id?: number
}

/**
 * Mirrors api.CreateTransactionRequest.
 *
 * `amount` must be sent as a JSON number. pgtype.Numeric feeds the raw bytes to
 * the Postgres numeric text scanner, so a quoted string fails to parse and the
 * server answers 400 "invalid request body".
 */
export interface CreateTransactionRequest {
  account_id: number
  amount: number
}
