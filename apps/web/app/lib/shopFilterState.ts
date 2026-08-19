// Persists the /shop page's filter selections across navigation (e.g. to a
// product detail page and back) without ever touching the URL — filtering
// is meant to feel like a single-page panel, not a sequence of navigations.
// Session-scoped only: a fresh tab/window starts clean.
const STORAGE_KEY = "shop-filter-state";

export type ShopFilterState = {
  typeSlugs: string[];
  categoryIds: string[];
  catalogId?: string;
  attributeValueIds: string[];
  onSale?: boolean;
  // The grid page the shopper was last on, so returning from a product detail
  // page lands them back where they were rather than resetting to page 1.
  page?: number;
};

export function loadShopFilterState(): ShopFilterState | null {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY);
    return raw ? (JSON.parse(raw) as ShopFilterState) : null;
  } catch {
    return null;
  }
}

export function saveShopFilterState(state: ShopFilterState): void {
  try {
    sessionStorage.setItem(STORAGE_KEY, JSON.stringify(state));
  } catch {
    // Storage unavailable (private browsing, quota) — filters just won't
    // survive navigation, which is a safe degradation.
  }
}
