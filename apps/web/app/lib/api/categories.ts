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
  // A placeholder is a grouping-only node (e.g. "Men", "Women") that products
  // can never be assigned to directly — only its leaf descendants hold products.
  is_placeholder: boolean;
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
  isPlaceholder?: boolean,
): Promise<Category> {
  return apiFetch<Category>("/api/v1/admin/categories", {
    method: "POST",
    body: {
      name,
      parent_id: parentId,
      product_type_id: productTypeId,
      internal_identifier: internalIdentifier,
      is_placeholder: isPlaceholder ?? false,
    },
  });
}

export function updateCategory(
  id: string,
  input: Partial<{
    name: string;
    parent_id: string | null;
    product_type_id: string;
    internal_identifier: string;
    is_placeholder: boolean;
  }>,
): Promise<Category> {
  return apiFetch<Category>(`/api/v1/admin/categories/${id}`, { method: "PATCH", body: input });
}

// Builds a category's path as "Type › Parent › Name" by walking parent_id up
// the flat category list. Used to disambiguate same-named categories (e.g.
// three "Women" placeholders under different types) in admin pickers without a
// stored label. Pass `types` to prepend the product type — the top of the
// hierarchy and usually what tells the duplicates apart. Cycle-guarded so a
// malformed parent chain can't loop forever.
export function categoryPath(category: Category, all: Category[], types?: { id: string; name: string }[]): string {
  const byId = new Map(all.map((c) => [c.id, c]));
  const names: string[] = [];
  const seen = new Set<string>();
  let current: Category | undefined = category;
  while (current && !seen.has(current.id)) {
    seen.add(current.id);
    names.unshift(current.name);
    current = current.parent_id ? byId.get(current.parent_id) : undefined;
  }
  const typeName = types?.find((t) => t.id === category.product_type_id)?.name;
  if (typeName) names.unshift(typeName);
  return names.join(" › ");
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
