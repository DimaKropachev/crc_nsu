import type { ApiErrorBody } from './types';

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '');
const TOKEN_KEY = 'nsu-career-access-token';

export class ApiClientError extends Error {
  public readonly status: number;
  public readonly code: string;
  public readonly details: Record<string, unknown> | null;

  constructor(status: number, code: string, message: string, details?: Record<string, unknown> | null) {
    super(message);
    this.name = 'ApiClientError';
    this.status = status;
    this.code = code;
    this.details = details ?? null;
  }
}

export function setAccessToken(token: string | null): void {
  if (token) sessionStorage.setItem(TOKEN_KEY, token);
  else sessionStorage.removeItem(TOKEN_KEY);
}

export function getAccessToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY);
}

export function clearSession(): void {
  setAccessToken(null);
  sessionStorage.removeItem('nsu-career-user');
}

async function parseError(response: Response): Promise<ApiClientError> {
  let body: ApiErrorBody | undefined;
  try {
    body = (await response.json()) as ApiErrorBody;
  } catch {
    // Some HTTP intermediaries return plain text or an empty body.
  }
  const detail = body?.error;
  const fallbackCode = response.status === 401 ? 'INVALID_TOKEN' : 'HTTP_ERROR';
  return new ApiClientError(
    response.status,
    detail?.code || fallbackCode,
    detail?.message || `Не удалось выполнить запрос (HTTP ${response.status})`,
    detail?.details,
  );
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (!(init.body instanceof FormData) && init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json; charset=utf-8');
  }
  const token = getAccessToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);

  let response: Response;
  try {
    response = await fetch(`${API_BASE_URL}/api/v1${path.startsWith('/') ? path : `/${path}`}`, {
      ...init,
      headers,
      credentials: 'omit',
    });
  } catch {
    throw new ApiClientError(0, 'NETWORK_ERROR', 'Не удалось связаться с сервером. Проверьте подключение и адрес API.');
  }

  if (response.status === 401) {
    window.dispatchEvent(new CustomEvent('career-auth-expired'));
  }
  if (!response.ok) throw await parseError(response);
  if (response.status === 204) return undefined as T;
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) return (await response.json()) as T;
  return (await response.blob()) as T;
}

export async function downloadFile(path: string, filename: string): Promise<void> {
  const blob = await request<Blob>(path);
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}

export const jsonBody = (data: unknown): string => JSON.stringify(data);
export const apiBaseUrl = API_BASE_URL;
