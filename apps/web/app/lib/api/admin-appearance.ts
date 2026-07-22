import { apiFetch } from "./client";

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

export function getHeroSettings(): Promise<HeroSettings> {
  return apiFetch<HeroSettings>("/api/v1/admin/hero");
}

export function saveHeroSettings(data: SaveHeroSettingsInput): Promise<HeroSettings> {
  return apiFetch<HeroSettings>("/api/v1/admin/hero", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
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

export function getPublicHeroSettings(): Promise<HeroSettings> {
  return apiFetch<HeroSettings>("/api/v1/storefront/hero", { auth: false });
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

export function getEditorialBanner(): Promise<EditorialBannerSettings> {
  return apiFetch<EditorialBannerSettings>("/api/v1/admin/editorial-banner");
}

export function saveEditorialBanner(
  data: SaveEditorialBannerInput,
): Promise<EditorialBannerSettings> {
  return apiFetch<EditorialBannerSettings>("/api/v1/admin/editorial-banner", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
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

export function getPublicEditorialBanner(): Promise<EditorialBannerSettings> {
  return apiFetch<EditorialBannerSettings>("/api/v1/storefront/editorial-banner", {
    auth: false,
  });
}
