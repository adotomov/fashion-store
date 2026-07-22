import { useEffect, useState } from "react";

import { useAuth } from "../../features/auth/AuthContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { useWishlist } from "../../features/wishlist/WishlistContext";
import { type StorefrontProduct, listStorefrontProducts, resolveImageUrl } from "../../lib/api/storefront";
import { getRecentlyViewedIds } from "../../lib/recentlyViewed";
import { Eyebrow, Heading } from "../ui/Text";
import { ProductCard } from "./ProductCard";

// One or two rows of the products this visitor recently opened. Browser-local:
// IDs come from localStorage, but the product data is re-fetched so prices and
// stock stay fresh. Renders nothing on a first visit with no history.
const MAX_TILES = 8;

export function RecentlyViewed() {
  const { locale, t } = useLanguage();
  const { isAuthenticated } = useAuth();
  const { isWishlisted, toggle } = useWishlist();
  const [products, setProducts] = useState<StorefrontProduct[] | null>(null);

  useEffect(() => {
    const ids = getRecentlyViewedIds();
    if (ids.length === 0) {
      setProducts([]);
      return;
    }
    listStorefrontProducts({ productIds: ids, locale })
      .then((fetched) => {
        // The API doesn't guarantee order, so restore most-recent-first and
        // drop anything no longer returned (deleted/inactive product).
        const byId = new Map(fetched.map((p) => [p.id, p]));
        const ordered = ids.map((id) => byId.get(id)).filter((p): p is StorefrontProduct => Boolean(p));
        setProducts(ordered.slice(0, MAX_TILES));
      })
      .catch(() => setProducts([]));
  }, [locale]);

  if (products !== null && products.length === 0) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <Eyebrow>{t("home.for_you", "For You")}</Eyebrow>
      <Heading as="h2" size="lg" className="mt-2">
        {t("home.recently_viewed", "Recently Viewed")}
      </Heading>

      <div className="mt-8 grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-3 lg:grid-cols-4">
        {products === null
          ? Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="aspect-[3/4] animate-pulse rounded-sm bg-stone-100" />
            ))
          : products.map((product) => (
              <ProductCard
                key={product.id}
                href={`/shop/${product.slug}`}
                image={product.image_url ? { src: resolveImageUrl(product.image_url), alt: product.name } : undefined}
                title={product.name}
                price={product.base_price}
                compareAtPrice={product.compare_at_price}
                promotionPrice={product.promotion_price}
                promotionLabel={product.promotion_label}
                outOfStock={!product.in_stock}
                isWishlisted={isAuthenticated && isWishlisted(product.id)}
                onToggleWishlist={isAuthenticated ? () => toggle(product.id) : undefined}
              />
            ))}
      </div>
    </section>
  );
}
