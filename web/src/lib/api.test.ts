import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError } from './api'

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ok: true }))
  vi.stubGlobal('fetch', fetchMock)
})
afterEach(() => vi.unstubAllGlobals())

/** The URL and init of the most recent fetch call. */
function lastCall(): { url: string; init: RequestInit } {
  const [url, init] = fetchMock.mock.calls.at(-1) as [string, RequestInit]
  return { url, init }
}

describe('request basics', () => {
  it('sends credentialed requests to the /api base', async () => {
    await api.meta()
    const { url, init } = lastCall()
    expect(url).toBe('/api/meta')
    expect(init.credentials).toBe('include')
  })

  it('sets no JSON content type on bodyless requests', async () => {
    await api.meta()
    expect(lastCall().init.headers).toBeUndefined()
  })

  it('sends a JSON body with a JSON content type', async () => {
    await api.devLogin('carlos', 'ABCD2345')
    const { url, init } = lastCall()
    expect(url).toBe('/api/auth/dev-login')
    expect(init.method).toBe('POST')
    expect(init.headers).toEqual({ 'Content-Type': 'application/json' })
    expect(JSON.parse(init.body as string)).toEqual({ username: 'carlos', invite_code: 'ABCD2345' })
  })

  it('omits an empty invite code from the payload', async () => {
    await api.devLogin('carlos', '')
    expect(JSON.parse(lastCall().init.body as string)).toEqual({ username: 'carlos' })
  })

  it('returns the parsed payload on success', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ user: { username: 'carlos' }, unread_notifications: 3 }))
    await expect(api.me()).resolves.toEqual({ user: { username: 'carlos' }, unread_notifications: 3 })
  })

  it('resolves to undefined on 204 without reading a body', async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }))
    await expect(api.logout()).resolves.toBeUndefined()
  })
})

describe('error handling', () => {
  it('throws an ApiError carrying status, code and the server message', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'forbidden', message: 'Not yours.' }, 403))
    const err = await api.me().catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toBeInstanceOf(Error)
    expect(err).toMatchObject({ name: 'ApiError', status: 403, code: 'forbidden', message: 'Not yours.' })
  })

  it('falls back to a generic message when the error body is not JSON', async () => {
    fetchMock.mockResolvedValue(new Response('<html>Bad gateway</html>', { status: 502 }))
    const err = await api.me().catch((e: unknown) => e)
    expect(err).toMatchObject({ status: 502, code: 'unknown', message: 'Request failed with 502' })
  })

  it('falls back to defaults when the JSON error envelope is incomplete', async () => {
    fetchMock.mockResolvedValue(jsonResponse({}, 500))
    const err = await api.me().catch((e: unknown) => e)
    expect(err).toMatchObject({ status: 500, code: 'unknown', message: 'Request failed with 500' })
  })

  it('lets network failures reject untouched', async () => {
    fetchMock.mockRejectedValue(new TypeError('Failed to fetch'))
    await expect(api.me()).rejects.toThrow('Failed to fetch')
  })
})

describe('query string building', () => {
  it('adds no ? when there are no filters', async () => {
    await api.threads()
    expect(lastCall().url).toBe('/api/threads')
    await api.threads({})
    expect(lastCall().url).toBe('/api/threads')
  })

  it('skips undefined, null and empty values but keeps zero and false', async () => {
    await api.threads({ sort: 'new', category: '', tag: undefined, nothing: null, page: 0, limit: 20 })
    expect(lastCall().url).toBe('/api/threads?sort=new&page=0&limit=20')
  })

  it('url-encodes special characters in values', async () => {
    await api.threads({ q: 'a b&c=d/é' })
    const url = new URL(lastCall().url, 'http://localhost')
    expect(url.searchParams.get('q')).toBe('a b&c=d/é')
    expect([...url.searchParams.keys()]).toEqual(['q']) // the & did not start a new param
  })
})

describe('githubLoginURL', () => {
  it('returns the bare OAuth URL without an invite', () => {
    expect(api.githubLoginURL()).toBe('/api/auth/github')
  })

  it('appends an encoded invite code', () => {
    expect(api.githubLoginURL('AB CD&x')).toBe('/api/auth/github?invite=AB%20CD%26x')
  })
})
