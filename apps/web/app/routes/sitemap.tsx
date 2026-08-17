import { listStorefrontProductsPage } from "../lib/api/storefront";
import { absoluteUrl } from "../lib/seo/config";

// Indexable static routes. Filtered/account/admin/cart paths are intentionally
// excluded (see robots.txt); the shop grid canonicalizes filters to /shop.
const STATIC_PATHS = ["/", "/shop", "/about", "/legal/terms", "/legal/privacy", "/help/faq", "/help/shipping"];

function xmlEscape(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;");
}

// Resource route: GET /sitemap.xml. Server-side enumerates every active
// product plus the static pages. Best-effort — if the API is unreachable we
// still emit the static routes rather than failing the response.
export async function loader() {
  const locs = STATIC_PATHS.map((p) => absoluteUrl(p));

  try {
    // No page/limit params → the storefront returns all active products.
    const { items } = await listStorefrontProductsPage({});
    for (const product of items) {
      locs.push(absoluteUrl(`/shop/${product.slug}`));
    }
  } catch {
    // API unavailable at render time — ship the static routes only.
  }

  const body =
    `<?xml version="1.0" encoding="UTF-8"?>\n` +
    `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n` +
    locs.map((loc) => `  <url><loc>${xmlEscape(loc)}</loc></url>`).join("\n") +
    `\n</urlset>\n`;

  return new Response(body, {
    headers: {
      "Content-Type": "application/xml; charset=utf-8",
      "Cache-Control": "public, max-age=3600",
    },
  });
}
