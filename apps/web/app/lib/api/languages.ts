import { apiFetch } from "./client";

export type Language = {
  code: string;
  name: string;
  is_default: boolean;
  enabled: boolean;
  country_code?: string;
};

export function listLanguages(): Promise<Language[]> {
  return apiFetch<Language[]>("/api/v1/admin/languages");
}

// Makes an enabled language the store's default display language (what a visitor
// sees before any location/account/browser signal applies).
export function setDefaultLanguage(code: string): Promise<Language> {
  return apiFetch<Language>(`/api/v1/admin/languages/${code}/default`, { method: "POST" });
}

// Sets (or clears, when empty) the ISO-3166 country a language is shown to via
// geo detection, e.g. bg → BG.
export function setLanguageCountry(code: string, country_code: string): Promise<Language> {
  return apiFetch<Language>(`/api/v1/admin/languages/${code}/country`, {
    method: "PUT",
    body: { country_code },
  });
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
