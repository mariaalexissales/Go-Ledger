import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { rateLimitStore } from './rateLimit'

describe('rateLimitStore', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-08-11T10:00:00.000Z'))
    rateLimitStore.clear()
  })

  afterEach(() => {
    rateLimitStore.clear()
    vi.useRealTimers()
  })

  it('starts unlimited', () => {
    expect(rateLimitStore.getSnapshot().until).toBeNull()
  })

  it('records when the block lifts', () => {
    rateLimitStore.trip(30)
    expect(rateLimitStore.getSnapshot().until).toBe(Date.now() + 30_000)
  })

  it('clears itself once the block elapses', () => {
    rateLimitStore.trip(30)
    vi.advanceTimersByTime(29_999)
    expect(rateLimitStore.getSnapshot().until).not.toBeNull()

    vi.advanceTimersByTime(1)
    expect(rateLimitStore.getSnapshot().until).toBeNull()
  })

  it('extends a block but never shortens one', () => {
    rateLimitStore.trip(60)
    const until = rateLimitStore.getSnapshot().until

    rateLimitStore.trip(5)
    expect(rateLimitStore.getSnapshot().until).toBe(until)

    rateLimitStore.trip(120)
    expect(rateLimitStore.getSnapshot().until).toBe(Date.now() + 120_000)
  })

  it('replaces the timer when extending, so the block does not lift early', () => {
    rateLimitStore.trip(10)
    vi.advanceTimersByTime(5_000)

    rateLimitStore.trip(60)
    vi.advanceTimersByTime(5_000)
    expect(rateLimitStore.getSnapshot().until).not.toBeNull()

    vi.advanceTimersByTime(55_000)
    expect(rateLimitStore.getSnapshot().until).toBeNull()
  })

  it('notifies subscribers on change and stops after unsubscribe', () => {
    const listener = vi.fn()
    const unsubscribe = rateLimitStore.subscribe(listener)

    rateLimitStore.trip(30)
    expect(listener).toHaveBeenCalledTimes(1)

    rateLimitStore.clear()
    expect(listener).toHaveBeenCalledTimes(2)

    unsubscribe()
    rateLimitStore.trip(30)
    expect(listener).toHaveBeenCalledTimes(2)
  })

  it('returns a stable snapshot reference between changes', () => {
    const first = rateLimitStore.getSnapshot()
    expect(rateLimitStore.getSnapshot()).toBe(first)

    rateLimitStore.trip(30)
    expect(rateLimitStore.getSnapshot()).not.toBe(first)
  })

  it('clear is safe when no block is active', () => {
    expect(() => rateLimitStore.clear()).not.toThrow()
    expect(rateLimitStore.getSnapshot().until).toBeNull()
  })
})
