import { apiFetch } from "./client";

// entityType must match the constants the backend's catalog storefront
// handler overlays translations with (see entityType* consts in
// apps/api/internal/modules/catalog/transport/http/storefront_handler.go).
export type TranslatableEntityType = "product" | "category" | "catalog" | "product_type" | "attribute" | "attribute_value";

export function getTranslations(
  entityType: TranslatableEntityType,
  entityId: string,
  locale: string,
): Promise<Record<string, string>> {
  return apiFetch<Record<string, string>>(`/api/v1/admin/translations/${entityType}/${entityId}/${locale}`);
}

export function setTranslations(
  entityType: TranslatableEntityType,
  entityId: string,
  locale: string,
  fields: Record<string, string>,
): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/translations/${entityType}/${entityId}/${locale}`, {
    method: "PUT",
    body: fields,
  });
}
