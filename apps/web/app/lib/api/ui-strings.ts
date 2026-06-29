import { apiFetch } from "./client";

export type UIString = {
  key: string;
  locale: string;
  value: string;
};

export function listAllUiStrings(): Promise<UIString[]> {
  return apiFetch<UIString[]>("/api/v1/admin/ui-strings");
}

export function upsertUiString(key: string, locale: string, value: string): Promise<void> {
  return apiFetch<void>("/api/v1/admin/ui-strings", { method: "PUT", body: { key, locale, value } });
}
