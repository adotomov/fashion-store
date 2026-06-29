import { apiFetch } from "./client";

export type PaymentMethod = {
  id: string;
  brand: string;
  last4: string;
  exp_month: number;
  exp_year: number;
  is_default: boolean;
};

export type PaymentMethodInput = {
  brand: string;
  last4: string;
  exp_month: number;
  exp_year: number;
  is_default: boolean;
};

export function listPaymentMethods(): Promise<PaymentMethod[]> {
  return apiFetch<PaymentMethod[]>("/api/v1/me/payment-methods");
}

export function createPaymentMethod(input: PaymentMethodInput): Promise<PaymentMethod> {
  return apiFetch<PaymentMethod>("/api/v1/me/payment-methods", { method: "POST", body: input });
}

export function updatePaymentMethod(id: string, input: Partial<PaymentMethodInput>): Promise<PaymentMethod> {
  return apiFetch<PaymentMethod>(`/api/v1/me/payment-methods/${id}`, { method: "PATCH", body: input });
}

export function deletePaymentMethod(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/me/payment-methods/${id}`, { method: "DELETE" });
}
