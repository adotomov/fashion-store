import { apiFetch } from "./client";
import { getToken } from "../auth/session";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export type DocumentType = "terms" | "privacy" | "faq" | "shipping";

export type StoreDocument = {
  locale: string;
  filename: string;
  url: string;
};

export function listStoreDocuments(type: DocumentType): Promise<StoreDocument[]> {
  return apiFetch<StoreDocument[]>(`/api/v1/admin/store-settings/documents/${type}`);
}

// Multipart upload tagged with a locale — bypasses apiFetch's JSON-only
// body handling, same pattern as store-settings.ts's uploadFile.
export async function uploadStoreDocument(type: DocumentType, locale: string, file: File): Promise<StoreDocument> {
  const token = getToken();
  const form = new FormData();
  form.append("file", file);
  form.append("locale", locale);

  const response = await fetch(`${API_BASE_URL}/api/v1/admin/store-settings/documents/${type}`, {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : undefined,
    body: form,
  });

  if (!response.ok) {
    throw new Error(`Upload failed with status ${response.status}`);
  }

  return (await response.json()) as StoreDocument;
}

export function deleteStoreDocument(type: DocumentType, locale: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/store-settings/documents/${type}?locale=${encodeURIComponent(locale)}`, {
    method: "DELETE",
  });
}

export type LegalContent = {
  locale: string;
  content_md: string;
};

export function getLegalContent(type: DocumentType, locale: string): Promise<LegalContent> {
  return apiFetch<LegalContent>(
    `/api/v1/admin/store-settings/documents/${type}/content?locale=${encodeURIComponent(locale)}`,
  );
}

export function saveLegalContent(type: DocumentType, locale: string, content_md: string): Promise<LegalContent> {
  return apiFetch<LegalContent>(`/api/v1/admin/store-settings/documents/${type}/content`, {
    method: "PUT",
    body: { locale, content_md },
  });
}

export function getStorefrontLegalContent(type: DocumentType, locale: string): Promise<LegalContent> {
  return apiFetch<LegalContent>(
    `/api/v1/storefront/store-settings/documents/${type}/content?locale=${encodeURIComponent(locale)}`,
    { auth: false },
  );
}
