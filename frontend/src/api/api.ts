// API layer: one function per endpoint. The only code that knows URLs or calls fetch.
import type {
  CreateNoteInput,
  CreatePipelineInput,
  CreateRunInput,
  Note,
  Pipeline,
  Run,
  RunPage,
  RunStatus,
  RunWithNotes,
} from '../types'

// Relative on purpose: in Docker nginx serves the app and proxies /api to the Go API,
// and `npm run dev` does the same. The browser only ever talks to one origin.
export const API_BASE = '/api/v1'

// ApiError carries the API's {"error": {"code", "message"}} so the UI can show it.
export class ApiError extends Error {
  readonly status: number
  readonly code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
  }
}

interface RequestOptions {
  method?: string
  body?: unknown
  headers?: Record<string, string>
  signal?: AbortSignal // lets TanStack Query cancel the request when the view goes away
}

// request is the only function that calls fetch: one place for the base URL, JSON
// encoding and turning error responses into ApiError.
export async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json', ...options.headers }
  let body: string | undefined
  if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(options.body)
  }

  let res: Response
  try {
    res = await fetch(API_BASE + path, { method: options.method ?? 'GET', headers, body, signal: options.signal })
  } catch (err) {
    // An abort is TanStack Query cancelling the request (the view went away): pass it on as is.
    if (err instanceof DOMException && err.name === 'AbortError') throw err
    throw new ApiError(0, 'network_error', 'Could not reach the server. Check your connection and try again.')
  }
  if (res.status === 204) {
    return undefined as T
  }

  const data: unknown = await res.json().catch(() => null)
  if (!res.ok) {
    const err = (data as { error?: { code?: string; message?: string } } | null)?.error
    throw new ApiError(res.status, err?.code ?? 'http_error', err?.message ?? `Request failed (${res.status})`)
  }
  return data as T
}

// Path segments are encoded so an id can never change the URL's shape.
function path(...segments: string[]): string {
  return segments.map((s) => `/${encodeURIComponent(s)}`).join('')
}

// --- Pipelines ---

export const listPipelines = (signal?: AbortSignal) => request<Pipeline[]>('/pipelines', { signal })

export const getPipeline = (id: string, signal?: AbortSignal) => request<Pipeline>(path('pipelines', id), { signal })

// The same key on a retry makes the API replay the first response instead of creating a second pipeline.
export const createPipeline = (input: CreatePipelineInput, idempotencyKey: string) =>
  request<Pipeline>('/pipelines', { method: 'POST', body: input, headers: { 'Idempotency-Key': idempotencyKey } })

export const renamePipeline = (id: string, name: string) =>
  request<Pipeline>(path('pipelines', id), { method: 'PATCH', body: { name } })

export const deletePipeline = (id: string) => request<void>(path('pipelines', id), { method: 'DELETE' })

// --- Runs ---

export function listRuns(
  pipelineId: string,
  params: { status?: RunStatus; cursor?: string; limit?: number },
  signal?: AbortSignal,
) {
  const query = new URLSearchParams()
  if (params.status) query.set('status', params.status)
  if (params.cursor) query.set('cursor', params.cursor)
  if (params.limit) query.set('limit', String(params.limit))
  const qs = query.toString()
  return request<RunPage>(`${path('pipelines', pipelineId, 'runs')}${qs ? `?${qs}` : ''}`, { signal })
}

export const getRun = (id: string, signal?: AbortSignal) => request<RunWithNotes>(path('runs', id), { signal })

export const createRun = (pipelineId: string, input: CreateRunInput) =>
  request<Run>(path('pipelines', pipelineId, 'runs'), { method: 'POST', body: input })

export const updateRunStatus = (id: string, status: RunStatus) =>
  request<Run>(path('runs', id), { method: 'PATCH', body: { status } })

// URL of the run's Server-Sent Events stream, for EventSource.
export const runEventsUrl = (id: string) => API_BASE + path('runs', id, 'events')

// --- Notes ---

export const addNote = (runId: string, input: CreateNoteInput) =>
  request<Note>(path('runs', runId, 'notes'), { method: 'POST', body: input })

export const deleteNote = (id: string) => request<void>(path('notes', id), { method: 'DELETE' })
