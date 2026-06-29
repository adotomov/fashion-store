import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { type StorefrontCatalog, listStorefrontCatalogs } from "../../lib/api/storefront";
import { Icon } from "../ui/Icon";
import { Eyebrow, Heading, Text } from "../ui/Text";

const gradients = [
  "from-clay-50 to-clay-100",
  "from-sage-50 to-sage-100",
  "from-stone-100 to-stone-200",
];

export function ShopByCollection() {
  const { locale } = useLanguage();
  const [catalogs, setCatalogs] = useState<StorefrontCatalog[] | null>(null);

  useEffect(() => {
    listStorefrontCatalogs(locale)
      .then(setCatalogs)
      .catch(() => setCatalogs([]));
  }, [locale]);

  if (catalogs !== null && catalogs.length === 0) return null;

  return (
    <section className="bg-stone-50 py-16">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <Eyebrow>Curated</Eyebrow>
        <Heading as="h2" size="lg" className="mt-2">
          Shop by Collection
        </Heading>

        <div className="mt-8 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {catalogs === null
            ? Array.from({ length: 3 }).map((_, i) => (
                <div key={i} className="h-48 animate-pulse rounded-sm bg-stone-200" />
              ))
            : catalogs.map((catalog, i) => (
                <Link
                  key={catalog.id}
                  to={`/shop?catalog_id=${catalog.id}`}
                  state={{ resetFilters: true }}
                  className={`group flex h-48 flex-col justify-end rounded-sm bg-gradient-to-br p-6 transition-transform hover:-translate-y-0.5 ${gradients[i % gradients.length]}`}
                >
                  <Heading as="h3" size="sm">
                    {catalog.name}
                  </Heading>
                  <Text size="sm" tone="muted" className="mt-1 flex items-center gap-1.5 group-hover:text-clay-600">
                    Shop the collection
                    <Icon name="chevronRight" size={14} />
                  </Text>
                </Link>
              ))}
        </div>
      </div>
    </section>
  );
}
