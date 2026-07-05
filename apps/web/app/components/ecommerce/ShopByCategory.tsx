import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { type NavType, getNav, resolveImageUrl } from "../../lib/api/storefront";
import { Badge } from "../ui/Badge";
import { Eyebrow, Heading, Text } from "../ui/Text";

export function ShopByCategory() {
  const { locale, t } = useLanguage();
  const [navTypes, setNavTypes] = useState<NavType[] | null>(null);

  useEffect(() => {
    getNav(locale)
      .then(setNavTypes)
      .catch(() => setNavTypes([]));
  }, [locale]);

  const categories = (navTypes ?? []).flatMap((type) => type.categories.map((category) => ({ type, category })));

  if (navTypes !== null && categories.length === 0) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <Eyebrow>{t("home.browse", "Browse")}</Eyebrow>
      <Heading as="h2" size="lg" className="mt-2">
        {t("home.shop_by_category", "Shop by Category")}
      </Heading>

      <div className="mt-8 grid grid-cols-2 gap-6 sm:grid-cols-3 lg:grid-cols-6">
        {navTypes === null
          ? Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="aspect-square animate-pulse rounded-sm bg-stone-100" />
            ))
          : categories.map(({ type, category }) => (
              <Link
                key={category.id}
                to={`/shop?category_id=${category.id}`}
                state={{ resetFilters: true }}
                className="group flex flex-col gap-2.5"
              >
                {category.image_url ? (
                  <span className="relative block aspect-square w-full overflow-hidden rounded-sm bg-stone-100">
                    <img
                      src={resolveImageUrl(category.image_url)}
                      alt={category.name}
                      className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
                    />
                    {category.has_promotion && (
                      <span className="absolute left-1.5 top-1.5">
                        <Badge variant="accent">{t("product.sale_badge", "Sale")}</Badge>
                      </span>
                    )}
                  </span>
                ) : (
                  <span className="relative flex aspect-square w-full items-center justify-center overflow-hidden rounded-sm bg-gradient-to-br from-stone-100 to-stone-200 transition-colors group-hover:from-clay-50 group-hover:to-clay-100">
                    <span className="font-display text-3xl text-stone-400 group-hover:text-clay-500">
                      {category.name.charAt(0).toUpperCase()}
                    </span>
                    {category.has_promotion && (
                      <span className="absolute left-1.5 top-1.5">
                        <Badge variant="accent">{t("product.sale_badge", "Sale")}</Badge>
                      </span>
                    )}
                  </span>
                )}
                <Text size="sm" className="text-center font-medium group-hover:text-clay-600">
                  {category.name}
                </Text>
                <Text size="xs" tone="muted" className="-mt-2 text-center">
                  {type.name}
                </Text>
              </Link>
            ))}
      </div>
    </section>
  );
}
