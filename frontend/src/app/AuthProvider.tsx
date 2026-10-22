import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import bridge from '@vkontakte/vk-bridge';
import { useQueryClient } from '@tanstack/react-query';
import { authApi } from '../shared/api/api';
import { clearSession, setAccessToken } from '../shared/api/client';
import { restoreVKSession } from '../shared/api/session';
import { mockAdmin, mockStudent } from '../shared/api/mock';
import type { User } from '../shared/api/types';

const USER_KEY = 'nsu-career-user';
const MOCK = import.meta.env.VITE_MOCK_MODE === 'true';

interface AuthContextValue {
  user: User | null;
  loading: boolean;
  error: string | null;
  loginDemo: (role: 'student' | 'admin') => void;
  logout: () => void;
  refreshUser: () => Promise<void>;
  updateNotificationConsent: (enabled: boolean) => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const saveUser = useCallback((nextUser: User) => {
    setUser(nextUser);
    sessionStorage.setItem(USER_KEY, JSON.stringify(nextUser));
  }, []);

  const logout = useCallback(() => {
    clearSession();
    queryClient.clear();
    setUser(null);
    setError(null);
    setLoading(false);
  }, [queryClient]);

  const refreshUser = useCallback(async () => {
    const current = await authApi.me();
    saveUser(current);
  }, [saveUser]);

  const updateNotificationConsent = useCallback((enabled: boolean) => {
    setUser((current) => {
      if (!current) return current;
      const next = { ...current, notification_consent: enabled };
      sessionStorage.setItem(USER_KEY, JSON.stringify(next));
      return next;
    });
  }, []);

  const loginDemo = useCallback((role: 'student' | 'admin') => {
    if (!MOCK) return;
    const demoUser = role === 'admin' ? mockAdmin : mockStudent;
    queryClient.clear();
    setAccessToken('local-mock-token');
    saveUser(demoUser);
    setError(null);
    setLoading(false);
  }, [saveUser, queryClient]);

  useEffect(() => {
    const handleExpired = () => {
      clearSession();
      queryClient.clear();
      setUser(null);
      setError('Сессия завершилась. Откройте приложение через VK ещё раз.');
    };
    window.addEventListener('career-auth-expired', handleExpired);
    return () => window.removeEventListener('career-auth-expired', handleExpired);
  }, [queryClient]);

  useEffect(() => {
    let cancelled = false;
    const initialize = async () => {
      if (MOCK) {
        const stored = sessionStorage.getItem(USER_KEY);
        if (stored) {
          try { setUser(JSON.parse(stored) as User); } catch { sessionStorage.removeItem(USER_KEY); }
        }
        setLoading(false);
        return;
      }

      try {
        await bridge.send('VKWebAppInit');
        const config = await bridge.send('VKWebAppGetConfig') as { scheme?: string };
        const scheme = config.scheme ?? '';
        if (/dark|space_gray/i.test(scheme)) document.documentElement.dataset.theme = 'dark';
        else if (/light|bright/i.test(scheme)) document.documentElement.dataset.theme = 'light';
      } catch {
        // VKWebAppInit/config can be unavailable in an ordinary browser; launch params are checked below.
      }

      try {
        const current = await restoreVKSession(window.location.search, () => cancelled);
        if (!cancelled && current) {
          saveUser(current);
          setError(null);
        }
      } catch (cause) {
        if (!cancelled) setError(cause instanceof Error ? cause.message : 'Не удалось проверить сессию.');
      } finally {
        if (!cancelled) setLoading(false);
      }
    };
    void initialize();
    return () => { cancelled = true; };
  }, [saveUser]);

  const value = useMemo(() => ({ user, loading, error, loginDemo, logout, refreshUser, updateNotificationConsent }), [user, loading, error, loginDemo, logout, refreshUser, updateNotificationConsent]);
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext);
  if (!context) throw new Error('useAuth must be used inside AuthProvider');
  return context;
}
