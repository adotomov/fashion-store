import { useEffect, useState } from "react";

import { useAuth } from "../../features/auth/AuthContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { useWishlist } from "../../features/wishlist/WishlistContext";
import type { HomeSectionConfig } from "../../lib/api/admin-home-sections";
import { type CategoryProductGroup, getCategoryProductGroups, resolveImageUrl } from "../../lib/api/storefront";
import { Eyebrow, Heading } from "../ui/Text";
import { CardSlider } from "./CardSlider";
import { ProductCard } from "./ProductCard";

type Props = {
  section: HomeSectionConfig;
};

export function BestInCategory({ section }: Props) {
  const { locale } = useLanguage();
  const { isAuthenticated } = useAuth();
  const { isWishlisted, toggle } = useWishlist();
  const [groups, setGroups] = useState<CategoryProductGroup[] | null>(null);

  useEffect(() => {
    getCategoryProductGroups(section.id, { locale, limit: 10 })
      .then(setGroups)
      .catch(() => setGroups([]));
  }, [section.id, locale]);

  if (groups !== null && groups.length === 0) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      {section.eyebrow && <Eyebrow>{section.eyebrow}</Eyebrow>}
      <Heading as="h2" size="lg" className="mt-2">
        {section.heading}
      </Heading>

      {groups === null ? (
        <div className="mt-8 grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-3 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="aspect-[3/4] animate-pulse rounded-sm bg-stone-100" />
          ))}
        </div>
      ) : (
        <div className="mt-8 flex flex-col gap-12">
          {groups.map((group) => (
            <CardSlider
              key={group.category_id}
              title={group.category_name}
              items={group.products}
              getKey={(product) => product.id}
              renderItem={(product) => (
                <ProductCard
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
              )}
            />
          ))}
        </div>
      )}
    </section>
  );
}
