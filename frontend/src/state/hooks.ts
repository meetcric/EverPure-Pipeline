// State layer: TanStack Query hooks (server data, cache, retries, idempotency keys) and the
// two hooks that own long-lived resources (the SSE connection and a ticking timer).
// The UI only uses these hooks; it never calls the API layer itself.
import { QueryClient, useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import * as api from '../api/api'
import {
  parseStatus,
  type CreateNoteInput,
  type CreatePipelineInput,
  type CreateRunInput,
  type RunStatus,
  type RunWithNotes,
} from '../types'

export function createQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: shouldRetry,
        staleTime: 5_000, // navigating back within 5s reuses the data instead of refetching
      },
    },
  })
}

// Retrying a 4xx can't succeed (bad input, missing run), so only network errors and 5xx are retried.
export function shouldRetry(failureCount: number, error: unknown): boolean {
  if (error instanceof api.ApiError && error.status >= 400 && error.status < 500) return false
  return failureCount < 2
}

// Cache keys. They are prefixes of each other on purpose: invalidating
// ['runs', pipelineId] refreshes every filtered list of that pipeline.
export const queryKeys = {
  pipelines: ['pipelines'] as const,
  pipeline: (id: string) => ['pipelines', id] as const,
  runs: (pipelineId: string) => ['runs', pipelineId] as const,
  runList: (pipelineId: string, status: RunStatus | undefined) => ['runs', pipelineId, status ?? 'all'] as const,
  run: (id: string) => ['run', id] as const,
}

// --- Pipelines ---

export function usePipelines() {
  return useQuery({
    queryKey: queryKeys.pipelines,
    // Passing the signal lets TanStack Query abort the request if the view unmounts first.
    queryFn: ({ signal }) => api.listPipelines(signal),
  })
}

export function usePipeline(id: string) {
  return useQuery({ queryKey: queryKeys.pipeline(id), queryFn: ({ signal }) => api.getPipeline(id, signal) })
}

// Sends an Idempotency-Key. Submitting the same input again (e.g. after a network error)
// reuses the key, so the API can't create a duplicate; different input gets a new key.
export function useCreatePipeline() {
  const qc = useQueryClient()
  const pending = useRef<{ input: string; key: string } | null>(null)
  return useMutation({
    mutationFn: (input: CreatePipelineInput) => {
      const signature = JSON.stringify(input)
      if (pending.current?.input !== signature) {
        pending.current = { input: signature, key: newIdempotencyKey() }
      }
      return api.createPipeline(input, pending.current.key)
    },
    onSuccess: () => {
      pending.current = null // done: the next pipeline is a new action
      return qc.invalidateQueries({ queryKey: queryKeys.pipelines })
    },
  })
}

function newIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2)}`
}

export function useRenamePipeline() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: string; name: string }) => api.renamePipeline(v.id, v.name),
    onSuccess: (pipeline) => {
      qc.setQueryData(queryKeys.pipeline(pipeline.id), pipeline)
      return qc.invalidateQueries({ queryKey: queryKeys.pipelines, exact: true })
    },
  })
}

export function useDeletePipeline() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deletePipeline(id),
    onSuccess: (_, id) => {
      // The API cascaded the delete to runs and notes; drop them from the cache too.
      qc.removeQueries({ queryKey: queryKeys.pipeline(id) })
      qc.removeQueries({ queryKey: queryKeys.runs(id) })
      return qc.invalidateQueries({ queryKey: queryKeys.pipelines, exact: true })
    },
  })
}

// --- Runs ---

export const RUNS_PAGE_SIZE = 20

// Cursor pagination: each page carries the cursor for the next one. The status filter is
// part of the key, so changing it starts a fresh list instead of reusing an old cursor.
export function useRunList(pipelineId: string, status: RunStatus | undefined) {
  return useInfiniteQuery({
    queryKey: queryKeys.runList(pipelineId, status),
    queryFn: ({ pageParam, signal }) =>
      api.listRuns(pipelineId, { status, cursor: pageParam || undefined, limit: RUNS_PAGE_SIZE }, signal),
    initialPageParam: '',
    getNextPageParam: (lastPage) => lastPage.nextCursor ?? undefined,
  })
}

export function useCreateRun(pipelineId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateRunInput) => api.createRun(pipelineId, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: queryKeys.runs(pipelineId) }),
  })
}

export function useRun(id: string) {
  return useQuery({ queryKey: queryKeys.run(id), queryFn: ({ signal }) => api.getRun(id, signal) })
}

export function useUpdateRunStatus(runId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (status: RunStatus) => api.updateRunStatus(runId, status),
    onSuccess: (run) => {
      // PATCH returns the run without notes: merge so the notes we already have stay.
      qc.setQueryData<RunWithNotes>(queryKeys.run(runId), (old) => (old ? { ...old, ...run } : old))
      return qc.invalidateQueries({ queryKey: queryKeys.runs(run.pipelineId) })
    },
    // A 409 means someone else moved the run first: reload it to show the real status.
    onError: () => qc.invalidateQueries({ queryKey: queryKeys.run(runId) }),
  })
}

// --- Notes (they live inside the cached run, so these update that entry) ---

export function useAddNote(runId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateNoteInput) => api.addNote(runId, input),
    onSuccess: (note) =>
      qc.setQueryData<RunWithNotes>(queryKeys.run(runId), (old) => (old ? { ...old, notes: [...old.notes, note] } : old)),
  })
}

export function useDeleteNote(runId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (noteId: string) => api.deleteNote(noteId),
    onSuccess: (_, noteId) =>
      qc.setQueryData<RunWithNotes>(queryKeys.run(runId), (old) =>
        old ? { ...old, notes: old.notes.filter((n) => n.id !== noteId) } : old,
      ),
    // e.g. 404 because it was already deleted elsewhere: reload the real list.
    onError: () => qc.invalidateQueries({ queryKey: queryKeys.run(runId) }),
  })
}

// --- Live status (SSE) ---

export type LiveState = 'connecting' | 'live' | 'reconnecting' | 'closed'

// useRunEvents keeps the cached run's status in sync with GET /runs/{id}/events while the
// component is mounted. The EventSource is closed in the effect cleanup, so navigating
// away never leaves a connection (or its listeners) behind.
export function useRunEvents(runId: string): LiveState {
  const qc = useQueryClient()
  const [state, setState] = useState<LiveState>('connecting')

  useEffect(() => {
    const source = new EventSource(api.runEventsUrl(runId))

    source.onopen = () => setState('live')
    // EventSource reconnects by itself after a dropped connection; CLOSED means it gave up (e.g. 404).
    source.onerror = () => setState(source.readyState === EventSource.CLOSED ? 'closed' : 'reconnecting')
    source.onmessage = (event: MessageEvent<string>) => {
      const status = parseEventStatus(event.data)
      if (!status) return

      let changed = false
      qc.setQueryData<RunWithNotes>(queryKeys.run(runId), (old) => {
        if (!old || old.status === status) return old
        changed = true
        return { ...old, status }
      })
      // The event only carries the status; finishedAt and durationMs come with a refetch.
      if (changed) void qc.invalidateQueries({ queryKey: queryKeys.run(runId) })
    }

    return () => source.close()
  }, [runId, qc])

  return state
}

function parseEventStatus(data: string): RunStatus | undefined {
  try {
    return parseStatus((JSON.parse(data) as { status?: unknown }).status)
  } catch {
    return undefined
  }
}

// --- Timer ---

// useNow returns the current time and re-renders every intervalMs while enabled.
// The interval is cleared when disabled or on unmount, so no timer outlives the view.
export function useNow(enabled: boolean, intervalMs = 1000): number {
  const [now, setNow] = useState(() => Date.now())

  useEffect(() => {
    if (!enabled) return
    const id = window.setInterval(() => setNow(Date.now()), intervalMs)
    return () => window.clearInterval(id)
  }, [enabled, intervalMs])

  return now
}
