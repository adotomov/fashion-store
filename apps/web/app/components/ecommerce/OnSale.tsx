import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useAuth } from "../../features/auth/AuthContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { useWishlist } from "../../features/wishlist/WishlistContext";
import type { HomeSectionConfig } from "../../lib/api/admin-home-sections";
import { type StorefrontProduct, listStorefrontProducts, resolveImageUrl } from "../../lib/api/storefront";
import { Icon } from "../ui/Icon";
import { Eyebrow, Heading } from "../ui/Text";
import { ProductCard } from "./ProductCard";

type Props = {
  section: HomeSectionConfig;
};

// Two rows of four. When there are more discounted products than fit, the last
// tile becomes a "View All" link instead of a ninth product.
const MAX_TILES = 8;

export function OnSale({ section }: Props) {
  const { locale, t } = useLanguage();
  const { isAuthenticated } = useAuth();
  const { isWishlisted, toggle } = useWishlist();
  const [products, setProducts] = useState<StorefrontProduct[] | null>(null);

  useEffect(() => {
    listStorefrontProducts({ hasPromotion: true, locale })
      .then(setProducts)
      .catch(() => setProducts([]));
  }, [locale]);

  if (products !== null && products.length === 0) return null;

  const total = products?.length ?? 0;
  const showViewAll = total > MAX_TILES;
  // Leave a slot for the View All tile only when we're actually overflowing.
  const visibleProducts = products
    ? products.slice(0, showViewAll ? MAX_TILES - 1 : MAX_TILES)
    : null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      {section.eyebrow && <Eyebrow className="text-clay-600">{section.eyebrow}</Eyebrow>}
      <Heading as="h2" size="lg" className="mt-2">
        {section.heading}
      </Heading>

      <div className="mt-8 grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-3 lg:grid-cols-4">
        {visibleProducts === null
          ? Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="aspect-[3/4] animate-pulse rounded-sm bg-stone-100" />
            ))
          : visibleProducts.map((product) => (
              <ProductCard
                key={product.id}
                href={`/shop/${product.slug}`}
                image={product.image_url ? { src: resolveImageUrl(product.image_url), alt: product.name } : undefined}
                title={product.name}
                price={product.base_price}
                compareAtPrice={product.compare_at_price}
                promotionPrice={product.promotion_price}
                promotionLabel={product.promotion_label}
                badge="Sale"
                outOfStock={!product.in_stock}
                isWishlisted={isAuthenticated && isWishlisted(product.id)}
                onToggleWishlist={isAuthenticated ? () => toggle(product.id) : undefined}
              />
            ))}
        {showViewAll && <ViewAllTile label={t("home.view_all_discounted", "View All Discounted Products")} />}
      </div>
    </section>
  );
}

function ViewAllTile({ label }: { label: string }) {
  return (
    <Link
      to="/shop?sale=true"
      state={{ resetFilters: true }}
      className="group flex aspect-[3/4] flex-col items-center justify-center gap-3 rounded-sm border border-dashed border-stone-300 bg-stone-50 p-4 text-center transition-colors hover:border-clay-400 hover:bg-clay-50"
    >
      <span className="flex h-11 w-11 items-center justify-center rounded-full bg-white text-stone-700 shadow-sm transition-colors group-hover:text-clay-600">
        <Icon name="chevronRight" size={20} />
      </span>
      <span className="font-medium text-stone-800 group-hover:text-clay-700">{label}</span>
    </Link>
  );
}
