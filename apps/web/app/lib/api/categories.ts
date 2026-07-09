import { apiFetch } from "./client";
import { getToken } from "../auth/session";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export type Category = {
  id: string;
  name: string;
  slug: string;
  parent_id?: string;
  product_type_id: string;
  // Internal identifier (e.g. "DR-01") used as the fixed prefix for variant
  // SKUs of products in this category. Empty string means none assigned.
  internal_identifier: string;
  // Present once a thumbnail has been uploaded — a relative, admin-gated
  // proxy path (see uploadCategoryThumbnail/loadCategoryThumbnailBlobUrl),
  // not a plain external URL.
  image_url?: string;
  created_at: string;
  updated_at: string;
};

export function listCategories(): Promise<Category[]> {
  return apiFetch<Category[]>("/api/v1/admin/categories");
}

export function createCategory(
  name: string,
  productTypeId: string,
  parentId?: string,
  internalIdentifier?: string,
): Promise<Category> {
  return apiFetch<Category>("/api/v1/admin/categories", {
    method: "POST",
    body: { name, parent_id: parentId, product_type_id: productTypeId, internal_identifier: internalIdentifier },
  });
}

export function updateCategory(
  id: string,
  input: Partial<{ name: string; parent_id: string | null; product_type_id: string; internal_identifier: string }>,
): Promise<Category> {
  return apiFetch<Category>(`/api/v1/admin/categories/${id}`, { method: "PATCH", body: input });
}

export function deleteCategory(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/categories/${id}`, { method: "DELETE" });
}

// Thumbnail upload is multipart (real file bytes), so it bypasses apiFetch's
// JSON-only body handling and attaches the auth header manually — same
// pattern as uploadProductMedia in lib/api/products.ts.
export async function uploadCategoryThumbnail(categoryId: string, file: File): Promise<Category> {
  const token = getToken();
  const form = new FormData();
  form.append("file", file);

  const response = await fetch(`${API_BASE_URL}/api/v1/admin/categories/${categoryId}/thumbnail`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    body: form,
  });

  if (!response.ok) {
    throw new Error(`Upload failed with status ${response.status}`);
  }

  return (await response.json()) as Category;
}

export function deleteCategoryThumbnail(categoryId: string): Promise<Category> {
  return apiFetch<Category>(`/api/v1/admin/categories/${categoryId}/thumbnail`, { method: "DELETE" });
}

// The serve-thumbnail endpoint is admin-gated (Bearer token), and <img src>
// can't attach custom headers, so callers must fetch the bytes via JS and
// use the resulting blob URL as the <img> source. Caller is responsible
// for calling URL.revokeObjectURL when done (e.g. on unmount).
export async function loadCategoryThumbnailBlobUrl(categoryId: string): Promise<string> {
  const token = getToken();
  const response = await fetch(`${API_BASE_URL}/api/v1/admin/categories/${categoryId}/thumbnail/file`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });

  if (!response.ok) {
    throw new Error(`Failed to load thumbnail with status ${response.status}`);
  }

  const blob = await response.blob();
  return URL.createObjectURL(blob);
}
