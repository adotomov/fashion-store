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

// A Speedy-resolved saved address: city/complex/street carry Speedy location
// codes (…_id) plus display names; house details are free text. Bulgaria only.
export type Address = {
  id: string;
  label: string;
  recipient_name: string;
  phone: string;
  country_code: string;
  country_id: number;
  site_id: number;
  city: string;
  post_code: string;
  complex_id: number;
  complex_name: string;
  street_id: number;
  street_name: string;
  street_no: string;
  block_no: string;
  entrance_no: string;
  floor_no: string;
  apartment_no: string;
  is_default: boolean;
};

export function listAddresses(): Promise<Address[]> {
  return apiFetch<Address[]>("/api/v1/me/addresses");
}

export type AddressInput = {
  label: string;
  recipient_name: string;
  phone: string;
  country_code: string;
  country_id: number;
  site_id: number;
  city: string;
  post_code: string;
  complex_id: number;
  complex_name: string;
  street_id: number;
  street_name: string;
  street_no: string;
  block_no: string;
  entrance_no: string;
  floor_no: string;
  apartment_no: string;
  is_default: boolean;
};

export function createAddress(input: AddressInput): Promise<Address> {
  return apiFetch<Address>("/api/v1/me/addresses", { method: "POST", body: input });
}

// Update replaces the whole address (a resolved address is atomic), so callers
// send the full input.
export function updateAddress(id: string, input: AddressInput): Promise<Address> {
  return apiFetch<Address>(`/api/v1/me/addresses/${id}`, { method: "PATCH", body: input });
}

export function deleteAddress(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/me/addresses/${id}`, { method: "DELETE" });
}
