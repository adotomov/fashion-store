const STORAGE_KEY = "fashion_store_cart_token";

// Identifies an anonymous cart so it can be used without registration. The
// backend issues this token the first time a guest adds an item; it's then
// persisted here and sent back via the X-Cart-Token header on every cart
// request until the guest logs in, at which point it's merged into their
// user cart and cleared.
export function getCartToken(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(STORAGE_KEY);
}

export function setCartToken(token: string): void {
  window.localStorage.setItem(STORAGE_KEY, token);
}

export function clearCartToken(): void {
  window.localStorage.removeItem(STORAGE_KEY);
}
