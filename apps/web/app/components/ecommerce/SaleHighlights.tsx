import { useEffect, useState } from "react";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { type StorefrontProduct, listStorefrontProducts, resolveImageUrl } from "../../lib/api/storefront";
import { Eyebrow, Heading } from "../ui/Text";
import { ProductCard } from "./ProductCard";

// Only renders once real discounted products exist (compare_at_price set
// higher than base_price) — no fabricated sale data.
export function SaleHighlights() {
  const { locale } = useLanguage();
  const [onSale, setOnSale] = useState<StorefrontProduct[] | null>(null);

  useEffect(() => {
    listStorefrontProducts({ locale })
      .then((products) =>
        setOnSale(
          products.filter((p) => p.compare_at_price && p.compare_at_price.amount > p.base_price.amount),
        ),
      )
      .catch(() => setOnSale([]));
  }, [locale]);

  if (!onSale || onSale.length === 0) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <Eyebrow className="text-clay-600">Limited Time</Eyebrow>
      <Heading as="h2" size="lg" className="mt-2">
        On Sale
      </Heading>

      <div className="mt-8 grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-3 lg:grid-cols-4">
        {onSale.map((product) => (
          <ProductCard
            key={product.id}
            href={`/shop/${product.slug}`}
            image={product.image_url ? { src: resolveImageUrl(product.image_url), alt: product.name } : undefined}
            title={product.name}
            price={product.base_price}
            compareAtPrice={product.compare_at_price}
            badge="Sale"
            outOfStock={!product.in_stock}
          />
        ))}
      </div>
    </section>
  );
}
