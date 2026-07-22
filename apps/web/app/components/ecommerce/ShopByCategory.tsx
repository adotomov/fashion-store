import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { type NavCategory, type NavType, getNav, resolveImageUrl } from "../../lib/api/storefront";
import { Badge } from "../ui/Badge";
import { Eyebrow, Heading } from "../ui/Text";
import { CardSlider } from "./CardSlider";

export function ShopByCategory() {
  const { locale, t } = useLanguage();
  const [navTypes, setNavTypes] = useState<NavType[] | null>(null);

  useEffect(() => {
    getNav(locale)
      .then(setNavTypes)
      .catch(() => setNavTypes([]));
  }, [locale]);

  const types = (navTypes ?? []).filter((type) => type.categories.length > 0);

  if (navTypes !== null && types.length === 0) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <Eyebrow>{t("home.browse", "Browse")}</Eyebrow>
      <Heading as="h2" size="lg" className="mt-2">
        {t("home.shop_by_category", "Shop by Category")}
      </Heading>

      {navTypes === null ? (
        <div className="mt-8 grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="aspect-square animate-pulse rounded-sm bg-stone-100" />
          ))}
        </div>
      ) : (
        <div className="mt-8 flex flex-col gap-12">
          {types.map((type) => (
            <CardSlider
              key={type.id}
              title={type.name}
              items={type.categories}
              getKey={(category) => category.id}
              renderItem={(category) => <CategoryCard category={category} />}
            />
          ))}
        </div>
      )}
    </section>
  );
}

function CategoryCard({ category }: { category: NavCategory }) {
  const { t } = useLanguage();

  return (
    <Link
      to={`/shop?category_id=${category.id}`}
      state={{ resetFilters: true }}
      className="group relative block aspect-square w-full overflow-hidden rounded-sm bg-stone-200"
    >
      {category.image_url ? (
        <img
          src={resolveImageUrl(category.image_url)}
          alt=""
          className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
        />
      ) : (
        <span className="block h-full w-full bg-gradient-to-br from-stone-400 to-stone-600" />
      )}

      {/* Dimming layer — keeps the white category name legible over both real
          photography and the gradient placeholder. */}
      <span className="absolute inset-0 bg-stone-900/40 transition-colors group-hover:bg-stone-900/30" />

      {category.has_promotion && (
        <span className="absolute left-2 top-2">
          <Badge variant="accent">{t("product.sale_badge", "Sale")}</Badge>
        </span>
      )}

      <span className="absolute inset-0 flex items-center justify-center p-3">
        <span className="text-center font-display text-xl font-medium tracking-wide text-white drop-shadow-[0_1px_6px_rgba(0,0,0,0.45)] sm:text-2xl">
          {category.name}
        </span>
      </span>
    </Link>
  );
}
