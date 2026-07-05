import { apiFetch } from "./client";

export type PromotionType = "percentage" | "fixed" | "bxgy";
export type TargetType = "all" | "category" | "product_type" | "product";

export type Promotion = {
  id: string;
  name: string;
  description: string;
  type: PromotionType;
  value_percent?: number;
  value_fixed_minor?: number;
  value_fixed_currency?: string;
  buy_qty?: number;
  get_qty?: number;
  get_discount_pct?: number;
  min_quantity: number;
  target_type: TargetType;
  category_ids: string[];
  type_ids: string[];
  product_ids: string[];
  starts_at?: string;
  ends_at?: string;
  is_active: boolean;
  priority: number;
  created_at: string;
  updated_at: string;
};

export type CreatePromotionInput = {
  name: string;
  description?: string;
  type: PromotionType;
  value_percent?: number;
  value_fixed_minor?: number;
  value_fixed_currency?: string;
  buy_qty?: number;
  get_qty?: number;
  get_discount_pct?: number;
  min_quantity?: number;
  target_type: TargetType;
  category_ids?: string[];
  type_ids?: string[];
  product_ids?: string[];
  starts_at?: string;
  ends_at?: string;
  is_active?: boolean;
  priority?: number;
};

export type UpdatePromotionInput = Partial<CreatePromotionInput>;

export function listPromotions(): Promise<Promotion[]> {
  return apiFetch<Promotion[]>("/api/v1/admin/promotions");
}

export function getPromotion(id: string): Promise<Promotion> {
  return apiFetch<Promotion>(`/api/v1/admin/promotions/${id}`);
}

export function createPromotion(input: CreatePromotionInput): Promise<Promotion> {
  return apiFetch<Promotion>("/api/v1/admin/promotions", { method: "POST", body: input });
}

export function updatePromotion(id: string, input: UpdatePromotionInput): Promise<Promotion> {
  return apiFetch<Promotion>(`/api/v1/admin/promotions/${id}`, { method: "PATCH", body: input });
}

export function deletePromotion(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/promotions/${id}`, { method: "DELETE" });
}

export type DiscountCode = {
  id: string;
  code: string;
  value_percent: number;
  starts_at?: string;
  expires_at?: string;
  max_uses?: number;
  use_count: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type CreateDiscountCodeInput = {
  code: string;
  value_percent: number;
  starts_at?: string;
  expires_at?: string;
  max_uses?: number;
  is_active?: boolean;
};

export type UpdateDiscountCodeInput = Partial<CreateDiscountCodeInput>;

export function listDiscountCodes(): Promise<DiscountCode[]> {
  return apiFetch<DiscountCode[]>("/api/v1/admin/discount-codes");
}

export function createDiscountCode(input: CreateDiscountCodeInput): Promise<DiscountCode> {
  return apiFetch<DiscountCode>("/api/v1/admin/discount-codes", { method: "POST", body: input });
}

export function updateDiscountCode(id: string, input: UpdateDiscountCodeInput): Promise<DiscountCode> {
  return apiFetch<DiscountCode>(`/api/v1/admin/discount-codes/${id}`, { method: "PATCH", body: input });
}

export function deleteDiscountCode(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/discount-codes/${id}`, { method: "DELETE" });
}
