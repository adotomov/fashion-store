import { apiFetch } from "./client";

export type Language = {
  code: string;
  name: string;
  is_default: boolean;
  enabled: boolean;
};

export function listLanguages(): Promise<Language[]> {
  return apiFetch<Language[]>("/api/v1/admin/languages");
}

export function addLanguage(code: string, name: string): Promise<Language> {
  return apiFetch<Language>("/api/v1/admin/languages", { method: "POST", body: { code, name } });
}

export function setLanguageEnabled(code: string, enabled: boolean): Promise<Language> {
  return apiFetch<Language>(`/api/v1/admin/languages/${code}`, { method: "PATCH", body: { enabled } });
}

export function deleteLanguage(code: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/languages/${code}`, { method: "DELETE" });
}
