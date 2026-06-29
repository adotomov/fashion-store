const STORAGE_KEY = "fashion_store_session_token";

// MVP token storage: localStorage. Backend remains the source of truth for
// authorization on every request; this only affects UX (route guards).
export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(STORAGE_KEY);
}

export function setToken(token: string): void {
  window.localStorage.setItem(STORAGE_KEY, token);
}

export function clearToken(): void {
  window.localStorage.removeItem(STORAGE_KEY);
}
