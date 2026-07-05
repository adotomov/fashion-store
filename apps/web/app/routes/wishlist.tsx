import { Link } from "react-router";

import { EmptyState } from "../components/admin/EmptyState";
import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { RequireAuth } from "../components/RequireAuth";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Icon } from "../components/ui/Icon";
import { Price } from "../components/ui/Price";
import { Heading, Text } from "../components/ui/Text";
import { useLanguage } from "../features/i18n/LanguageContext";
import { useWishlist } from "../features/wishlist/WishlistContext";
import { resolveImageUrl } from "../lib/api/storefront";
import { cn } from "../lib/utils/cn";

export const handle = { title: "Wishlist" };

export default function Wishlist() {
  const { t } = useLanguage();
  const { items, isLoading, toggle } = useWishlist();

  return (
    <RequireAuth>
      <div className="flex min-h-screen flex-col">
        <Header />
        <main className="flex-1">
          <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6 lg:px-8">
            <Heading as="h1" size="lg">
              {t("wishlist.title", "Wishlist")}
            </Heading>

            <div className="mt-8">
              {isLoading ? (
                <Text size="sm" tone="muted">
                  {t("common.loading", "Loading…")}
                </Text>
              ) : items.length === 0 ? (
                <EmptyState
                  icon="wishlist"
                  title={t("wishlist.empty_title", "Your wishlist is empty")}
                  description={t("wishlist.empty_desc", "Save items you love by tapping the heart on any product.")}
                />
              ) : (
                <ul className="flex flex-col gap-4">
                  {items.map((item) => (
                    <li key={item.id} className="flex gap-4 rounded-sm border border-stone-200 bg-white p-4">
                      <Link
                        to={`/shop/${item.product_slug}`}
                        className="relative block h-24 w-20 flex-shrink-0 overflow-hidden rounded-sm bg-stone-50"
                      >
                        {item.image_url ? (
                          <img
                            src={resolveImageUrl(item.image_url)}
                            alt={item.product_name}
                            className={cn("h-full w-full object-cover", !item.in_stock && "opacity-60")}
                          />
                        ) : (
                          <span className="flex h-full w-full items-center justify-center bg-gradient-to-br from-stone-100 to-stone-200">
                            <span className="font-display text-2xl text-stone-400">
                              {item.product_name.charAt(0).toUpperCase()}
                            </span>
                          </span>
                        )}
                        {!item.in_stock && (
                          <Badge variant="danger" className="absolute left-1 top-1 px-1.5 py-0.5 text-[10px]">
                            {t("product.sold_out", "Sold Out")}
                          </Badge>
                        )}
                      </Link>

                      <div className="flex flex-1 flex-col justify-between">
                        <div>
                          <Link to={`/shop/${item.product_slug}`}>
                            <Text className="font-medium">{item.product_name}</Text>
                          </Link>
                          {item.sizes.length > 0 && (
                            <Text size="sm" tone="muted" className="mt-1">
                              {t("wishlist.sizes_label", "Sizes:")} {item.sizes.join(", ")}
                            </Text>
                          )}
                        </div>
                        <div className="mt-2 flex items-center gap-3">
                          <Price
                            price={item.promotion_price ?? item.base_price}
                            compareAtPrice={item.promotion_price ? item.base_price : item.compare_at_price}
                            size="sm"
                          />
                          <Badge variant={item.in_stock ? "success" : "danger"}>
                            {item.in_stock ? t("product.available", "Available") : t("product.sold_out", "Sold Out")}
                          </Badge>
                        </div>
                      </div>

                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={t("wishlist.remove", "Remove from wishlist")}
                        title={t("wishlist.remove", "Remove from wishlist")}
                        onClick={() => toggle(item.product_id)}
                        className="self-start text-danger-600 hover:bg-danger-50"
                      >
                        <Icon name="trash" size={16} />
                      </Button>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </div>
        </main>
        <Footer />
      </div>
    </RequireAuth>
  );
}
