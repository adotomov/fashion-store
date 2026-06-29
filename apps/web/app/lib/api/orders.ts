import type { Money } from "../money/money";
import { apiFetch } from "./client";

// Backend money shape (amount_minor) differs from the frontend's Money type
// (amount) — converted at this API boundary, same as lib/api/products.ts.
type MoneyDTO = { amount_minor: number; currency: string };

function fromMoneyDTO(dto: MoneyDTO): Money {
  return { amount: dto.amount_minor, currency: dto.currency };
}

export type OrderStatus = "pending" | "paid" | "shipped" | "delivered" | "cancelled";

export type OrderItem = {
  id: string;
  product_name: string;
  variant_label?: string;
  quantity: number;
  unit_price: Money;
};

export type Order = {
  id: string;
  order_number: string;
  status: OrderStatus;
  total: Money;
  placed_at: string;
  items: OrderItem[];
  carrier?: string;
  tracking_number?: string;
  shipment_status?: string;
};

type RawOrderItem = Omit<OrderItem, "unit_price"> & { unit_price: MoneyDTO };
type RawOrder = Omit<Order, "total" | "items"> & { total: MoneyDTO; items: RawOrderItem[] };

export async function listOrders(): Promise<Order[]> {
  const raw = await apiFetch<RawOrder[]>("/api/v1/me/orders");
  return raw.map((o) => ({
    ...o,
    total: fromMoneyDTO(o.total),
    items: o.items.map((item) => ({ ...item, unit_price: fromMoneyDTO(item.unit_price) })),
  }));
}
