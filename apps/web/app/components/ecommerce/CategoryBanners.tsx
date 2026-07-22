import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { type NavType, getNav, resolveImageUrl } from "../../lib/api/storefront";
import { Icon } from "../ui/Icon";
import { CardSlider } from "./CardSlider";

// Gradient placeholders used when a department has no category imagery yet —
// rotated so adjacent tiles never share the same tone.
const gradients = [
  "from-clay-300 to-clay-500",
  "from-sage-300 to-sage-500",
  "from-stone-400 to-stone-600",
  "from-clay-400 to-sage-500",
];

// Large "shop the department" banners: one big tile per parent type, shown high
// on the page as a bold navigational entry point. Distinct from ShopByCategory,
// which is a per-category slider deeper down. Presented as a headless slider so
// that more than a rowful of departments pages horizontally with arrows rather
// than wrapping onto a second row.
export function CategoryBanners() {
  const { locale } = useLanguage();
  const [navTypes, setNavTypes] = useState<NavType[] | null>(null);

  useEffect(() => {
    getNav(locale)
      .then(setNavTypes)
      .catch(() => setNavTypes([]));
  }, [locale]);

  const types = (navTypes ?? []).filter((type) => type.categories.length > 0);

  if (navTypes !== null && types.length === 0) return null;

  // Assign gradients by absolute position so a department keeps its tone as the
  // slider pages, rather than shifting colour with its on-screen slot.
  const tiles = types.map((type, i) => ({ type, gradient: gradients[i % gradients.length] }));

  return (
    <section className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
      {navTypes === null ? (
        <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="aspect-[3/4] animate-pulse rounded-sm bg-stone-100" />
          ))}
        </div>
      ) : (
        <CardSlider
          items={tiles}
          getKey={({ type }) => type.id}
          renderItem={({ type, gradient }) => <DepartmentTile type={type} gradient={gradient} />}
        />
      )}
    </section>
  );
}

function DepartmentTile({ type, gradient }: { type: NavType; gradient: string }) {
  const { t } = useLanguage();
  // Borrow the first available category image as the department backdrop.
  const backdrop = type.categories.find((c) => c.image_url)?.image_url;

  return (
    <Link
      to={`/shop?type=${type.slug}`}
      state={{ resetFilters: true }}
      className="group relative block aspect-[3/4] w-full overflow-hidden rounded-sm bg-stone-200"
    >
      {backdrop ? (
        <img
          src={resolveImageUrl(backdrop)}
          alt=""
          className="h-full w-full object-cover transition-transform duration-500 group-hover:scale-105"
        />
      ) : (
        <span className={`block h-full w-full bg-gradient-to-br ${gradient}`} />
      )}

      {/* Bottom-weighted scrim keeps the label legible over any backdrop. */}
      <span className="absolute inset-0 bg-gradient-to-t from-stone-950/70 via-stone-950/10 to-transparent transition-colors group-hover:from-stone-950/60" />

      <span className="absolute inset-x-0 bottom-0 flex flex-col gap-1 p-5">
        <span className="font-display text-2xl font-medium tracking-wide text-white drop-shadow-[0_1px_6px_rgba(0,0,0,0.45)]">
          {type.name}
        </span>
        <span className="flex items-center gap-1.5 text-sm font-medium text-white/85 transition-colors group-hover:text-white">
          {t("home.shop_now", "Shop now")}
          <Icon name="chevronRight" size={15} className="transition-transform group-hover:translate-x-0.5" />
        </span>
      </span>
    </Link>
  );
}
