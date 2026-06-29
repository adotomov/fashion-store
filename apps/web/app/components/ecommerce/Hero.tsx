import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { type StorefrontProduct, listStorefrontProducts, resolveImageUrl } from "../../lib/api/storefront";
import { cn } from "../../lib/utils/cn";
import { buttonStyles } from "../ui/Button";
import { Price } from "../ui/Price";
import { Eyebrow, Heading, Text } from "../ui/Text";

export function Hero() {
  const { locale } = useLanguage();
  const [highlight, setHighlight] = useState<StorefrontProduct | null>(null);

  useEffect(() => {
    listStorefrontProducts({ limit: 1, locale })
      .then((products) => setHighlight(products[0] ?? null))
      .catch(() => {});
  }, [locale]);

  return (
    <section className="relative isolate overflow-hidden bg-stone-950">
      {/* Mesh gradient backdrop: layered radial blobs in the brand palette,
          darkened with a top-to-bottom scrim so the headline stays legible
          without needing a licensed photograph. */}
      <div
        className="absolute inset-0"
        style={{
          backgroundImage: [
            "radial-gradient(ellipse 60% 50% at 20% 20%, rgba(178,84,60,0.55), transparent 60%)",
            "radial-gradient(ellipse 50% 60% at 85% 15%, rgba(92,122,82,0.45), transparent 60%)",
            "radial-gradient(ellipse 70% 60% at 50% 100%, rgba(163,144,127,0.5), transparent 60%)",
            "linear-gradient(135deg, #1f1f1f 0%, #323232 50%, #1f1f1f 100%)",
          ].join(", "),
        }}
      />
      <div className="absolute inset-0 bg-gradient-to-t from-stone-950 via-stone-950/40 to-stone-950/10" />

      <div className="relative mx-auto flex min-h-[34rem] max-w-7xl flex-col justify-center px-4 py-20 sm:px-6 lg:min-h-[40rem] lg:px-8">
        <div className="max-w-xl">
          <Eyebrow className="text-clay-100/90">New Season</Eyebrow>
          <Heading as="h1" size="xl" className="mt-3 text-white">
            Quietly considered style, for every day.
          </Heading>
          <Text size="lg" className="mt-4 max-w-md text-stone-200">
            Clothing, jewelry, bags, and accessories — thoughtfully made, finished by hand, and built to last beyond
            a single season.
          </Text>
          <div className="mt-8 flex flex-wrap items-center gap-4">
            <Link
              to="/shop"
              state={{ resetFilters: true }}
              className={cn(buttonStyles({ variant: "primary", size: "lg" }), "bg-white text-stone-900 hover:bg-stone-100")}
            >
              Shop All Items
            </Link>
            <Link
              to="/shop?sale=true"
              state={{ resetFilters: true }}
              className={cn(buttonStyles({ variant: "outline", size: "lg" }), "border-white/40 text-white hover:bg-white/10")}
            >
              View the Sale
            </Link>
          </div>
        </div>

        {highlight && (
          <Link
            to="/shop"
            state={{ resetFilters: true }}
            className="absolute bottom-8 right-4 hidden w-64 items-center gap-3 rounded-sm bg-white/95 p-3 shadow-xl backdrop-blur transition-transform hover:-translate-y-0.5 sm:right-6 sm:flex lg:right-8"
          >
            <span className="block aspect-square w-16 shrink-0 overflow-hidden rounded-sm bg-stone-100">
              {highlight.image_url ? (
                <img
                  src={resolveImageUrl(highlight.image_url)}
                  alt={highlight.name}
                  className="h-full w-full object-cover"
                />
              ) : (
                <span className="flex h-full w-full items-center justify-center font-display text-xl text-stone-400">
                  {highlight.name.charAt(0).toUpperCase()}
                </span>
              )}
            </span>
            <span className="flex flex-col gap-1">
              <Text size="xs" tone="muted" className="uppercase tracking-wide">
                Just In
              </Text>
              <Text size="sm" className="font-medium leading-tight">
                {highlight.name}
              </Text>
              <Price price={highlight.base_price} compareAtPrice={highlight.compare_at_price} size="sm" />
            </span>
          </Link>
        )}
      </div>
    </section>
  );
}
