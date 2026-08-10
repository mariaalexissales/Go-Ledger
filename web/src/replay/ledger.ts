import type { Account } from '@/features/accounts/accounts.types'
import type { Transaction } from '@/features/transactions/transactions.types'
import { loadIndex } from './fixtures'

/**
 * An in-memory stand-in for the accounts and transactions tables, seeded from a
 * snapshot of the real seeded database.
 *
 * Writes work so the forms are not dead ends, but they live in this tab only and
 * vanish on reload. The UI says so rather than letting anyone think they just
 * wrote to a ledger.
 */
class ReplayLedger {
  private accounts: Account[] = []
  private transactions: Transaction[] = []
  private nextAccountId = 1
  private nextTransactionId = 1
  private ready: Promise<void> | null = null

  private async ensureLoaded(): Promise<void> {
    this.ready ??= loadIndex().then((index) => {
      this.accounts = index.accounts.map((account) => ({ ...account }))
      this.transactions = index.transactions.map((transaction) => ({ ...transaction }))
      this.nextAccountId = maxId(this.accounts) + 1
      this.nextTransactionId = maxId(this.transactions) + 1
    })

    return this.ready
  }

  async listAccounts(search: string): Promise<Account[]> {
    await this.ensureLoaded()

    const needle = search.trim().toLowerCase()
    const matches = needle
      ? this.accounts.filter((account) => account.name.toLowerCase().includes(needle))
      : this.accounts

    return [...matches].sort((a, b) => a.id - b.id)
  }

  async getAccount(id: number): Promise<Account | undefined> {
    await this.ensureLoaded()
    return this.accounts.find((account) => account.id === id)
  }

  async createAccount(name: string): Promise<Account> {
    await this.ensureLoaded()

    const account: Account = {
      id: this.nextAccountId++,
      name,
      balance: 0,
      created_at: new Date().toISOString(),
    }

    this.accounts.push(account)
    return account
  }

  async deleteAccount(id: number): Promise<boolean> {
    await this.ensureLoaded()

    const index = this.accounts.findIndex((account) => account.id === id)
    if (index === -1) return false

    this.accounts.splice(index, 1)
    // ON DELETE CASCADE, same as the schema.
    this.transactions = this.transactions.filter((transaction) => transaction.account_id !== id)
    return true
  }

  async listTransactions(accountId?: number): Promise<Transaction[]> {
    await this.ensureLoaded()

    const matches =
      accountId === undefined
        ? this.transactions
        : this.transactions.filter((transaction) => transaction.account_id === accountId)

    // Matches the server: newest first, id as the tiebreaker.
    return [...matches].sort(
      (a, b) =>
        new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime() || b.id - a.id,
    )
  }

  async getTransaction(id: number): Promise<Transaction | undefined> {
    await this.ensureLoaded()
    return this.transactions.find((transaction) => transaction.id === id)
  }

  async createTransaction(accountId: number, amount: number): Promise<Transaction | null> {
    await this.ensureLoaded()

    const account = this.accounts.find((candidate) => candidate.id === accountId)
    if (!account) return null // the foreign key violation, in miniature

    const transaction: Transaction = {
      id: this.nextTransactionId++,
      account_id: accountId,
      amount,
      timestamp: new Date().toISOString(),
    }

    this.transactions.push(transaction)
    account.balance = round2((account.balance ?? 0) + amount)
    return transaction
  }

  async deleteTransaction(id: number): Promise<boolean> {
    await this.ensureLoaded()

    const index = this.transactions.findIndex((transaction) => transaction.id === id)
    if (index === -1) return false

    this.transactions.splice(index, 1)
    return true
  }
}

function maxId(rows: { id: number }[]): number {
  return rows.reduce((max, row) => Math.max(max, row.id), 0)
}

// NUMERIC(15,2). Keep the same two-decimal shape the column enforces.
function round2(value: number): number {
  return Math.round(value * 100) / 100
}

export const replayLedger = new ReplayLedger()
