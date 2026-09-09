export interface Account {
  id: number
  name: string
  balance: number | null
  created_at: string
}

export interface AccountListParams {
  limit?: number
  offset?: number
  q?: string
}

export interface CreateAccountRequest {
  name: string
}
