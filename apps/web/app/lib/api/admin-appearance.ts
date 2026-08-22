import { apiFetch } from "./client";

// Editorial home-page content (hero, editorial banner, home sections) is
// translatable: English lives in the base row, other locales are layered on
// via ?locale=. The default locale needs no query param.
function localeQuery(locale?: string): string {
  return locale && locale !== "en" ? `?locale=${encodeURIComponent(locale)}` : "";
}

export type HeroSettings = {
  eyebrow: string;
  heading: string;
  subtext: string;
  cta_primary_label: string;
  cta_primary_url: string;
  cta_secondary_label?: string;
  cta_secondary_url?: string;
  background_image_url?: string;
  updated_at: string;
};

export type SaveHeroSettingsInput = Omit<HeroSettings, "updated_at" | "background_image_url">;

export function getHeroSettings(locale?: string): Promise<HeroSettings> {
  return apiFetch<HeroSettings>(`/api/v1/admin/hero${localeQuery(locale)}`);
}

export function saveHeroSettings(data: SaveHeroSettingsInput, locale?: string): Promise<HeroSettings> {
  return apiFetch<HeroSettings>(`/api/v1/admin/hero${localeQuery(locale)}`, {
    method: "PUT",
    body: data,
  });
}

export function uploadHeroBackground(file: File): Promise<HeroSettings> {
  const form = new FormData();
  form.append("file", file);
  return apiFetch<HeroSettings>("/api/v1/admin/hero/background", {
    method: "POST",
    body: form,
  });
}

export function deleteHeroBackground(): Promise<HeroSettings> {
  return apiFetch<HeroSettings>("/api/v1/admin/hero/background", {
    method: "DELETE",
  });
}

export function getPublicHeroSettings(locale?: string): Promise<HeroSettings> {
  return apiFetch<HeroSettings>(`/api/v1/storefront/hero${localeQuery(locale)}`, { auth: false });
}

// Editorial ("Shop the Look") banner — a singleton, admin-configurable
// mid-page banner mirroring the hero's image + copy + CTA shape.

export type EditorialBannerSettings = {
  enabled: boolean;
  eyebrow: string;
  heading: string;
  subtext: string;
  cta_label: string;
  cta_url: string;
  image_url?: string;
  updated_at: string;
};

export type SaveEditorialBannerInput = Omit<
  EditorialBannerSettings,
  "updated_at" | "image_url"
>;

export function getEditorialBanner(locale?: string): Promise<EditorialBannerSettings> {
  return apiFetch<EditorialBannerSettings>(`/api/v1/admin/editorial-banner${localeQuery(locale)}`);
}

export function saveEditorialBanner(
  data: SaveEditorialBannerInput,
  locale?: string,
): Promise<EditorialBannerSettings> {
  return apiFetch<EditorialBannerSettings>(`/api/v1/admin/editorial-banner${localeQuery(locale)}`, {
    method: "PUT",
    body: data,
  });
}

export function uploadEditorialBannerImage(file: File): Promise<EditorialBannerSettings> {
  const form = new FormData();
  form.append("file", file);
  return apiFetch<EditorialBannerSettings>("/api/v1/admin/editorial-banner/image", {
    method: "POST",
    body: form,
  });
}

export function deleteEditorialBannerImage(): Promise<EditorialBannerSettings> {
  return apiFetch<EditorialBannerSettings>("/api/v1/admin/editorial-banner/image", {
    method: "DELETE",
  });
}

export function getPublicEditorialBanner(locale?: string): Promise<EditorialBannerSettings> {
  return apiFetch<EditorialBannerSettings>(`/api/v1/storefront/editorial-banner${localeQuery(locale)}`, {
    auth: false,
  });
}
