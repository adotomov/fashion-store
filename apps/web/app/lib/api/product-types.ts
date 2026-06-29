import { apiFetch } from "./client";

export type ProductType = {
  id: string;
  name: string;
  slug: string;
  position: number;
  created_at: string;
  updated_at: string;
};

export function listProductTypes(): Promise<ProductType[]> {
  return apiFetch<ProductType[]>("/api/v1/admin/product-types");
}

export function createProductType(name: string): Promise<ProductType> {
  return apiFetch<ProductType>("/api/v1/admin/product-types", {
    method: "POST",
    body: { name },
  });
}

export function updateProductType(id: string, input: Partial<{ name: string; position: number }>): Promise<ProductType> {
  return apiFetch<ProductType>(`/api/v1/admin/product-types/${id}`, { method: "PATCH", body: input });
}

export function deleteProductType(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/product-types/${id}`, { method: "DELETE" });
}
