import type { Money } from "../money/money";
import { apiFetch } from "./client";

type MoneyDTO = { amount_minor: number; currency: string };

function fromMoneyDTO(dto: MoneyDTO): Money {
  return { amount: dto.amount_minor, currency: dto.currency };
}

export type AdminOrderStatus =
  | "pending"
  | "pending_payment"
  | "paid"
  | "payment_failed"
  | "shipped"
  | "delivered"
  | "cancelled"
  | "refunded"
  | "partially_refunded";

export type AdminOrderAddress = {
  recipient_name: string;
  phone: string;
  line1: string;
  line2: string;
  city: string;
  region: string;
  postal_code: string;
  country_code: string;
};

export type AdminOrderPayment = {
  provider: string;
  provider_reference?: string;
  status: string;
  amount: Money;
  captured: Money;
  refunded: Money;
};

export type AdminOrderItem = {
  id: string;
  product_name: string;
  variant_label?: string;
  quantity: number;
  unit_price: Money;
};

export type AdminOrder = {
  id: string;
  order_number: string;
  status: AdminOrderStatus;
  total: Money;
  placed_at: string;
  contact_name?: string;
  contact_email?: string;
  contact_phone?: string;
  shipping_address: AdminOrderAddress;
  billing_address: AdminOrderAddress;
  delivery_method: string;
  delivery_fee: Money;
  payment_method: string;
  payment?: AdminOrderPayment;
  carrier?: string;
  tracking_number?: string;
  shipment_status?: string;
  viewed_by_admin_at?: string;
  items: AdminOrderItem[];
};

type RawOrderItem = Omit<AdminOrderItem, "unit_price"> & { unit_price: MoneyDTO };
type RawOrderPayment = Omit<AdminOrderPayment, "amount" | "captured" | "refunded"> & {
  amount: MoneyDTO;
  captured: MoneyDTO;
  refunded: MoneyDTO;
};
type RawAdminOrder = Omit<AdminOrder, "total" | "delivery_fee" | "payment" | "items"> & {
  total: MoneyDTO;
  delivery_fee: MoneyDTO;
  payment?: RawOrderPayment;
  items: RawOrderItem[];
};

function fromRawOrder(raw: RawAdminOrder): AdminOrder {
  return {
    ...raw,
    total: fromMoneyDTO(raw.total),
    delivery_fee: fromMoneyDTO(raw.delivery_fee),
    payment: raw.payment
      ? {
          ...raw.payment,
          amount: fromMoneyDTO(raw.payment.amount),
          captured: fromMoneyDTO(raw.payment.captured),
          refunded: fromMoneyDTO(raw.payment.refunded),
        }
      : undefined,
    items: raw.items.map((item) => ({ ...item, unit_price: fromMoneyDTO(item.unit_price) })),
  };
}

export async function listAdminOrders(filter?: { status?: string; unviewedOnly?: boolean }): Promise<AdminOrder[]> {
  const params = new URLSearchParams();
  if (filter?.status) params.set("status", filter.status);
  if (filter?.unviewedOnly) params.set("unviewed_only", "true");
  const query = params.toString();
  const raw = await apiFetch<RawAdminOrder[]>(`/api/v1/admin/orders${query ? `?${query}` : ""}`);
  return raw.map(fromRawOrder);
}

export async function getAdminOrder(id: string): Promise<AdminOrder> {
  const raw = await apiFetch<RawAdminOrder>(`/api/v1/admin/orders/${id}`);
  return fromRawOrder(raw);
}

export type UpdateFulfillmentInput = Partial<{
  status: AdminOrderStatus;
  carrier: string;
  tracking_number: string;
  shipment_status: string;
}>;

export async function updateOrderFulfillment(id: string, input: UpdateFulfillmentInput): Promise<AdminOrder> {
  const raw = await apiFetch<RawAdminOrder>(`/api/v1/admin/orders/${id}`, { method: "PATCH", body: input });
  return fromRawOrder(raw);
}

export async function refundOrder(id: string, amountMinor: number, reason?: string): Promise<void> {
  await apiFetch(`/api/v1/admin/orders/${id}/refund`, {
    method: "POST",
    body: { amount_minor: amountMinor, reason },
  });
}

export type PaymentTransaction = {
  id: string;
  type: "initiated" | "captured" | "failed" | "refunded";
  status?: string;
  provider: string;
  provider_reference?: string;
  amount: Money;
  created_at: string;
};

type RawPaymentTransaction = Omit<PaymentTransaction, "amount"> & { amount: MoneyDTO };

export async function listPaymentTransactions(id: string): Promise<PaymentTransaction[]> {
  const raw = await apiFetch<RawPaymentTransaction[]>(`/api/v1/admin/orders/${id}/transactions`);
  return raw.map((txn) => ({ ...txn, amount: fromMoneyDTO(txn.amount) }));
}

export async function getUnviewedOrderCount(): Promise<number> {
  const { count } = await apiFetch<{ count: number }>("/api/v1/admin/orders/unviewed-count");
  return count;
}

export type OrderStatsRange = "7d" | "30d" | "90d";

export type CountBreakdown = { label: string; count: number };

export type DailyOrderCount = { date: string; count: number; revenue: Money };

export type OrderStats = {
  order_count: number;
  revenue: Money;
  avg_order_value: Money;
  status_breakdown: CountBreakdown[];
  by_city: CountBreakdown[];
  by_country: CountBreakdown[];
  by_delivery_method: CountBreakdown[];
  daily_counts: DailyOrderCount[];
};

type RawDailyOrderCount = Omit<DailyOrderCount, "revenue"> & { revenue: MoneyDTO };
type RawOrderStats = Omit<OrderStats, "revenue" | "avg_order_value" | "daily_counts"> & {
  revenue: MoneyDTO;
  avg_order_value: MoneyDTO;
  daily_counts: RawDailyOrderCount[];
};

export async function getOrderStats(range: OrderStatsRange = "7d"): Promise<OrderStats> {
  const raw = await apiFetch<RawOrderStats>(`/api/v1/admin/orders/stats?range=${range}`);
  return {
    ...raw,
    revenue: fromMoneyDTO(raw.revenue),
    avg_order_value: fromMoneyDTO(raw.avg_order_value),
    daily_counts: raw.daily_counts.map((d) => ({ ...d, revenue: fromMoneyDTO(d.revenue) })),
  };
}
