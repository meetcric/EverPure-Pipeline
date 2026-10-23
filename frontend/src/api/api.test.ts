import { afterEach, describe, expect, it, vi } from 'vitest'
import { ApiError, request } from './api'

function mockFetch(status: number, body?: unknown) {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(body === undefined ? null : JSON.stringify(body), { status }),
  )
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('request', () => {
  it('turns the API error envelope into an ApiError', async () => {
    mockFetch(409, { error: { code: 'invalid_transition', message: 'invalid status transition: passed → running' } })

    const err = await request('/runs/r1', { method: 'PATCH', body: { status: 'running' } }).catch((e: unknown) => e)

    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 409, code: 'invalid_transition', message: 'invalid status transition: passed → running' })
  })

  it('sends JSON to /api/v1 and returns undefined for 204', async () => {
    const fetchMock = mockFetch(204)

    await expect(request('/notes/n1', { method: 'DELETE' })).resolves.toBeUndefined()
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/notes/n1', expect.objectContaining({ method: 'DELETE' }))
  })

  it('sets Content-Type only when there is a body', async () => {
    const fetchMock = mockFetch(201, { id: 'p1' })

    await request('/pipelines', { method: 'POST', body: { name: 'x' } })

    const init = fetchMock.mock.calls[0][1] as RequestInit
    expect(init.body).toBe('{"name":"x"}')
    expect(init.headers).toMatchObject({ 'Content-Type': 'application/json' })
  })

  it('turns a network failure into a readable ApiError', async () => {
    vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('Failed to fetch')))

    const err = await request('/pipelines').catch((e: unknown) => e)

    expect(err).toMatchObject({ status: 0, code: 'network_error' })
  })
})
