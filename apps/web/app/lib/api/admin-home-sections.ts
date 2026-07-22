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

// Curated category groups (the "Best in its category" section): up to 5
// categories, each with an ordered, hand-picked product list.
export type SectionCategoryGroup = {
  category_id: string;
  product_ids: string[];
};

export function getSectionCategoryGroups(sectionId: string): Promise<SectionCategoryGroup[]> {
  return apiFetch<SectionCategoryGroup[]>(
    `/api/v1/admin/home-sections/${encodeURIComponent(sectionId)}/category-groups`,
  );
}

export function setSectionCategoryGroups(
  sectionId: string,
  groups: SectionCategoryGroup[],
): Promise<SectionCategoryGroup[]> {
  return apiFetch<SectionCategoryGroup[]>(
    `/api/v1/admin/home-sections/${encodeURIComponent(sectionId)}/category-groups`,
    { method: "PUT", body: groups },
  );
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
