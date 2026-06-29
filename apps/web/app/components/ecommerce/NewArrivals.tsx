import { useEffect, useState } from "react";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { type StorefrontProduct, listStorefrontProducts, resolveImageUrl } from "../../lib/api/storefront";
import { Eyebrow, Heading } from "../ui/Text";
import { ProductCard } from "./ProductCard";

export function NewArrivals() {
  const { locale } = useLanguage();
  const [products, setProducts] = useState<StorefrontProduct[] | null>(null);

  useEffect(() => {
    listStorefrontProducts({ limit: 8, locale })
      .then(setProducts)
      .catch(() => setProducts([]));
  }, [locale]);

  if (products !== null && products.length === 0) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <Eyebrow>New In</Eyebrow>
      <Heading as="h2" size="lg" className="mt-2">
        New Arrivals
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
              />
            ))}
      </div>
    </section>
  );
}
