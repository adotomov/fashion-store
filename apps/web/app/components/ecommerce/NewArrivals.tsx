import { useEffect, useState } from "react";

import { useAuth } from "../../features/auth/AuthContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { useWishlist } from "../../features/wishlist/WishlistContext";
import { type StorefrontProduct, listStorefrontProducts, resolveImageUrl } from "../../lib/api/storefront";
import { Eyebrow, Heading } from "../ui/Text";
import { ProductCard } from "./ProductCard";

export function NewArrivals() {
  const { locale, t } = useLanguage();
  const { isAuthenticated } = useAuth();
  const { isWishlisted, toggle } = useWishlist();
  const [products, setProducts] = useState<StorefrontProduct[] | null>(null);

  useEffect(() => {
    // A discounted (actively promoted) product belongs in the On Sale section,
    // not here — so we exclude anything with a promotion price. We over-fetch
    // and then trim to 8, since the exclusion happens client-side.
    listStorefrontProducts({ limit: 24, locale })
      .then((all) => setProducts(all.filter((p) => !p.promotion_price).slice(0, 8)))
      .catch(() => setProducts([]));
  }, [locale]);

  if (products !== null && products.length === 0) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <Eyebrow>{t("home.new_in", "New In")}</Eyebrow>
      <Heading as="h2" size="lg" className="mt-2">
        {t("home.new_arrivals", "New Arrivals")}
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
