import { apiFetch } from "./client";

// VAT tax groups (Bulgarian fiscal groups А–Ж). Managed under Invoices &
// Taxes → Tax; referenced by products to drive per-line VAT on invoices.
export type TaxGroup = {
  id: string;
  identifier: string;
  vat_rate: number;
};

// The valid Cyrillic identifiers, mirroring domain.TaxGroupIdentifiers on the
// backend. Used to populate the create/edit dropdown.
export const TAX_GROUP_IDENTIFIERS = ["А", "Б", "В", "Г", "Д", "Е", "Ж"] as const;

export function listTaxGroups(): Promise<TaxGroup[]> {
  return apiFetch<TaxGroup[]>("/api/v1/admin/tax-groups");
}

export function createTaxGroup(identifier: string, vatRate: number): Promise<TaxGroup> {
  return apiFetch<TaxGroup>("/api/v1/admin/tax-groups", {
    method: "POST",
    body: { identifier, vat_rate: vatRate },
  });
}

export function updateTaxGroup(id: string, identifier: string, vatRate: number): Promise<TaxGroup> {
  return apiFetch<TaxGroup>(`/api/v1/admin/tax-groups/${id}`, {
    method: "PUT",
    body: { identifier, vat_rate: vatRate },
  });
}

export function deleteTaxGroup(id: string): Promise<void> {
  return apiFetch<void>(`/api/v1/admin/tax-groups/${id}`, { method: "DELETE" });
}
