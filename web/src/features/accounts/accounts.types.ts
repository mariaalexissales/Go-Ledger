/** Mirrors api.Account in internal/api/models.go. */
export interface Account {
  id: number
  name: string
  /** pgtype.Numeric marshals as a bare JSON number, or null when unset. */
  balance: number | null
  created_at: string
}

export interface AccountListParams {
  limit?: number
  offset?: number
  q?: string
}

/** Mirrors api.CreateAccountRequest. */
export interface CreateAccountRequest {
  name: string
}
