import type { AccountListParams } from '@/features/accounts/accounts.types'
import type { TransactionListParams } from '@/features/transactions/transactions.types'
import type { EventListParams } from '@/features/security/security.types'

/**
 * One factory for every cache key. Ad-hoc arrays at call sites are how
 * invalidation quietly stops matching.
 */
export const qk = {
  accounts: {
    all: ['accounts'] as const,
    list: (params: AccountListParams) => ['accounts', 'list', params] as const,
    detail: (id: number) => ['accounts', 'detail', id] as const,
    transactions: (id: number, params: TransactionListParams) =>
      ['accounts', 'detail', id, 'transactions', params] as const,
  },
  transactions: {
    all: ['transactions'] as const,
    list: (params: TransactionListParams) => ['transactions', 'list', params] as const,
  },
  security: {
    all: ['security'] as const,
    events: (params: EventListParams) => ['security', 'events', params] as const,
    stats: (window: string) => ['security', 'stats', window] as const,
    config: () => ['security', 'config'] as const,
  },
  demos: {
    all: ['demos'] as const,
    list: () => ['demos', 'list'] as const,
  },
} as const
