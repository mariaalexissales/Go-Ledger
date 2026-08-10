import { ApiError, toApiError } from './errors'
import { rateLimitStore } from '@/lib/rateLimit'
import { REPLAY } from '@/replay/mode'
import { replayRequest } from '@/replay/transport'

/**
 * Empty by default so every request is relative: the Vite proxy handles dev and
 * same-origin handles the container build. Set VITE_API_URL only to talk to a
 * server on another origin, which is exactly when CORS matters.
 */
const BASE_URL = import.meta.env.VITE_API_URL ?? ''

export function apiUrl(path: string, params?: QueryParams): string {
  const query = params ? buildQuery(params) : ''
  return `${BASE_URL}${path}${query}`
}

/**
 * Deliberately `object` rather than an index-signature type: the feature
 * modules pass their own typed param interfaces, and interfaces do not get
 * implicit index signatures in TypeScript.
 */
export type QueryParams = object

function buildQuery(params: QueryParams): string {
  const search = new URLSearchParams()

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    search.set(key, String(value))
  }

  const query = search.toString()
  return query ? `?${query}` : ''
}

interface RequestOptions {
  method?: string
  params?: QueryParams
  body?: unknown
  signal?: AbortSignal
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', params, body, signal } = options

  // The static build has no server to talk to; recorded fixtures answer instead.
  if (REPLAY) {
    return replayRequest<T>(method, path, params, body)
  }

  let res: Response
  try {
    res = await fetch(apiUrl(path, params), {
      method,
      signal,
      headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
      body: body === undefined ? undefined : JSON.stringify(body),
    })
  } catch (cause) {
    if (cause instanceof DOMException && cause.name === 'AbortError') throw cause
    throw new ApiError(0, 'Could not reach the server. Is it running?')
  }

  if (!res.ok) {
    const error = await toApiError(res)

    // Surface 429s globally: the banner and the disabled mutation buttons are
    // driven from one place rather than per-component error handling.
    if (error.isRateLimited) {
      rateLimitStore.trip(error.retryAfterSec ?? 30)
    }

    throw error
  }

  if (res.status === 204) return undefined as T

  return (await res.json()) as T
}

export const api = {
  get: <T>(path: string, params?: QueryParams, signal?: AbortSignal) =>
    request<T>(path, { params, signal }),
  post: <T>(path: string, body?: unknown, signal?: AbortSignal) =>
    request<T>(path, { method: 'POST', body, signal }),
  put: <T>(path: string, body?: unknown, signal?: AbortSignal) =>
    request<T>(path, { method: 'PUT', body, signal }),
  delete: <T>(path: string, signal?: AbortSignal) =>
    request<T>(path, { method: 'DELETE', signal }),
}

/** Shared envelope for every collection endpoint. */
export interface ListResponse<T> {
  data: T[]
  total: number
  limit: number
  offset: number
}
