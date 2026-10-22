import { afterEach, describe, expect, it, vi } from 'vitest';
import { clearSession, getAccessToken, setAccessToken } from '../src/shared/api/client';
import { restoreVKSession } from '../src/shared/api/session';

const launch = '?vk_app_id=123&vk_user_id=456&sign=signed-value';
const user = { id: 'user-1' };
const json = (value: unknown, status = 200) => new Response(JSON.stringify(value), {
  status, headers: { 'Content-Type': 'application/json' },
});

afterEach(() => {
  vi.unstubAllGlobals();
  clearSession();
});

describe('VK session restoration', () => {
  it('reuses a valid token without requiring launch parameters', async () => {
    setAccessToken('valid-token');
    const fetchMock = vi.fn().mockResolvedValue(json(user));
    vi.stubGlobal('fetch', fetchMock);
    await expect(restoreVKSession('', () => false)).resolves.toEqual(user);
    expect(fetchMock).toHaveBeenCalledOnce();
    expect(getAccessToken()).toBe('valid-token');
  });

  it('replaces an expired token using the original signed launch query', async () => {
    setAccessToken('expired-token');
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ error: { code: 'INVALID_TOKEN', message: 'Expired' } }, 401))
      .mockResolvedValueOnce(json({ access_token: 'new-token', user }));
    vi.stubGlobal('fetch', fetchMock);
    await expect(restoreVKSession(launch, () => false)).resolves.toEqual(user);
    expect(getAccessToken()).toBe('new-token');
    expect(JSON.parse(String(fetchMock.mock.calls[1][1].body))).toEqual({ launch_params: launch.slice(1) });
  });

  it('preserves the token on a server error without retrying login', async () => {
    setAccessToken('existing-token');
    const fetchMock = vi.fn().mockResolvedValue(json({ error: { code: 'UNAVAILABLE', message: 'Unavailable' } }, 503));
    vi.stubGlobal('fetch', fetchMock);
    await expect(restoreVKSession(launch, () => false)).rejects.toMatchObject({ status: 503 });
    expect(getAccessToken()).toBe('existing-token');
    expect(fetchMock).toHaveBeenCalledOnce();
  });

  it('does not store a login response after cancellation', async () => {
    let cancelled = false;
    vi.stubGlobal('fetch', vi.fn().mockImplementation(async () => {
      cancelled = true;
      return json({ access_token: 'new-token', user });
    }));
    await expect(restoreVKSession(launch, () => cancelled)).resolves.toBeNull();
    expect(getAccessToken()).toBeNull();
  });
});