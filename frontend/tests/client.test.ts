import { afterEach, describe, expect, it, vi } from 'vitest';
import { clearSession, request, setAccessToken, ApiClientError } from '../src/shared/api/client';
import { resumeAnalysisApi } from '../src/shared/api/api';

afterEach(() => {
  vi.unstubAllGlobals();
  clearSession();
});

describe('HTTP client', () => {
  it('sends the bearer token and decodes a JSON response', async () => {
    setAccessToken('test-access-token');
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({ id: 'abc' }), {
      status: 200, headers: { 'Content-Type': 'application/json' },
    }));
    vi.stubGlobal('fetch', fetchMock);

    const result = await request<{ id: string }>('/me');
    expect(result.id).toBe('abc');
    expect(fetchMock).toHaveBeenCalledOnce();
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(new Headers(init.headers).get('Authorization')).toBe('Bearer test-access-token');
  });


  it('sends explicit consent when creating an AI resume analysis', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      id: 'analysis-1', file_id: 'file-1', status: 'queued', attempt_count: 0,
      created_at: '2026-10-20T10:00:00Z', updated_at: '2026-10-20T10:00:00Z',
    }), { status: 202, headers: { 'Content-Type': 'application/json' } }));
    vi.stubGlobal('fetch', fetchMock);

    await resumeAnalysisApi.create('file-1', true);
    const [, init] = fetchMock.mock.calls[0] as [string, RequestInit];
    expect(JSON.parse(String(init.body))).toEqual({ file_id: 'file-1', external_ai_consent: true });
  });

  it('converts the backend error format to a typed error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'SLOT_UNAVAILABLE', message: 'Время занято', details: {} },
    }), { status: 409, headers: { 'Content-Type': 'application/json' } })));

    await expect(request('/consultations')).rejects.toMatchObject({
      name: 'ApiClientError', status: 409, code: 'SLOT_UNAVAILABLE', message: 'Время занято',
    } satisfies Partial<ApiClientError>);
  });

  it('does not try to parse a body for 204 responses', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })));
    await expect(request<void>('/consultations/abc/cancel', { method: 'POST' })).resolves.toBeUndefined();
  });

  it('dispatches a session-expired event after 401', async () => {
    const eventSpy = vi.fn();
    window.addEventListener('career-auth-expired', eventSpy);
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({ error: { code: 'INVALID_TOKEN', message: 'Invalid' } }), {
      status: 401, headers: { 'Content-Type': 'application/json' },
    })));
    await expect(request('/me')).rejects.toMatchObject({ status: 401, code: 'INVALID_TOKEN' });
    expect(eventSpy).toHaveBeenCalledOnce();
    window.removeEventListener('career-auth-expired', eventSpy);
  });
});
