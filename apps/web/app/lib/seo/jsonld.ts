import type { Money } from "../money/money";
import { type StorefrontProductDetail, resolveImageUrl } from "../api/storefront";
import { SITE_NAME, SITE_URL, absoluteUrl } from "./config";

// Money is stored in EUR minor units (see lib/money). Schema.org prices are
// decimal strings in major units; currency is always EUR (BGN is display-only).
function priceString(money: Money): string {
  return (money.amount / 100).toFixed(2);
}

// Organization — establishes the brand entity for knowledge-panel / logo use.
export function organizationJsonLd(): Record<string, unknown> {
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    name: SITE_NAME,
    url: SITE_URL,
    logo: absoluteUrl("/favicon.ico"),
  };
}

// WebSite + Sitelinks Search Box (maps to the storefront's /shop?q= search).
export function websiteJsonLd(): Record<string, unknown> {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    name: SITE_NAME,
    url: SITE_URL,
    potentialAction: {
      "@type": "SearchAction",
      target: {
        "@type": "EntryPoint",
        urlTemplate: `${SITE_URL}/shop?q={search_term_string}`,
      },
      "query-input": "required name=search_term_string",
    },
  };
}

// Product — drives price/availability rich results. Rendered client-side once
// the product has loaded (lean SEO mode has no server loader for this route).
export function productJsonLd(product: StorefrontProductDetail): Record<string, unknown> {
  const effectivePrice = product.promotion_price ?? product.base_price;
  const url = absoluteUrl(`/shop/${product.slug}`);
  const images = product.media.map((m) => resolveImageUrl(m.url)).filter(Boolean);

  return {
    "@context": "https://schema.org",
    "@type": "Product",
    name: product.name,
    ...(product.description ? { description: product.description } : {}),
    ...(images.length > 0 ? { image: images } : {}),
    sku: product.id,
    url,
    brand: { "@type": "Brand", name: SITE_NAME },
    offers: {
      "@type": "Offer",
      url,
      priceCurrency: effectivePrice.currency,
      price: priceString(effectivePrice),
      availability: product.in_stock ? "https://schema.org/InStock" : "https://schema.org/OutOfStock",
    },
  };
}

// BreadcrumbList — the trail shown under a result in Google.
export function breadcrumbJsonLd(items: { name: string; path: string }[]): Record<string, unknown> {
  return {
    "@context": "https://schema.org",
    "@type": "BreadcrumbList",
    itemListElement: items.map((item, index) => ({
      "@type": "ListItem",
      position: index + 1,
      name: item.name,
      item: absoluteUrl(item.path),
    })),
  };
}
