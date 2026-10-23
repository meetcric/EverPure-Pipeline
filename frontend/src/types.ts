// Domain types shared by every layer (like the backend's model package). Imports nothing.

export const PRODUCTS = ['flasharray', 'flashblade', 'hyperscale'] as const
export type Product = (typeof PRODUCTS)[number]

export const RUN_STATUSES = ['queued', 'running', 'passed', 'failed', 'flaky'] as const
export type RunStatus = (typeof RUN_STATUSES)[number]

export function parseStatus(value: unknown): RunStatus | undefined {
  return RUN_STATUSES.find((s) => s === value)
}

export interface Pipeline {
  id: string
  name: string
  product: Product
  createdAt: string
}

export interface Run {
  id: string
  pipelineId: string
  status: RunStatus
  startedAt: string
  finishedAt?: string // only once the run has finished
  durationMs?: number
  triggeredBy: string
  commitSha: string
}

export interface Note {
  id: string
  runId: string
  author: string
  body: string
  createdAt: string
}

export interface RunWithNotes extends Run {
  notes: Note[]
}

export interface RunPage {
  items: Run[]
  nextCursor: string | null // null = last page
}

export interface CreatePipelineInput {
  name: string
  product: Product
}

export interface CreateRunInput {
  triggeredBy: string
  commitSha: string
}

export interface CreateNoteInput {
  author: string
  body: string
}
