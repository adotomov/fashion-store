import type { Cart, CartItem } from "../api/cart";
import type { StorefrontProduct } from "../api/storefront";
import type { Money } from "../money/money";

import { track } from "./gtag";

// Typed GA4 ecommerce helpers. Each function maps our domain objects
// (StorefrontProduct, Cart, CartItem) onto GA4's recommended ecommerce schema
// and fires the corresponding event, so call sites stay one-liners.
//
// Money is stored as integer minor units in EUR (see lib/money/money.ts). GA
// wants a major-unit number and a currency code, so we divide by 100 and pass
// the underlying currency. BGN is a display-only conversion and is never
// reported to GA.

// GA4 item as documented at
// https://developers.google.com/analytics/devguides/collection/ga4/reference/events
export type GAItem = {
  // GA4 requires at least one of item_id / item_name. Product-derived items
  // carry both; items rebuilt from a placed order (which stores names, not
  // product UUIDs) carry item_name only.
  item_id?: string;
  item_name: string;
  price?: number;
  quantity?: number;
  item_variant?: string;
  item_list_id?: string;
  item_list_name?: string;
  index?: number;
};

const toMajor = (money: Money): number => money.amount / 100;

// The price a shopper actually pays: the promotion price when one is active,
// otherwise the base price. compare_at_price is the struck-through "was" price
// and is intentionally ignored here.
const effectivePrice = (product: StorefrontProduct): Money => product.promotion_price ?? product.base_price;

type ProductItemOpts = {
  index?: number;
  quantity?: number;
  listId?: string;
  listName?: string;
  variantLabel?: string;
};

// productToItem maps a storefront product (optionally with the selected variant
// and list context) onto a GA4 item.
export function productToItem(product: StorefrontProduct, opts: ProductItemOpts = {}): GAItem {
  return {
    item_id: product.id,
    item_name: product.name,
    price: toMajor(effectivePrice(product)),
    quantity: opts.quantity,
    item_variant: opts.variantLabel,
    item_list_id: opts.listId,
    item_list_name: opts.listName,
    index: opts.index,
  };
}

// cartItemToItem maps a line in the cart onto a GA4 item (unit price × its own
// quantity).
export function cartItemToItem(item: CartItem): GAItem {
  return {
    item_id: item.product_id,
    item_name: item.product_name,
    item_variant: item.variant_label,
    price: toMajor(item.unit_price),
    quantity: item.quantity,
  };
}

// ---- Catalog browsing ------------------------------------------------------

export function trackViewItemList(products: StorefrontProduct[], listId: string, listName: string): void {
  track("view_item_list", {
    item_list_id: listId,
    item_list_name: listName,
    items: products.map((p, index) => productToItem(p, { index, listId, listName })),
  });
}

export function trackSelectItem(
  product: StorefrontProduct,
  listId: string,
  listName: string,
  index?: number,
): void {
  track("select_item", {
    item_list_id: listId,
    item_list_name: listName,
    items: [productToItem(product, { index, listId, listName })],
  });
}

export function trackViewItem(product: StorefrontProduct, variantLabel?: string): void {
  const price = effectivePrice(product);
  track("view_item", {
    currency: price.currency,
    value: toMajor(price),
    items: [productToItem(product, { quantity: 1, variantLabel })],
  });
}

// ---- Cart & wishlist -------------------------------------------------------

export function trackAddToCart(product: StorefrontProduct, quantity: number, variantLabel?: string): void {
  const price = effectivePrice(product);
  track("add_to_cart", {
    currency: price.currency,
    value: toMajor(price) * quantity,
    items: [productToItem(product, { quantity, variantLabel })],
  });
}

export function trackRemoveFromCart(item: CartItem): void {
  track("remove_from_cart", {
    currency: item.unit_price.currency,
    value: toMajor(item.unit_price) * item.quantity,
    items: [cartItemToItem(item)],
  });
}

export function trackViewCart(cart: Cart): void {
  track("view_cart", {
    currency: cart.subtotal.currency,
    value: toMajor(cart.subtotal),
    items: cart.items.map(cartItemToItem),
  });
}

export function trackAddToWishlist(product: StorefrontProduct): void {
  const price = effectivePrice(product);
  track("add_to_wishlist", {
    currency: price.currency,
    value: toMajor(price),
    items: [productToItem(product, { quantity: 1 })],
  });
}

// ---- Checkout funnel -------------------------------------------------------

export function trackBeginCheckout(cart: Cart): void {
  track("begin_checkout", {
    currency: cart.subtotal.currency,
    value: toMajor(cart.subtotal),
    items: cart.items.map(cartItemToItem),
  });
}

// shippingTier is the chosen delivery method code (e.g. "speedy_office").
export function trackAddShippingInfo(cart: Cart, shippingTier: string): void {
  track("add_shipping_info", {
    currency: cart.subtotal.currency,
    value: toMajor(cart.subtotal),
    shipping_tier: shippingTier,
    items: cart.items.map(cartItemToItem),
  });
}

// paymentType is the chosen payment method code (e.g. "card", "cod").
export function trackAddPaymentInfo(cart: Cart, paymentType: string): void {
  track("add_payment_info", {
    currency: cart.subtotal.currency,
    value: toMajor(cart.subtotal),
    payment_type: paymentType,
    items: cart.items.map(cartItemToItem),
  });
}

export type PurchaseParams = {
  transactionId: string;
  currency: string;
  value: number; // major units
  tax?: number; // major units
  shipping?: number; // major units
  items: GAItem[];
};

// trackPurchase fires the revenue event. The caller builds the item list and
// totals from the placed order; de-duplication (one purchase per order, even if
// the confirmation page re-renders) is the caller's responsibility.
export function trackPurchase(params: PurchaseParams): void {
  track("purchase", {
    transaction_id: params.transactionId,
    currency: params.currency,
    value: params.value,
    tax: params.tax,
    shipping: params.shipping,
    items: params.items,
  });
}

// ---- Account ---------------------------------------------------------------

export function trackSignUp(method = "Google"): void {
  track("sign_up", { method });
}

export function trackLogin(method = "Google"): void {
  track("login", { method });
}
