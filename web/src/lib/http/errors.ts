export class ApiError extends Error {
  readonly status: number
  readonly retryAfterSec?: number

  constructor(status: number, message: string, retryAfterSec?: number) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.retryAfterSec = retryAfterSec
  }

  get isRateLimited() {
    return this.status === 429
  }
}

const STATUS_FALLBACK: Record<number, string> = {
  400: 'The request was rejected as invalid.',
  404: 'Not found.',
  405: 'That method is not allowed here.',
  409: 'That conflicts with something already in progress.',
  429: 'Rate limit exceeded.',
  500: 'The server hit an unexpected error.',
  502: 'The server is unreachable.',
  503: 'The server is unavailable.',
}

export async function toApiError(res: Response): Promise<ApiError> {
  const retryAfter = parseRetryAfter(res)

  let raw = ''
  try {
    raw = await res.text()
  } catch {}

  return new ApiError(res.status, extractMessage(raw, res.status), retryAfter)
}

function extractMessage(raw: string, status: number): string {
  const trimmed = raw.trim()

  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    try {
      const parsed = JSON.parse(trimmed) as { error?: unknown; message?: unknown }
      const message = parsed.error ?? parsed.message
      if (typeof message === 'string' && message) return message
    } catch {}
  }

  if (trimmed) {
    return trimmed.length > 200 ? `${trimmed.slice(0, 200)}…` : trimmed
  }

  return STATUS_FALLBACK[status] ?? `Request failed with status ${status}.`
}

function parseRetryAfter(res: Response): number | undefined {
  const header = res.headers.get('Retry-After')
  if (!header) return undefined

  const seconds = Number(header)
  return Number.isFinite(seconds) && seconds >= 0 ? seconds : undefined
}
