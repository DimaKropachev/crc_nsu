import { authApi } from './api';
import { ApiClientError, clearSession, getAccessToken, setAccessToken } from './client';
import type { User } from './types';

export async function restoreVKSession(launchParams: string, isCancelled: () => boolean): Promise<User | null> {
  if (getAccessToken()) {
    try {
      return await authApi.me();
    } catch (cause) {
      if (!(cause instanceof ApiClientError) || cause.status !== 401) throw cause;
      if (isCancelled()) return null;
      clearSession();
    }
  }

  if (isCancelled()) return null;
  const launch = new URLSearchParams(launchParams);
  if (!launch.get('sign') || !launch.get('vk_user_id') || !launch.get('vk_app_id')) {
    throw new Error('Откройте мини-приложение из VK, чтобы выполнить вход.');
  }
  const result = await authApi.loginWithVK(launchParams.replace(/^\?/, ''));
  if (isCancelled()) return null;
  setAccessToken(result.access_token);
  return result.user;
}