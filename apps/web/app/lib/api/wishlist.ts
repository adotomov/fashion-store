import type { Money } from "../money/money";
import { apiFetch } from "./client";

type MoneyDTO = { amount_minor: number; currency: string };

function fromMoneyDTO(dto: MoneyDTO): Money {
  return { amount: dto.amount_minor, currency: dto.currency };
}

export type WishlistItem = {
  id: string;
  product_id: string;
  product_name: string;
  product_slug: string;
  image_url?: string;
  base_price: Money;
  compare_at_price?: Money;
  promotion_price?: Money;
  promotion_label?: string;
  in_stock: boolean;
  sizes: string[];
  created_at: string;
};

type RawWishlistItem = Omit<WishlistItem, "base_price" | "compare_at_price" | "promotion_price"> & {
  base_price: MoneyDTO;
  compare_at_price?: MoneyDTO;
  promotion_price?: MoneyDTO;
};

function fromRawItem(raw: RawWishlistItem): WishlistItem {
  return {
    ...raw,
    base_price: fromMoneyDTO(raw.base_price),
    compare_at_price: raw.compare_at_price ? fromMoneyDTO(raw.compare_at_price) : undefined,
    promotion_price: raw.promotion_price ? fromMoneyDTO(raw.promotion_price) : undefined,
  };
}

export async function listWishlist(): Promise<WishlistItem[]> {
  const raw = await apiFetch<RawWishlistItem[]>("/api/v1/wishlist");
  return raw.map(fromRawItem);
}

export async function addToWishlist(productId: string): Promise<WishlistItem> {
  const raw = await apiFetch<RawWishlistItem>(`/api/v1/wishlist/${productId}`, { method: "POST" });
  return fromRawItem(raw);
}

export function removeFromWishlist(productId: string): Promise<void> {
  return apiFetch<void>(`/api/v1/wishlist/${productId}`, { method: "DELETE" });
}
