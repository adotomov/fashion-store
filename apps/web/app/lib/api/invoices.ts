import { getToken } from "../auth/session";
import { apiFetch } from "./client";

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

export type InvoiceListItem = {
  id: string;
  invoice_number: string;
  document_type: "фактура" | "сторно";
  order_id: string;
  order_number: string;
  payment_method: string;
  recipient_name: string;
  total_incl_vat: number;
  currency: string;
  created_at: string;
  storno_of_invoice_id?: string;
};

export type LineItem = {
  product_name: string;
  variant_label: string;
  quantity: number;
  unit_price_incl_vat: number;
  unit_price_excl_vat: number;
  vat_per_unit: number;
  line_total_incl_vat: number;
  line_total_excl_vat: number;
  line_vat_amount: number;
};

export type Invoice = InvoiceListItem & {
  recipient_address: string;
  recipient_email: string;
  company_name: string;
  company_legal_type: string;
  company_eik: string;
  nra_store_number: string;
  card_provider?: string;
  card_provider_reference?: string;
  courier_name?: string;
  courier_identifier?: string;
  subtotal_excl_vat: number;
  vat_amount: number;
  vat_rate: number;
  delivery_fee: number;
  discount_amount?: number;
  line_items: LineItem[];
};

export type InvoiceSettings = {
  company_name: string;
  company_legal_type: string;
  company_eik: string;
  company_address_street: string;
  company_address_city: string;
  company_address_postal_code: string;
  company_address_country: string;
  company_email: string;
  company_phone: string;
  nra_store_number: string;
  vat_number: string;
  vat_rate: number;
};

export type Courier = {
  id: string;
  name: string;
  identifier: string;
  is_active: boolean;
  sort_order: number;
};

export type ListInvoicesParams = {
  from?: string;
  to?: string;
  document_type?: string;
  payment_method?: string;
  q?: string;
  limit?: number;
  offset?: number;
};

export type ListInvoicesResponse = {
  invoices: InvoiceListItem[];
  total: number;
};

export async function listInvoices(params: ListInvoicesParams = {}): Promise<ListInvoicesResponse> {
  const qs = new URLSearchParams();
  if (params.from) qs.set("from", params.from);
  if (params.to) qs.set("to", params.to);
  if (params.document_type) qs.set("document_type", params.document_type);
  if (params.payment_method) qs.set("payment_method", params.payment_method);
  if (params.q) qs.set("q", params.q);
  if (params.limit != null) qs.set("limit", String(params.limit));
  if (params.offset != null) qs.set("offset", String(params.offset));
  const query = qs.toString();
  return apiFetch(`/api/v1/admin/invoices${query ? `?${query}` : ""}`);
}

export async function generateInvoice(orderID: string): Promise<Invoice> {
  return apiFetch(`/api/v1/admin/invoices/generate/${orderID}`, { method: "POST" });
}

export async function generateStorno(invoiceID: string): Promise<Invoice> {
  return apiFetch(`/api/v1/admin/invoices/${invoiceID}/storno`, { method: "POST" });
}

export async function exportInvoicesCSV(from: string, to: string): Promise<void> {
  const token = getToken();
  const qs = new URLSearchParams({ from, to });
  const res = await fetch(`${API_BASE_URL}/api/v1/admin/invoices/export?${qs}`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  if (!res.ok) throw new Error("Export failed");
  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `invoices-${from}-to-${to}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

export async function getInvoiceSettings(): Promise<InvoiceSettings> {
  return apiFetch("/api/v1/admin/invoice-settings");
}

export async function saveInvoiceSettings(data: Partial<InvoiceSettings>): Promise<InvoiceSettings> {
  return apiFetch("/api/v1/admin/invoice-settings", { method: "PUT", body: data });
}

export async function listCouriers(): Promise<Courier[]> {
  return apiFetch("/api/v1/admin/invoice-couriers");
}

export type CreateCourierInput = { name: string; identifier: string; is_active: boolean; sort_order: number };

export async function createCourier(data: CreateCourierInput): Promise<Courier> {
  return apiFetch("/api/v1/admin/invoice-couriers", { method: "POST", body: data });
}

export async function updateCourier(id: string, data: Partial<CreateCourierInput>): Promise<Courier> {
  return apiFetch(`/api/v1/admin/invoice-couriers/${id}`, { method: "PUT", body: data });
}

export async function deleteCourier(id: string): Promise<void> {
  return apiFetch(`/api/v1/admin/invoice-couriers/${id}`, { method: "DELETE" });
}
