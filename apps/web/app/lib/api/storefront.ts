import type { Money } from "../money/money";
import { apiFetch } from "./client";

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

// Backend money shape (amount_minor) differs from the frontend's Money type
// (amount) — converted at this API boundary, same as lib/api/products.ts.
type MoneyDTO = { amount_minor: number; currency: string };

function fromMoneyDTO(dto: MoneyDTO): Money {
  return { amount: dto.amount_minor, currency: dto.currency };
}

export type NavCategory = {
  id: string;
  name: string;
  slug: string;
  image_url?: string;
  has_promotion?: boolean;
};

export type NavType = {
  id: string;
  name: string;
  slug: string;
  categories: NavCategory[];
};

function withLocale(params: URLSearchParams, locale?: string): URLSearchParams {
  if (locale && locale !== "en") params.set("locale", locale);
  return params;
}

export function getNav(locale?: string): Promise<NavType[]> {
  const query = withLocale(new URLSearchParams(), locale).toString();
  return apiFetch<NavType[]>(`/api/v1/storefront/nav${query ? `?${query}` : ""}`, { auth: false });
}

export type StorefrontProduct = {
  id: string;
  name: string;
  slug: string;
  description: string;
  base_price: Money;
  compare_at_price?: Money;
  promotion_price?: Money;
  promotion_label?: string;
  image_url?: string;
  in_stock: boolean;
  created_at: string;
};

type RawStorefrontProduct = Omit<StorefrontProduct, "base_price" | "compare_at_price" | "promotion_price"> & {
  base_price: MoneyDTO;
  compare_at_price?: MoneyDTO;
  promotion_price?: MoneyDTO;
};

function fromRawProduct(raw: RawStorefrontProduct): StorefrontProduct {
  return {
    ...raw,
    base_price: fromMoneyDTO(raw.base_price),
    compare_at_price: raw.compare_at_price ? fromMoneyDTO(raw.compare_at_price) : undefined,
    promotion_price: raw.promotion_price ? fromMoneyDTO(raw.promotion_price) : undefined,
  };
}

export type StorefrontProductFilters = {
  categoryIds?: string[];
  catalogId?: string;
  attributeValueIds?: string[];
  q?: string;
  limit?: number;
  locale?: string;
  productIds?: string[];
  hasPromotion?: boolean;
  // 1-based page. When set, the endpoint returns a single page (≤30 items)
  // plus the full match count in `total`.
  page?: number;
  pageSize?: number;
};

function buildProductFilterParams(filters: StorefrontProductFilters): URLSearchParams {
  const params = new URLSearchParams();
  for (const id of filters.categoryIds ?? []) params.append("category_id", id);
  if (filters.catalogId) params.set("catalog_id", filters.catalogId);
  for (const id of filters.attributeValueIds ?? []) params.append("attribute_value_id", id);
  if (filters.q) params.set("q", filters.q);
  if (filters.limit) params.set("limit", String(filters.limit));
  for (const id of filters.productIds ?? []) params.append("product_id", id);
  if (filters.hasPromotion) params.set("has_promotion", "true");
  if (filters.page) params.set("page", String(filters.page));
  if (filters.pageSize) params.set("page_size", String(filters.pageSize));
  return withLocale(params, filters.locale);
}

// The list endpoint always returns an envelope; `items` is the (possibly
// paginated) page and `total` is the full match count.
type RawStorefrontProductPage = {
  items: RawStorefrontProduct[];
  total: number;
};

export type StorefrontProductPage = {
  items: StorefrontProduct[];
  total: number;
};

// Paginated variant: pass `page` (1-based) to get one page plus the total
// match count, for rendering page controls.
export async function listStorefrontProductsPage(
  filters: StorefrontProductFilters = {},
): Promise<StorefrontProductPage> {
  const query = buildProductFilterParams(filters).toString();
  const raw = await apiFetch<RawStorefrontProductPage>(
    `/api/v1/storefront/products${query ? `?${query}` : ""}`,
    { auth: false },
  );
  return { items: (raw.items ?? []).map(fromRawProduct), total: raw.total ?? 0 };
}

// Convenience wrapper for callers that only need the items (curated home
// sections, recently-viewed, etc.) — unwraps the envelope.
export async function listStorefrontProducts(filters: StorefrontProductFilters = {}): Promise<StorefrontProduct[]> {
  return (await listStorefrontProductsPage(filters)).items;
}

export type StorefrontMedia = {
  id: string;
  url: string;
  alt_text?: string;
};

export type StorefrontAttributeRef = {
  id: string;
  name: string;
  type: "text" | "color";
};

export type StorefrontVariantAttributeValue = {
  id: string;
  attribute_id: string;
  value: string;
  color_hex?: string;
};

export type StorefrontVariant = {
  id: string;
  price_override?: Money;
  attributes: StorefrontVariantAttributeValue[];
  // Absent means no inventory item has been assigned yet — treated the
  // same as 0 (out of stock), since inventory tracking is per-variant.
  quantity_available?: number;
};

export type StorefrontProductDetail = StorefrontProduct & {
  media: StorefrontMedia[];
  attributes: StorefrontAttributeRef[];
  variants: StorefrontVariant[];
};

type RawStorefrontVariant = Omit<StorefrontVariant, "price_override"> & { price_override?: MoneyDTO };

type RawStorefrontProductDetail = Omit<RawStorefrontProduct, never> & {
  media?: StorefrontMedia[];
  attributes?: StorefrontAttributeRef[];
  variants?: RawStorefrontVariant[];
};

export async function getStorefrontProduct(slug: string, locale?: string): Promise<StorefrontProductDetail> {
  const query = withLocale(new URLSearchParams(), locale).toString();
  const raw = await apiFetch<RawStorefrontProductDetail>(
    `/api/v1/storefront/products/${encodeURIComponent(slug)}${query ? `?${query}` : ""}`,
    { auth: false },
  );
  return {
    ...fromRawProduct(raw),
    media: raw.media ?? [],
    attributes: raw.attributes ?? [],
    variants: (raw.variants ?? []).map((v) => ({
      ...v,
      price_override: v.price_override ? fromMoneyDTO(v.price_override) : undefined,
    })),
  };
}

export type AttributeFacet = {
  attribute_id: string;
  attribute_name: string;
  attribute_type: "text" | "color";
  values: { id: string; value: string; color_hex?: string }[];
};

export function listFacets(
  filters: { categoryIds?: string[]; catalogId?: string; locale?: string } = {},
): Promise<AttributeFacet[]> {
  const params = new URLSearchParams();
  for (const id of filters.categoryIds ?? []) params.append("category_id", id);
  if (filters.catalogId) params.set("catalog_id", filters.catalogId);
  const query = withLocale(params, filters.locale).toString();
  return apiFetch<AttributeFacet[]>(`/api/v1/storefront/facets${query ? `?${query}` : ""}`, { auth: false });
}

export type StorefrontCatalog = {
  id: string;
  name: string;
  slug: string;
};

export function listStorefrontCatalogs(locale?: string): Promise<StorefrontCatalog[]> {
  const query = withLocale(new URLSearchParams(), locale).toString();
  return apiFetch<StorefrontCatalog[]>(`/api/v1/storefront/catalogs${query ? `?${query}` : ""}`, { auth: false });
}

// image_url fields from the API are relative paths — this resolves them
// against the API origin so <img src> works regardless of where the
// frontend is served from.
export function resolveImageUrl(path: string): string {
  return `${API_BASE_URL}${path}`;
}

export type StorefrontStoreSettings = {
  store_name: string;
  legal_entity_name?: string;
  locale: string;
  currency: string;
  contact_email?: string;
  contact_phone?: string;
  company_description?: string;
  facebook_url?: string;
  instagram_url?: string;
  logo_url?: string;
};

export function getStoreSettings(): Promise<StorefrontStoreSettings> {
  return apiFetch<StorefrontStoreSettings>("/api/v1/storefront/store-settings", { auth: false });
}

export type StorefrontAddress = {
  id: string;
  label: string;
  line1: string;
  line2?: string;
  city?: string;
  region?: string;
  postal_code?: string;
  country?: string;
  is_default: boolean;
};

export function listStorefrontAddresses(): Promise<StorefrontAddress[]> {
  return apiFetch<StorefrontAddress[]>("/api/v1/storefront/store-settings/addresses", { auth: false });
}

export type DocumentType = "terms" | "privacy" | "faq" | "shipping";

// File-serving GET, not JSON — callers point an <a href> or fetch directly
// at the resolved URL.
export function storefrontDocumentUrl(type: DocumentType, locale?: string): string {
  const query = withLocale(new URLSearchParams(), locale).toString();
  return resolveImageUrl(`/api/v1/storefront/store-settings/documents/${type}/file${query ? `?${query}` : ""}`);
}

export type Language = {
  code: string;
  name: string;
  is_default: boolean;
  enabled: boolean;
};

export function listEnabledLanguages(): Promise<Language[]> {
  return apiFetch<Language[]>("/api/v1/storefront/languages", { auth: false });
}

export function getUiStrings(locale?: string): Promise<Record<string, string>> {
  const query = withLocale(new URLSearchParams(), locale).toString();
  return apiFetch<Record<string, string>>(`/api/v1/storefront/ui-strings${query ? `?${query}` : ""}`, {
    auth: false,
  });
}

export async function getBestInCategoryProducts(locale?: string): Promise<StorefrontProduct[]> {
  const query = withLocale(new URLSearchParams(), locale).toString();
  const raw = await apiFetch<RawStorefrontProduct[]>(
    `/api/v1/storefront/products/best-in-category${query ? `?${query}` : ""}`,
    { auth: false },
  );
  return raw.map(fromRawProduct);
}

export type CategoryProductGroup = {
  category_id: string;
  category_name: string;
  products: StorefrontProduct[];
};

type RawCategoryProductGroup = {
  category_id: string;
  category_name: string;
  products: RawStorefrontProduct[];
};

// Curated groups for the "Best in its category" section: one group per
// admin-selected category, each with up to `limit` hand-picked products.
export async function getCategoryProductGroups(
  section: string,
  opts: { locale?: string; limit?: number } = {},
): Promise<CategoryProductGroup[]> {
  const params = new URLSearchParams();
  params.set("section", section);
  if (opts.limit) params.set("limit", String(opts.limit));
  const query = withLocale(params, opts.locale).toString();
  const raw = await apiFetch<RawCategoryProductGroup[]>(
    `/api/v1/storefront/products/category-groups?${query}`,
    { auth: false },
  );
  return raw.map((g) => ({
    category_id: g.category_id,
    category_name: g.category_name,
    products: g.products.map(fromRawProduct),
  }));
}
