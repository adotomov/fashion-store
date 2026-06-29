import { apiFetch } from "./client";
import { getToken } from "../auth/session";
import type { Money } from "../money/money";

export type ProductStatus = "draft" | "active" | "archived";

// Backend money shape (amount_minor) differs from the frontend's Money type
// (amount) used by storefront components — converted at this API boundary.
type MoneyDTO = { amount_minor: number; currency: string };

function fromMoneyDTO(dto: MoneyDTO): Money {
  return { amount: dto.amount_minor, currency: dto.currency };
}

function toMoneyDTO(money: Money): MoneyDTO {
  return { amount_minor: money.amount, currency: money.currency };
}

export type AttributeValueRef = {
  id: string;
  attribute_id: string;
  value: string;
};

// A product references attributes by name only (e.g. [Size, Color]) — no
// value yet. Variants are what carry actual values per attribute.
export type AttributeRef = {
  id: string;
  name: string;
};

export type ProductVariant = {
  id: string;
  product_id: string;
  price_override?: Money;
  attribute_value_ids: string[];
  attributes: AttributeValueRef[];
  // Absent means no SKU/inventory item has been assigned to this variant yet.
  inventory_item_id?: string;
  quantity_available?: number;
  created_at: string;
  updated_at: string;
};

export type ProductMedia = {
  id: string;
  product_id: string;
  bucket: string;
  object_key: string;
  content_type: string;
  size_bytes: number;
  position: number;
  alt_text: string;
  created_at: string;
};

export type Product = {
  id: string;
  name: string;
  slug: string;
  description: string;
  status: ProductStatus;
  base_price: Money;
  category_ids?: string[];
  catalog_ids?: string[];
  attributes?: AttributeRef[];
  variant_count: number;
  variants?: ProductVariant[];
  media?: ProductMedia[];
  created_at: string;
  updated_at: string;
};

type RawProduct = Omit<Product, "base_price"> & { base_price: MoneyDTO };
type RawVariant = Omit<ProductVariant, "price_override"> & { price_override?: MoneyDTO };

function fromRawProduct(raw: RawProduct): Product {
  return { ...raw, base_price: fromMoneyDTO(raw.base_price) };
}

function fromRawVariant(raw: RawVariant): ProductVariant {
  return { ...raw, price_override: raw.price_override ? fromMoneyDTO(raw.price_override) : undefined };
}

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export async function listProducts(): Promise<Product[]> {
  const raw = await apiFetch<RawProduct[]>("/api/v1/admin/products");
  return raw.map(fromRawProduct);
}

export async function getProduct(id: string): Promise<Product> {
  const raw = await apiFetch<RawProduct>(`/api/v1/admin/products/${id}`);
  return fromRawProduct(raw);
}

export async function createProduct(name: string): Promise<Product> {
  const raw = await apiFetch<RawProduct>("/api/v1/admin/products", { method: "POST", body: { name } });
  return fromRawProduct(raw);
}

export async function updateProduct(
  id: string,
  input: Partial<{ name: string; description: string; status: ProductStatus; base_price: Money }>,
): Promise<Product> {
  const body: Record<string, unknown> = { ...input };
  if (input.base_price) {
    body.base_price = toMoneyDTO(input.base_price);
  }
  const raw = await apiFetch<RawProduct>(`/api/v1/admin/products/${id}`, { method: "PATCH", body });
  return fromRawProduct(raw);
}

export function deleteProduct(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/products/${id}`, { method: "DELETE" });
}

export function setProductCategories(id: string, categoryIds: string[]): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/products/${id}/categories`, {
    method: "PUT",
    body: { category_ids: categoryIds },
  });
}

export function setProductCatalogs(id: string, catalogIds: string[]): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/products/${id}/catalogs`, {
    method: "PUT",
    body: { catalog_ids: catalogIds },
  });
}

export function setProductAttributes(id: string, attributeIds: string[]): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/products/${id}/attributes`, {
    method: "PUT",
    body: { attribute_ids: attributeIds },
  });
}

export async function createVariant(
  productId: string,
  attributeValueIds: string[],
  priceOverride?: Money,
): Promise<ProductVariant> {
  const raw = await apiFetch<RawVariant>(`/api/v1/admin/products/${productId}/variants`, {
    method: "POST",
    body: {
      attribute_value_ids: attributeValueIds,
      price_override: priceOverride ? toMoneyDTO(priceOverride) : undefined,
    },
  });
  return fromRawVariant(raw);
}

export async function updateVariant(
  productId: string,
  variantId: string,
  input: { attributeValueIds: string[]; priceOverride?: Money; clearPriceOverride?: boolean },
): Promise<ProductVariant> {
  const raw = await apiFetch<RawVariant>(`/api/v1/admin/products/${productId}/variants/${variantId}`, {
    method: "PATCH",
    body: {
      attribute_value_ids: input.attributeValueIds,
      price_override: input.priceOverride ? toMoneyDTO(input.priceOverride) : undefined,
      clear_price_override: input.clearPriceOverride ?? false,
    },
  });
  return fromRawVariant(raw);
}

export function deleteVariant(productId: string, variantId: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/products/${productId}/variants/${variantId}`, { method: "DELETE" });
}

// Media upload is multipart (real file bytes), so it bypasses apiFetch's
// JSON-only body handling and attaches the auth header manually.
export async function uploadProductMedia(
  productId: string,
  file: File,
  position: number,
  altText: string,
): Promise<ProductMedia> {
  const token = getToken();
  const form = new FormData();
  form.append("file", file);
  form.append("position", String(position));
  form.append("alt_text", altText);

  const response = await fetch(`${API_BASE_URL}/api/v1/admin/products/${productId}/media`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    body: form,
  });

  if (!response.ok) {
    throw new Error(`Upload failed with status ${response.status}`);
  }

  return (await response.json()) as ProductMedia;
}

export function updateMedia(
  productId: string,
  mediaId: string,
  input: Partial<{ position: number; alt_text: string }>,
): Promise<ProductMedia> {
  return apiFetch<ProductMedia>(`/api/v1/admin/products/${productId}/media/${mediaId}`, {
    method: "PATCH",
    body: input,
  });
}

export function deleteMedia(productId: string, mediaId: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/products/${productId}/media/${mediaId}`, { method: "DELETE" });
}

// The serve-media endpoint is admin-gated (Bearer token), and <img src>
// can't attach custom headers, so callers must fetch the bytes via JS and
// use the resulting blob URL as the <img> source. Caller is responsible
// for calling URL.revokeObjectURL when done (e.g. on unmount).
export async function loadMediaBlobUrl(productId: string, mediaId: string): Promise<string> {
  const token = getToken();
  const response = await fetch(`${API_BASE_URL}/api/v1/admin/products/${productId}/media/${mediaId}/file`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });

  if (!response.ok) {
    throw new Error(`Failed to load media with status ${response.status}`);
  }

  const blob = await response.blob();
  return URL.createObjectURL(blob);
}

export type TopProduct = {
  product_id: string;
  product_name: string;
  quantity_sold: number;
  order_count: number;
};

export type CatalogStats = {
  total_products: number;
  active_products: number;
  draft_products: number;
  archived_products: number;
  total_variants: number;
  total_categories: number;
  top_products: TopProduct[];
};

export function getCatalogStats(): Promise<CatalogStats> {
  return apiFetch<CatalogStats>("/api/v1/admin/catalog/stats");
}
