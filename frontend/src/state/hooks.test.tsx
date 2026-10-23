import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { RunWithNotes } from '../types'
import { queryKeys, useCreatePipeline, useRunEvents } from './hooks'

// A stand-in for the browser's EventSource that records every instance, so the test can
// push messages and check that each one was closed.
class FakeEventSource {
  static readonly CLOSED = 2
  static instances: FakeEventSource[] = []
  readonly url: string
  readyState = 0
  closed = false
  onopen: (() => void) | null = null
  onerror: (() => void) | null = null
  onmessage: ((event: MessageEvent<string>) => void) | null = null

  constructor(url: string) {
    this.url = url
    FakeEventSource.instances.push(this)
  }

  close() {
    this.closed = true
    this.readyState = FakeEventSource.CLOSED
  }

  emit(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent<string>)
  }
}

const run: RunWithNotes = {
  id: 'run-1',
  pipelineId: 'pl-1',
  status: 'queued',
  startedAt: '2026-10-10T01:00:00Z',
  triggeredBy: 'ci',
  commitSha: 'abcd',
  notes: [],
}

function setup() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  client.setQueryData(queryKeys.run(run.id), run)
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  )
  return { client, wrapper }
}

beforeEach(() => {
  FakeEventSource.instances = []
  vi.stubGlobal('EventSource', FakeEventSource)
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('useRunEvents', () => {
  it('updates the cached status when an event arrives', () => {
    const { client, wrapper } = setup()
    renderHook(() => useRunEvents(run.id), { wrapper })

    const source = FakeEventSource.instances[0]
    expect(source.url).toBe('/api/v1/runs/run-1/events')
    act(() => source.emit({ status: 'running' }))

    expect(client.getQueryData<RunWithNotes>(queryKeys.run(run.id))?.status).toBe('running')
  })

  it('ignores malformed events', () => {
    const { client, wrapper } = setup()
    renderHook(() => useRunEvents(run.id), { wrapper })

    act(() => FakeEventSource.instances[0].emit({ status: 'banana' }))

    expect(client.getQueryData<RunWithNotes>(queryKeys.run(run.id))?.status).toBe('queued')
  })

  // Mirrors the harness: open and leave the run page 50 times; no connection may stay open.
  it('closes the stream on unmount, every time', () => {
    const { wrapper } = setup()
    for (let i = 0; i < 50; i++) {
      const { unmount } = renderHook(() => useRunEvents(run.id), { wrapper })
      unmount()
    }

    expect(FakeEventSource.instances).toHaveLength(50)
    expect(FakeEventSource.instances.every((s) => s.closed)).toBe(true)
  })
})

describe('useCreatePipeline', () => {
  it('reuses the Idempotency-Key for the same input and makes a new one for different input', async () => {
    const keys: string[] = []
    vi.stubGlobal(
      'fetch',
      vi.fn((_url: string, init: RequestInit) => {
        keys.push((init.headers as Record<string, string>)['Idempotency-Key'])
        // First call: the network drops. Later calls: the API answers.
        if (keys.length === 1) return Promise.reject(new TypeError('Failed to fetch'))
        return Promise.resolve(new Response(JSON.stringify({ id: 'p1' }), { status: 201 }))
      }),
    )
    const { wrapper } = setup()
    const { result } = renderHook(() => useCreatePipeline(), { wrapper })
    const input = { name: 'Nightly', product: 'flasharray' } as const

    await act(() => result.current.mutateAsync(input).catch(() => undefined)) // network error
    await act(() => result.current.mutateAsync(input)) // retry of the same action
    await act(() => result.current.mutateAsync({ ...input, name: 'Other' })) // a new action

    expect(keys[1]).toBe(keys[0])
    expect(keys[2]).not.toBe(keys[0])
  })
})
