import type { MetaDescriptor } from "react-router";

import { DEFAULT_DESCRIPTION, SITE_NAME, absoluteUrl } from "./config";

export type BuildMetaInput = {
  /** Page title, without the brand suffix (e.g. "Summer Dresses"). */
  title?: string;
  /** Meta description / og:description. Falls back to the site default. */
  description?: string;
  /** Route pathname used to build the canonical + og:url (query stripped). */
  path: string;
  /** Absolute image URL for og:image / twitter:image. */
  image?: string;
  /** Open Graph object type. */
  type?: "website" | "product" | "article";
  /** Emit noindex,nofollow (used for cart/checkout/account/admin/etc.). */
  noindex?: boolean;
  /**
   * Use `title` verbatim as the document title instead of appending
   * " | <SITE_NAME>". Home page passes its own fully-formed title.
   */
  titleAsIs?: boolean;
  /** Structured data rendered server-side as <script type="application/ld+json">. */
  jsonLd?: Record<string, unknown> | Record<string, unknown>[];
};

// Central builder for a route's <head> tags: title, description, canonical,
// Open Graph, Twitter card, robots, and optional JSON-LD. Every indexable
// public route funnels through this so the tag set stays consistent.
export function buildMeta(input: BuildMetaInput): MetaDescriptor[] {
  const { title, description, path, image, type = "website", noindex = false, titleAsIs = false } = input;

  const fullTitle = !title ? SITE_NAME : titleAsIs ? title : `${title} | ${SITE_NAME}`;
  const desc = (description ?? DEFAULT_DESCRIPTION).trim();
  // Canonical always drops the query string so filtered/paginated variants
  // consolidate onto the clean path.
  const canonical = absoluteUrl(path.split("?")[0]);

  const tags: MetaDescriptor[] = [
    { title: fullTitle },
    { name: "description", content: desc },
    { tagName: "link", rel: "canonical", href: canonical },

    { property: "og:title", content: fullTitle },
    { property: "og:description", content: desc },
    { property: "og:type", content: type },
    { property: "og:url", content: canonical },
    { property: "og:site_name", content: SITE_NAME },

    { name: "twitter:card", content: image ? "summary_large_image" : "summary" },
    { name: "twitter:title", content: fullTitle },
    { name: "twitter:description", content: desc },
  ];

  if (image) {
    tags.push({ property: "og:image", content: image });
    tags.push({ name: "twitter:image", content: image });
  }

  if (noindex) {
    tags.push({ name: "robots", content: "noindex, nofollow" });
  }

  const jsonLd = input.jsonLd;
  if (jsonLd) {
    for (const block of Array.isArray(jsonLd) ? jsonLd : [jsonLd]) {
      tags.push({ "script:ld+json": block });
    }
  }

  return tags;
}
