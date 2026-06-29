import { clearToken, getToken, setToken } from "../auth/session";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";
const REFRESH_PATH = "/api/v1/auth/refresh";

export class ApiError extends Error {
  status: number;
  code?: string;

  constructor(status: number, message: string, code?: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

type RequestOptions = {
  method?: string;
  body?: unknown;
  auth?: boolean;
  headers?: Record<string, string>;
  /** internal: set when retrying after a refresh, to avoid infinite loops */
  _isRetry?: boolean;
};

type SessionResponse = {
  token: string;
  expires_at: string;
};

// Deduplicates concurrent refresh attempts: if several requests 401 at
// once, they all await the same in-flight refresh instead of each
// triggering their own.
let refreshPromise: Promise<string> | null = null;

async function refreshToken(currentToken: string): Promise<string> {
  if (!refreshPromise) {
    refreshPromise = (async () => {
      const response = await fetch(`${API_BASE_URL}${REFRESH_PATH}`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: currentToken }),
      });

      if (!response.ok) {
        clearToken();
        throw new ApiError(response.status, "session expired, please sign in again", "session_expired");
      }

      const session = (await response.json()) as SessionResponse;
      setToken(session.token);
      return session.token;
    })().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

export async function apiFetch<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, auth = true, _isRetry = false } = options;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...options.headers,
  };

  const token = auth ? getToken() : null;
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (response.status === 401 && auth && token && !_isRetry && path !== REFRESH_PATH) {
    try {
      await refreshToken(token);
    } catch {
      throw new ApiError(401, "session expired, please sign in again", "session_expired");
    }
    return apiFetch<T>(path, { ...options, _isRetry: true });
  }

  if (!response.ok) {
    let message = `Request failed with status ${response.status}`;
    let code: string | undefined;
    try {
      const data = await response.json();
      message = data?.error?.message ?? message;
      code = data?.error?.code;
    } catch {
      // response had no JSON body
    }
    throw new ApiError(response.status, message, code);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
