import { useEffect, useState } from "react";
import { Link } from "react-router";

import { getPublicEditorialBanner, type EditorialBannerSettings } from "../../lib/api/admin-appearance";
import { resolveImageUrl } from "../../lib/api/storefront";
import { cn } from "../../lib/utils/cn";
import { buttonStyles } from "../ui/Button";
import { Eyebrow, Heading, Text } from "../ui/Text";

// Full-bleed editorial ("Shop the Look") banner shown mid-home-page to break
// the rhythm of product-grid sections with a single large lifestyle image and
// a clear call to action. Admin-configurable; renders nothing until enabled.
export function EditorialBanner() {
  const [banner, setBanner] = useState<EditorialBannerSettings | null>(null);

  useEffect(() => {
    getPublicEditorialBanner()
      .then(setBanner)
      .catch(() => setBanner(null));
  }, []);

  if (!banner || !banner.enabled) return null;

  return (
    <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
      <div className="relative isolate overflow-hidden rounded-lg bg-stone-950">
        {banner.image_url ? (
          <div
            className="absolute inset-0 bg-cover bg-center transition-transform duration-700"
            style={{ backgroundImage: `url(${resolveImageUrl(banner.image_url)})` }}
          />
        ) : (
          <div
            className="absolute inset-0"
            style={{
              backgroundImage: [
                "radial-gradient(ellipse 55% 60% at 15% 25%, rgba(92,122,82,0.5), transparent 60%)",
                "radial-gradient(ellipse 60% 55% at 85% 80%, rgba(178,84,60,0.5), transparent 60%)",
                "linear-gradient(120deg, #262626 0%, #3a3a3a 50%, #262626 100%)",
              ].join(", "),
            }}
          />
        )}
        <div className="absolute inset-0 bg-gradient-to-r from-stone-950/80 via-stone-950/40 to-transparent" />

        <div className="relative flex min-h-[22rem] flex-col justify-center px-6 py-16 sm:px-10 lg:min-h-[28rem] lg:px-16">
          <div className="max-w-lg">
            {banner.eyebrow && <Eyebrow className="text-clay-100/90">{banner.eyebrow}</Eyebrow>}
            <Heading as="h2" size="lg" className="mt-3 text-white">
              {banner.heading}
            </Heading>
            {banner.subtext && (
              <Text size="lg" className="mt-4 max-w-md text-stone-200">
                {banner.subtext}
              </Text>
            )}
            {banner.cta_label && banner.cta_url && (
              <div className="mt-8">
                <Link
                  to={banner.cta_url}
                  state={{ resetFilters: true }}
                  className={cn(
                    buttonStyles({ variant: "primary", size: "lg" }),
                    "bg-white text-stone-900 hover:bg-stone-100",
                  )}
                >
                  {banner.cta_label}
                </Link>
              </div>
            )}
          </div>
        </div>
      </div>
    </section>
  );
}
