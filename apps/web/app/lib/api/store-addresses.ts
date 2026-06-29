import { apiFetch } from "./client";

export type StoreAddress = {
  id: string;
  label: string;
  line1: string;
  line2?: string;
  city?: string;
  region?: string;
  postal_code?: string;
  country?: string;
  is_default: boolean;
};

export type UpsertStoreAddressInput = {
  label: string;
  line1: string;
  line2?: string;
  city?: string;
  region?: string;
  postal_code?: string;
  country?: string;
  is_default: boolean;
};

export function listStoreAddresses(): Promise<StoreAddress[]> {
  return apiFetch<StoreAddress[]>("/api/v1/admin/store-settings/addresses");
}

export function createStoreAddress(input: UpsertStoreAddressInput): Promise<StoreAddress> {
  return apiFetch<StoreAddress>("/api/v1/admin/store-settings/addresses", { method: "POST", body: input });
}

export function updateStoreAddress(id: string, input: UpsertStoreAddressInput): Promise<StoreAddress> {
  return apiFetch<StoreAddress>(`/api/v1/admin/store-settings/addresses/${id}`, { method: "PATCH", body: input });
}

export function deleteStoreAddress(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/store-settings/addresses/${id}`, { method: "DELETE" });
}
