import { apiFetch } from "./client";

export type Profile = {
  id: string;
  email: string;
  full_name: string;
  phone: string;
  roles: string[];
};

export function getProfile(): Promise<Profile> {
  return apiFetch<Profile>("/api/v1/me");
}

export function updateProfile(input: Partial<{ full_name: string; phone: string }>): Promise<Profile> {
  return apiFetch<Profile>("/api/v1/me", { method: "PATCH", body: input });
}

export type Address = {
  id: string;
  label: string;
  recipient_name: string;
  phone: string;
  line1: string;
  line2: string;
  city: string;
  region: string;
  postal_code: string;
  country_code: string;
  is_default: boolean;
};

export function listAddresses(): Promise<Address[]> {
  return apiFetch<Address[]>("/api/v1/me/addresses");
}

export type AddressInput = {
  label: string;
  recipient_name: string;
  phone: string;
  line1: string;
  line2: string;
  city: string;
  region: string;
  postal_code: string;
  country_code: string;
  is_default: boolean;
};

export function createAddress(input: AddressInput): Promise<Address> {
  return apiFetch<Address>("/api/v1/me/addresses", { method: "POST", body: input });
}

export function updateAddress(id: string, input: Partial<AddressInput>): Promise<Address> {
  return apiFetch<Address>(`/api/v1/me/addresses/${id}`, { method: "PATCH", body: input });
}

export function deleteAddress(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/me/addresses/${id}`, { method: "DELETE" });
}
