import { useState } from "react";
import { Link } from "react-router";

import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { buttonStyles } from "../components/ui/Button";
import { Icon } from "../components/ui/Icon";
import { Price } from "../components/ui/Price";
import { Heading, Text } from "../components/ui/Text";
import { useLanguage } from "../features/i18n/LanguageContext";
import { useStoreBranding } from "../features/store-settings/StoreSettingsContext";
import { useCart } from "../features/cart/CartContext";
import type { CartItem } from "../lib/api/cart";
import { resolveImageUrl } from "../lib/api/storefront";
import { formatMoneyDual } from "../lib/money/money";

export const handle = { title: "Cart" };

export default function CartPage() {
  const { t } = useLanguage();
  const { storeLocale } = useStoreBranding();
  const { cart, isLoading } = useCart();
  const items = cart?.items ?? [];

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1 bg-stone-50">
        <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
          <Heading as="h1" size="lg">
            {t("cart.title", "Your Cart")}
          </Heading>

          {isLoading ? (
            <Text size="sm" tone="muted" className="py-16 text-center">
              {t("common.loading", "Loading…")}
            </Text>
          ) : items.length === 0 ? (
            <div className="flex flex-col items-center gap-4 py-16 text-center">
              <Text tone="muted">{t("cart.empty", "Your cart is empty")}</Text>
              <Link to="/shop" className={buttonStyles({ variant: "primary" })}>
                {t("common.continue_shopping", "Continue Shopping")}
              </Link>
            </div>
          ) : (
            <div className="mt-8 grid grid-cols-1 gap-10 lg:grid-cols-[1fr_320px]">
              <ul className="flex flex-col gap-4">
                {items.map((item) => (
                  <CartLineItem key={item.id} item={item} />
                ))}
              </ul>

              <div className="h-fit rounded-sm border border-stone-200 bg-white p-6">
                <Heading as="h2" size="sm">
                  {t("cart.summary", "Summary")}
                </Heading>
                <div className="mt-4 flex items-center justify-between border-t border-stone-200 pt-4">
                  <Text size="sm" className="font-medium">
                    {t("cart.subtotal", "Subtotal")}
                  </Text>
                  <Text size="sm" className="font-medium">
                    {formatMoneyDual(cart!.subtotal, storeLocale)}
                  </Text>
                </div>
                <Link to="/checkout" className={buttonStyles({ variant: "primary", size: "lg", className: "mt-6 w-full" })}>
                  {t("cart.checkout", "Checkout")}
                </Link>
              </div>
            </div>
          )}
        </div>
      </main>
      <Footer />
    </div>
  );
}

function CartLineItem({ item }: { item: CartItem }) {
  const { t } = useLanguage();
  const { updateQuantity, removeItem } = useCart();
  const [isUpdating, setIsUpdating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function changeQuantity(quantity: number) {
    if (quantity < 1) return;
    setIsUpdating(true);
    setError(null);
    try {
      await updateQuantity(item.id, quantity);
    } catch {
      setError(t("cart.update_error", "Could not update quantity."));
    } finally {
      setIsUpdating(false);
    }
  }

  async function handleRemove() {
    setIsUpdating(true);
    setError(null);
    try {
      await removeItem(item.id);
    } catch {
      setError(t("cart.remove_error", "Could not remove this item."));
      setIsUpdating(false);
    }
  }

  return (
    <li className="flex gap-4 rounded-sm border border-stone-200 bg-white p-4">
      <Link
        to={`/shop/${item.product_slug}`}
        className="block h-24 w-20 shrink-0 overflow-hidden rounded-sm bg-stone-100"
      >
        {item.image_url && (
          <img src={resolveImageUrl(item.image_url)} alt={item.product_name} className="h-full w-full object-cover" />
        )}
      </Link>

      <div className="flex flex-1 flex-col gap-1">
        <div className="flex items-start justify-between gap-3">
          <div>
            <Link to={`/shop/${item.product_slug}`} className="font-medium text-stone-900 hover:underline">
              {item.product_name}
            </Link>
            {item.variant_label && (
              <Text size="sm" tone="muted">
                {item.variant_label}
              </Text>
            )}
          </div>
          <Price price={item.unit_price} size="sm" />
        </div>

        {item.quantity > item.available_quantity && (
          <Text size="sm" tone="danger">
            {t("cart.only", "Only")} {item.available_quantity} {t("cart.left_in_stock", "left in stock")}
          </Text>
        )}
        {error && (
          <Text size="sm" tone="danger">
            {error}
          </Text>
        )}

        <div className="mt-auto flex items-center justify-between">
          <div className="flex items-center gap-2 rounded-sm border border-stone-300">
            <button
              type="button"
              aria-label={t("cart.decrease_qty", "Decrease quantity")}
              disabled={isUpdating}
              onClick={() => changeQuantity(item.quantity - 1)}
              className="flex h-8 w-8 items-center justify-center text-stone-600 hover:bg-stone-50 disabled:opacity-50"
            >
              <Icon name="minus" size={14} />
            </button>
            <Text size="sm" className="w-6 text-center">
              {item.quantity}
            </Text>
            <button
              type="button"
              aria-label={t("cart.increase_qty", "Increase quantity")}
              disabled={isUpdating}
              onClick={() => changeQuantity(item.quantity + 1)}
              className="flex h-8 w-8 items-center justify-center text-stone-600 hover:bg-stone-50 disabled:opacity-50"
            >
              <Icon name="plus" size={14} />
            </button>
          </div>

          <button
            type="button"
            aria-label={t("cart.remove_item", "Remove item")}
            disabled={isUpdating}
            onClick={handleRemove}
            className="rounded-sm p-2 text-stone-500 hover:bg-stone-50 hover:text-danger-500 disabled:opacity-50"
          >
            <Icon name="trash" size={16} />
          </button>
        </div>
      </div>
    </li>
  );
}
