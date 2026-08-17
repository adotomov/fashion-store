// Canonical origin of the storefront. Used to build absolute URLs for
// canonical links, Open Graph, sitemap, and JSON-LD. Overridable at build time
// (VITE_SITE_URL) for preview environments; defaults to production.
export const SITE_URL = (import.meta.env.VITE_SITE_URL ?? "https://verani.bg").replace(/\/$/, "");

// Brand name shown in <title> suffixes, og:site_name, and Organization JSON-LD.
export const SITE_NAME = "Verani";

// Fallback description used on pages that don't supply their own.
export const DEFAULT_DESCRIPTION =
  "Verani — curated clothing, jewelry, bags, and accessories. Shop the latest arrivals with fast delivery across Bulgaria.";

// Absolute URL for a path on the storefront (leading slash optional).
export function absoluteUrl(path = "/"): string {
  if (/^https?:\/\//i.test(path)) return path;
  return `${SITE_URL}/${path.replace(/^\//, "")}`;
}
