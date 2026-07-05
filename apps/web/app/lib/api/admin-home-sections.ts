import { apiFetch } from "./client";

export type HomeSectionConfig = {
  id: string;
  enabled: boolean;
  eyebrow: string;
  heading: string;
  updated_at: string;
};

// Admin endpoints (auth required)

export function listAdminHomeSections(): Promise<HomeSectionConfig[]> {
  return apiFetch<HomeSectionConfig[]>("/api/v1/admin/home-sections");
}

export function saveHomeSection(
  id: string,
  data: { enabled: boolean; eyebrow: string; heading: string },
): Promise<HomeSectionConfig> {
  return apiFetch<HomeSectionConfig>(`/api/v1/admin/home-sections/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: data,
  });
}

export function getAdminSectionProductIDs(sectionId: string): Promise<string[]> {
  return apiFetch<string[]>(`/api/v1/admin/home-sections/${encodeURIComponent(sectionId)}/products`);
}

export function setSectionProducts(sectionId: string, productIds: string[]): Promise<string[]> {
  return apiFetch<string[]>(`/api/v1/admin/home-sections/${encodeURIComponent(sectionId)}/products`, {
    method: "PUT",
    body: productIds,
  });
}

// Public storefront endpoints (no auth)

export function getPublicHomeSections(): Promise<HomeSectionConfig[]> {
  return apiFetch<HomeSectionConfig[]>("/api/v1/storefront/home-sections", { auth: false });
}

export function getPublicSectionProductIDs(sectionId: string): Promise<string[]> {
  return apiFetch<string[]>(
    `/api/v1/storefront/home-sections/${encodeURIComponent(sectionId)}/product-ids`,
    { auth: false },
  );
}
