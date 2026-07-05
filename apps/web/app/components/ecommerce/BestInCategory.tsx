import { useEffect, useState } from "react";

import { useAuth } from "../../features/auth/AuthContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { useWishlist } from "../../features/wishlist/WishlistContext";
import type { HomeSectionConfig } from "../../lib/api/admin-home-sections";
import { type StorefrontProduct, getBestInCategoryProducts, resolveImageUrl } from "../../lib/api/storefront";
import { Eyebrow, Heading } from "../ui/Text";
import { ProductCard } from "./ProductCard";

type Props = {
  section: HomeSectionConfig;
};

export function BestInCategory({ section }: Props) {
  const { locale } = useLanguage();
  const { isAuthenticated } = useAuth();
  const { isWishlisted, toggle } = useWishlist();
  const [products, setProducts] = useState<StorefrontProduct[] | null>(null);

  useEffect(() => {
    getBestInCategoryProducts(locale)
      .then(setProducts)
      .catch(() => setProducts([]));
  }, [locale]);

  if (products !== null && products.length === 0) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      {section.eyebrow && <Eyebrow>{section.eyebrow}</Eyebrow>}
      <Heading as="h2" size="lg" className="mt-2">
        {section.heading}
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
                outOfStock={!product.in_stock}
                isWishlisted={isAuthenticated && isWishlisted(product.id)}
                onToggleWishlist={isAuthenticated ? () => toggle(product.id) : undefined}
              />
            ))}
      </div>
    </section>
  );
}
