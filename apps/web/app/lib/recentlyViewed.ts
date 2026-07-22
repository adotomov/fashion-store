// Tracks the products a visitor has opened, most-recent-first, in localStorage.
// Purely browser-local (no backend, no account required) — the home page reads
// these IDs and re-fetches fresh product data so prices/stock stay current.
const STORAGE_KEY = "recently-viewed";

// Store a small buffer beyond what the section displays so the row still fills
// even if a couple of viewed products were since deleted or went out of stock.
const MAX_STORED = 12;

export function getRecentlyViewedIds(): string[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? (parsed.filter((v) => typeof v === "string") as string[]) : [];
  } catch {
    return [];
  }
}

export function addRecentlyViewed(productId: string): void {
  if (typeof window === "undefined" || !productId) return;
  try {
    const existing = getRecentlyViewedIds().filter((id) => id !== productId);
    const next = [productId, ...existing].slice(0, MAX_STORED);
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(next));
  } catch {
    // Storage unavailable (private mode, quota) — history just won't persist.
  }
}
