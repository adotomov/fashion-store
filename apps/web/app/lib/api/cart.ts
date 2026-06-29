import type { Money } from "../money/money";
import { clearCartToken, getCartToken, setCartToken } from "../cart/session";
import { apiFetch } from "./client";

type MoneyDTO = { amount_minor: number; currency: string };

function fromMoneyDTO(dto: MoneyDTO): Money {
  return { amount: dto.amount_minor, currency: dto.currency };
}

export type CartItem = {
  id: string;
  variant_id: string;
  product_id: string;
  product_name: string;
  product_slug: string;
  variant_label?: string;
  image_url?: string;
  unit_price: Money;
  line_total: Money;
  quantity: number;
  available_quantity: number;
};

export type Cart = {
  id: string;
  guest_token?: string;
  items: CartItem[];
  subtotal: Money;
  item_count: number;
};

type RawCartItem = Omit<CartItem, "unit_price" | "line_total"> & {
  unit_price: MoneyDTO;
  line_total: MoneyDTO;
};

type RawCart = Omit<Cart, "items" | "subtotal"> & {
  items: RawCartItem[];
  subtotal: MoneyDTO;
};

// The backend assigns a fresh guest_token the first time an anonymous cart
// is created — persist it immediately so the next request identifies the
// same cart.
function fromRawCart(raw: RawCart): Cart {
  if (raw.guest_token) setCartToken(raw.guest_token);
  return {
    ...raw,
    items: raw.items.map((item) => ({
      ...item,
      unit_price: fromMoneyDTO(item.unit_price),
      line_total: fromMoneyDTO(item.line_total),
    })),
    subtotal: fromMoneyDTO(raw.subtotal),
  };
}

function cartHeaders(): Record<string, string> {
  const token = getCartToken();
  return token ? { "X-Cart-Token": token } : {};
}

export async function getCart(): Promise<Cart> {
  const raw = await apiFetch<RawCart>("/api/v1/cart", { headers: cartHeaders() });
  return fromRawCart(raw);
}

export async function addCartItem(variantId: string, quantity: number = 1): Promise<Cart> {
  const raw = await apiFetch<RawCart>("/api/v1/cart/items", {
    method: "POST",
    body: { variant_id: variantId, quantity },
    headers: cartHeaders(),
  });
  return fromRawCart(raw);
}

export async function updateCartItemQuantity(itemId: string, quantity: number): Promise<Cart> {
  const raw = await apiFetch<RawCart>(`/api/v1/cart/items/${itemId}`, {
    method: "PATCH",
    body: { quantity },
    headers: cartHeaders(),
  });
  return fromRawCart(raw);
}

export async function removeCartItem(itemId: string): Promise<Cart> {
  const raw = await apiFetch<RawCart>(`/api/v1/cart/items/${itemId}`, {
    method: "DELETE",
    headers: cartHeaders(),
  });
  return fromRawCart(raw);
}

// Called right after login: folds whatever was in the guest cart into the
// now-authenticated user's cart, then clears the now-unused guest token.
export async function mergeGuestCartIntoUser(): Promise<Cart | null> {
  const token = getCartToken();
  if (!token) return null;
  const raw = await apiFetch<RawCart>("/api/v1/cart/merge", {
    method: "POST",
    body: { guest_token: token },
  });
  clearCartToken();
  return fromRawCart(raw);
}
