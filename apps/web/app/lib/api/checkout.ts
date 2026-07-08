import type { Money } from "../money/money";
import { getCartToken } from "../cart/session";
import { apiFetch } from "./client";

type MoneyDTO = { amount_minor: number; currency: string };

function fromMoneyDTO(dto: MoneyDTO): Money {
  return { amount: dto.amount_minor, currency: dto.currency };
}

export type DeliveryMethodCode = "speedy" | "easybox";
export type PaymentMethodCode = "cash_on_delivery" | "card_on_easybox" | "card_online";

export type DeliveryMethod = {
  code: DeliveryMethodCode;
  name: string;
  fee: Money;
  /** Payment methods compatible with this delivery method (server-decided). */
  payment_methods: PaymentMethodCode[];
};

type RawDeliveryMethod = Omit<DeliveryMethod, "fee"> & { fee: MoneyDTO };

export function listDeliveryMethods(): Promise<DeliveryMethod[]> {
  return apiFetch<RawDeliveryMethod[]>("/api/v1/checkout/delivery-methods", { auth: false }).then((raw) =>
    raw.map((m) => ({ ...m, fee: fromMoneyDTO(m.fee) })),
  );
}

export type Contact = {
  full_name: string;
  email: string;
  phone: string;
};

export type CheckoutAddress = {
  recipient_name: string;
  phone: string;
  line1: string;
  line2: string;
  city: string;
  region: string;
  postal_code: string;
  country_code: string;
};

export type Card = {
  number: string;
  exp_month: number;
  exp_year: number;
  cvv: string;
};

export type PlaceOrderInput = {
  contact: Contact;
  shipping_address: CheckoutAddress;
  billing_address: CheckoutAddress;
  delivery_method: DeliveryMethodCode;
  delivery_office_id?: string;
  payment_method: PaymentMethodCode;
  card?: Card;
  discount_code?: string;
};

export type OrderItem = {
  product_name: string;
  variant_label?: string;
  quantity: number;
  unit_price: Money;
};

export type PlacedOrder = {
  id: string;
  order_number: string;
  status: string;
  total: Money;
  delivery_method: DeliveryMethodCode;
  delivery_fee: Money;
  payment_method: PaymentMethodCode;
  placed_at: string;
  discount_code?: string;
  discount_amount?: Money;
  items: OrderItem[];
};

type RawOrderItem = Omit<OrderItem, "unit_price"> & { unit_price: MoneyDTO };
type RawPlacedOrder = Omit<PlacedOrder, "total" | "delivery_fee" | "discount_amount" | "items"> & {
  total: MoneyDTO;
  delivery_fee: MoneyDTO;
  discount_amount?: MoneyDTO;
  items: RawOrderItem[];
};

export type DiscountCodeValidation = {
  code: string;
  value_percent: number;
  valid: boolean;
};

export async function validateDiscountCode(code: string): Promise<DiscountCodeValidation> {
  return apiFetch<DiscountCodeValidation>(
    `/api/v1/checkout/discount?code=${encodeURIComponent(code)}`,
    { auth: false },
  );
}

export async function placeOrder(input: PlaceOrderInput): Promise<PlacedOrder> {
  const token = getCartToken();
  const raw = await apiFetch<RawPlacedOrder>("/api/v1/checkout", {
    method: "POST",
    body: input,
    headers: token ? { "X-Cart-Token": token } : {},
  });
  return {
    ...raw,
    total: fromMoneyDTO(raw.total),
    delivery_fee: fromMoneyDTO(raw.delivery_fee),
    discount_amount: raw.discount_amount ? fromMoneyDTO(raw.discount_amount) : undefined,
    items: raw.items.map((item) => ({ ...item, unit_price: fromMoneyDTO(item.unit_price) })),
  };
}
