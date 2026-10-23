import { describe, expect, it } from 'vitest'
import { formatDuration, shortSha } from './format'

describe('formatDuration', () => {
  it.each([
    [4_320_000, '1h 12m'], // seed run-0001
    [75_000, '1m 15s'],
    [900, '1s'],
    [0, '0s'],
    [-5, '0s'], // clock skew must not show a negative duration
  ])('%i ms → %s', (ms, want) => {
    expect(formatDuration(ms)).toBe(want)
  })
})

describe('shortSha', () => {
  it('keeps the first 7 characters', () => {
    expect(shortSha('9f1c2a7d3b4e5f60718293a4b5c6d7e8f9a0b1c2')).toBe('9f1c2a7')
  })
})
