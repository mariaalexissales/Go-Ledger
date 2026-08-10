import { describe, expect, it } from 'vitest'
import { ApiError, toApiError } from './errors'

/**
 * The whole reason this module exists: the API surface does not speak one error
 * dialect. Anything that assumes it does breaks in the error path, which is
 * exactly where you cannot afford a second failure.
 */
function response(body: string, status: number, headers?: HeadersInit) {
  return new Response(body, { status, headers })
}

describe('toApiError', () => {
  it('reads the JSON envelope our handlers emit', async () => {
    const error = await toApiError(response('{"error":"account not found"}', 404))

    expect(error).toBeInstanceOf(ApiError)
    expect(error.status).toBe(404)
    expect(error.message).toBe('account not found')
  })

  it('falls back to plain text from anything in front of the server', async () => {
    const error = await toApiError(response('502 Bad Gateway', 502))

    expect(error.message).toBe('502 Bad Gateway')
  })

  it('uses a status-derived message when the body is empty', async () => {
    const error = await toApiError(response('', 405))

    expect(error.message).toBe('That method is not allowed here.')
  })

  it('parses Retry-After on a 429 and flags it as rate limited', async () => {
    const error = await toApiError(
      response('{"error":"rate limit exceeded"}', 429, { 'Retry-After': '42' }),
    )

    expect(error.isRateLimited).toBe(true)
    expect(error.retryAfterSec).toBe(42)
    expect(error.message).toBe('rate limit exceeded')
  })

  it('leaves retryAfterSec undefined when the header is absent or unparseable', async () => {
    expect((await toApiError(response('', 429))).retryAfterSec).toBeUndefined()
    expect(
      (await toApiError(response('', 429, { 'Retry-After': 'Wed, 21 Oct 2026 07:28:00 GMT' })))
        .retryAfterSec,
    ).toBeUndefined()
  })

  it('does not throw on malformed JSON', async () => {
    const error = await toApiError(response('{not json', 500))

    expect(error.status).toBe(500)
    expect(error.message).toBe('{not json')
  })

  it('caps a long body so an HTML error page cannot fill the UI', async () => {
    const error = await toApiError(response('x'.repeat(500), 500))

    expect(error.message).toHaveLength(201) // 200 chars plus the ellipsis
  })
})
