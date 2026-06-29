import { apiFetch } from "./client";
import { getToken } from "../auth/session";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export type StoreSettings = {
  store_name: string;
  legal_entity_name?: string;
  locale: string;
  currency: string;
  contact_email?: string;
  contact_phone?: string;
  company_description?: string;
  // Relative, admin-gated proxy path — never a plain external URL.
  logo_url?: string;
  updated_at: string;
};

export function getStoreSettings(): Promise<StoreSettings> {
  return apiFetch<StoreSettings>("/api/v1/admin/store-settings");
}

export function updateStoreSettings(
  input: Partial<{
    store_name: string;
    legal_entity_name: string;
    locale: string;
    currency: string;
    contact_email: string;
    contact_phone: string;
    company_description: string;
  }>,
): Promise<StoreSettings> {
  return apiFetch<StoreSettings>("/api/v1/admin/store-settings", { method: "PATCH", body: input });
}

// Uploads are multipart (real file bytes), so they bypass apiFetch's
// JSON-only body handling and attach the auth header manually — same
// pattern as uploadCategoryThumbnail in lib/api/categories.ts.
async function uploadFile(path: string, file: File): Promise<StoreSettings> {
  const token = getToken();
  const form = new FormData();
  form.append("file", file);

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    body: form,
  });

  if (!response.ok) {
    throw new Error(`Upload failed with status ${response.status}`);
  }

  return (await response.json()) as StoreSettings;
}

// The serve-file endpoints are admin-gated (Bearer token), and <img src>
// can't attach custom headers, so callers must fetch the bytes via JS and
// use the resulting blob URL as the source. Caller is responsible for
// calling URL.revokeObjectURL when done (e.g. on unmount).
export async function loadBlobUrl(path: string): Promise<string> {
  const token = getToken();
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
  });

  if (!response.ok) {
    throw new Error(`Failed to load file with status ${response.status}`);
  }

  const blob = await response.blob();
  return URL.createObjectURL(blob);
}

export function uploadStoreLogo(file: File): Promise<StoreSettings> {
  return uploadFile("/api/v1/admin/store-settings/logo", file);
}

export function deleteStoreLogo(): Promise<StoreSettings> {
  return apiFetch<StoreSettings>("/api/v1/admin/store-settings/logo", { method: "DELETE" });
}

export function loadStoreLogoBlobUrl(): Promise<string> {
  return loadBlobUrl("/api/v1/admin/store-settings/logo/file");
}
