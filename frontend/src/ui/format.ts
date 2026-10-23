// UI helpers: formatting and labels. Presentation only, no API or cache knowledge.
import type { RunStatus } from '../types'

// --- Formatting ---

export function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' })
}

// 4320000 → "1h 12m", 75000 → "1m 15s", 900 → "1s"
export function formatDuration(ms: number): string {
  const total = Math.max(0, Math.round(ms / 1000))
  const h = Math.floor(total / 3600)
  const m = Math.floor((total % 3600) / 60)
  const s = total % 60
  if (h > 0) return `${h}h ${m}m`
  if (m > 0) return `${m}m ${s}s`
  return `${s}s`
}

export const shortSha = (sha: string) => sha.slice(0, 7)

// --- Run status ---

export const STATUS_LABELS: Record<RunStatus, string> = {
  queued: 'Queued',
  running: 'Running',
  passed: 'Passed',
  failed: 'Failed',
  flaky: 'Flaky',
}

// Mirrors the API's transition rules so the UI only enables moves the server accepts.
// The server is still the source of truth: if it rejects a move, the error is shown.
export const STATUS_ACTIONS: readonly { to: RunStatus; label: string; from: readonly RunStatus[] }[] = [
  { to: 'running', label: 'Start', from: ['queued'] },
  { to: 'passed', label: 'Pass', from: ['running'] },
  { to: 'failed', label: 'Fail', from: ['queued', 'running'] },
  { to: 'flaky', label: 'Mark flaky', from: ['running'] },
]

export const isFinished = (s: RunStatus) => s === 'passed' || s === 'failed' || s === 'flaky'

// The API layer turns every failure into an Error with a readable message.
export function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Something went wrong.'
}
