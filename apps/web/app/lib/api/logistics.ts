import { apiFetch } from "./client";

// Speedy location typeahead + office/locker lookups. These back the structured
// address form and the checkout office/locker pickers. Results are cached in
// memory per query (see memo) so re-typing or re-opening a form doesn't refetch
// — the server caches too, but this spares even the round-trip.

export type Site = {
  id: number;
  name: string;
  type?: string;
  municipality?: string;
  region?: string;
  post_code?: string;
};

export type NamedLocation = {
  id: number;
  name: string;
  type?: string;
};

export type Office = {
  id: string;
  name: string;
  type: string;
};

const cache = new Map<string, Promise<unknown>>();

function memo<T>(key: string, load: () => Promise<T>): Promise<T> {
  const hit = cache.get(key);
  if (hit) return hit as Promise<T>;
  const p = load().catch((err) => {
    // Don't cache failures — a transient error should be retryable.
    cache.delete(key);
    throw err;
  });
  cache.set(key, p);
  return p as Promise<T>;
}

/** Cities/towns matching a name fragment (Bulgaria only). */
export function searchSites(q: string): Promise<Site[]> {
  const query = q.trim();
  const params = new URLSearchParams({ q: query });
  return memo(`sites|${query.toLowerCase()}`, () =>
    apiFetch<Site[]>(`/api/v1/logistics/sites?${params.toString()}`, { auth: false }),
  );
}

/** Residential complexes (кв./жк.) within a site. */
export function searchComplexes(siteId: number, q: string): Promise<NamedLocation[]> {
  const query = q.trim();
  const params = new URLSearchParams({ siteId: String(siteId), q: query });
  return memo(`complexes|${siteId}|${query.toLowerCase()}`, () =>
    apiFetch<NamedLocation[]>(`/api/v1/logistics/complexes?${params.toString()}`, { auth: false }),
  );
}

/** Streets within a site. */
export function searchStreets(siteId: number, q: string): Promise<NamedLocation[]> {
  const query = q.trim();
  const params = new URLSearchParams({ siteId: String(siteId), q: query });
  return memo(`streets|${siteId}|${query.toLowerCase()}`, () =>
    apiFetch<NamedLocation[]>(`/api/v1/logistics/streets?${params.toString()}`, { auth: false }),
  );
}

/** Speedy offices (type "OFFICE") or EasyBox lockers ("APT") within a site. */
export function searchOffices(siteId: number, type: "OFFICE" | "APT", q = ""): Promise<Office[]> {
  const query = q.trim();
  const params = new URLSearchParams({ siteId: String(siteId), type, q: query });
  return memo(`offices|${type}|${siteId}|${query.toLowerCase()}`, () =>
    apiFetch<Office[]>(`/api/v1/logistics/offices?${params.toString()}`, { auth: false }),
  );
}
