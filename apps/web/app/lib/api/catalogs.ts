import { apiFetch } from "./client";
import { getToken } from "../auth/session";

export type CatalogStatus = "draft" | "active" | "disabled";

export type Catalog = {
  id: string;
  name: string;
  slug: string;
  description: string;
  status: CatalogStatus;
  created_at: string;
  updated_at: string;
};

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export function listCatalogs(): Promise<Catalog[]> {
  return apiFetch<Catalog[]>("/api/v1/admin/catalogs");
}

export function createCatalog(name: string): Promise<Catalog> {
  return apiFetch<Catalog>("/api/v1/admin/catalogs", { method: "POST", body: { name } });
}

export function updateCatalog(
  id: string,
  input: Partial<{ name: string; description: string; status: CatalogStatus }>,
): Promise<Catalog> {
  return apiFetch<Catalog>(`/api/v1/admin/catalogs/${id}`, { method: "PATCH", body: input });
}

export function deleteCatalog(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/catalogs/${id}`, { method: "DELETE" });
}

// CSV/JSON export returns a file body, not JSON, so it bypasses apiFetch
// and triggers a browser download directly via a Blob URL.
export async function downloadCatalogExport(id: string, format: "csv" | "json"): Promise<void> {
  const token = getToken();
  const response = await fetch(`${API_BASE_URL}/api/v1/admin/catalogs/${id}/export?format=${format}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });

  if (!response.ok) {
    throw new Error(`Export failed with status ${response.status}`);
  }

  const blob = await response.blob();
  const disposition = response.headers.get("Content-Disposition") ?? "";
  const match = /filename="([^"]+)"/.exec(disposition);
  const filename = match?.[1] ?? `catalog.${format}`;

  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  link.remove();
  URL.revokeObjectURL(url);
}
