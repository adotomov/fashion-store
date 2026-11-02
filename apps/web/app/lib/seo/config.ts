import { runtimeConfig } from "../config/runtime";

// Canonical origin of the storefront. Used to build absolute URLs for
// canonical links, Open Graph, sitemap, and JSON-LD. Set per environment with
// SITE_URL on the server (see lib/config/runtime.ts); defaults to production.
export const SITE_URL = runtimeConfig().siteUrl;

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
