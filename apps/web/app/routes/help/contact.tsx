import { useEffect, useState } from "react";

import { Footer } from "../../components/ecommerce/Footer";
import { Header } from "../../components/ecommerce/Header";
import { Heading, Text } from "../../components/ui/Text";
import { Icon, type IconName } from "../../components/ui/Icon";
import { useLanguage } from "../../features/i18n/LanguageContext";
import {
  type StorefrontAddress,
  type StorefrontStoreSettings,
  getStoreSettings,
  listStorefrontAddresses,
  resolveImageUrl,
} from "../../lib/api/storefront";
import { buildMeta } from "../../lib/seo/meta";

export function meta() {
  return buildMeta({
    title: "Contact Us",
    description: "Get in touch with Verani — our contact details, opening hours, and store location.",
    path: "/help/contact",
  });
}

// One line of address, in the order most readable for a shopfront.
function formatAddress(a: StorefrontAddress): string {
  return [a.line1, a.line2, a.city, a.region, a.postal_code, a.country].filter(Boolean).join(", ");
}

export default function Contact() {
  const { t, locale } = useLanguage();
  const [settings, setSettings] = useState<StorefrontStoreSettings | null>(null);
  const [addresses, setAddresses] = useState<StorefrontAddress[]>([]);

  useEffect(() => {
    getStoreSettings().then(setSettings).catch(() => setSettings(null));
    listStorefrontAddresses().then(setAddresses).catch(() => setAddresses([]));
  }, []);

  const primaryAddress = addresses.find((a) => a.is_default) ?? addresses[0] ?? null;
  const addressLine = primaryAddress ? formatAddress(primaryAddress) : null;
  const storeImageUrl = settings?.store_image_url ? resolveImageUrl(settings.store_image_url) : null;

  // An explicit map_location (usually "lat,lng") pins the map exactly; otherwise
  // fall back to geocoding the store address.
  const mapQuery = settings?.map_location?.trim() || addressLine;

  const hasAnyDetail = Boolean(
    settings?.contact_email || settings?.contact_phone || addressLine || settings?.opening_hours,
  );

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        {/* Store photo */}
        <div className="relative h-64 w-full overflow-hidden bg-stone-100 sm:h-80 lg:h-[26rem]">
          {storeImageUrl ? (
            <img src={storeImageUrl} alt={t("contact.title", "Contact Us")} className="h-full w-full object-cover" />
          ) : (
            <div className="h-full w-full bg-gradient-to-br from-stone-200 via-stone-100 to-clay-100" />
          )}
        </div>

        <div className="mx-auto max-w-3xl px-4 py-14 sm:px-6 lg:px-8">
          <Heading as="h1" size="lg">
            {t("contact.title", "Contact Us")}
          </Heading>
          <Text className="mt-4 leading-relaxed" tone="muted">
            {t("contact.intro", "We'd love to hear from you. Visit us in store or reach out anytime.")}
          </Text>

          {!hasAnyDetail ? (
            <Text className="mt-8" tone="muted">
              {t("contact.unavailable", "Contact details are coming soon.")}
            </Text>
          ) : (
            <div className="mt-10 grid grid-cols-1 gap-10 sm:grid-cols-2">
              {/* Get in touch */}
              <section>
                <Heading as="h2" size="sm">
                  {t("contact.details_heading", "Get in touch")}
                </Heading>
                <ul className="mt-4 flex flex-col gap-4">
                  {settings?.contact_email && (
                    <ContactRow icon="mail" label={t("contact.email_label", "Email")}>
                      <a href={`mailto:${settings.contact_email}`} className="hover:text-stone-900">
                        {settings.contact_email}
                      </a>
                    </ContactRow>
                  )}
                  {settings?.contact_phone && (
                    <ContactRow icon="phone" label={t("contact.phone_label", "Phone")}>
                      <a href={`tel:${settings.contact_phone.replace(/\s+/g, "")}`} className="hover:text-stone-900">
                        {settings.contact_phone}
                      </a>
                    </ContactRow>
                  )}
                  {addressLine && (
                    <ContactRow icon="mapPin" label={t("contact.address_label", "Address")}>
                      {addressLine}
                    </ContactRow>
                  )}
                </ul>
              </section>

              {/* Opening hours */}
              {settings?.opening_hours && (
                <section>
                  <Heading as="h2" size="sm">
                    {t("contact.hours_heading", "Opening hours")}
                  </Heading>
                  <div className="mt-4 flex items-start gap-3">
                    <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-stone-100 text-clay-600">
                      <Icon name="clock" size={18} />
                    </span>
                    <Text size="sm" className="whitespace-pre-line leading-relaxed text-stone-700">
                      {settings.opening_hours}
                    </Text>
                  </div>
                </section>
              )}
            </div>
          )}

          {/* Map */}
          {mapQuery && (
            <section className="mt-14">
              <Heading as="h2" size="sm">
                {t("contact.find_us_heading", "Find us")}
              </Heading>
              <div className="mt-4 overflow-hidden rounded-sm border border-stone-200">
                <iframe
                  title={t("contact.find_us_heading", "Find us")}
                  // z=16 forces a street-level zoom centered on the query and
                  // iwloc drops a marker there — without them the keyless embed
                  // defaults to a wide, region-level view that looks like it
                  // "missed" the spot. hl localizes the map. mapQuery is an
                  // explicit lat,lng pin when set, else the store address.
                  src={`https://maps.google.com/maps?q=${encodeURIComponent(mapQuery)}&z=16&hl=${encodeURIComponent(locale)}&iwloc=&output=embed`}
                  loading="lazy"
                  referrerPolicy="no-referrer-when-downgrade"
                  className="h-80 w-full border-0"
                />
              </div>
            </section>
          )}
        </div>
      </main>
      <Footer />
    </div>
  );
}

function ContactRow({ icon, label, children }: { icon: IconName; label: string; children: React.ReactNode }) {
  return (
    <li className="flex items-start gap-3">
      <span className="mt-0.5 flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-stone-100 text-clay-600">
        <Icon name={icon} size={18} />
      </span>
      <span className="flex flex-col">
        <span className="text-xs uppercase tracking-wide text-stone-400">{label}</span>
        <span className="text-sm text-stone-700">{children}</span>
      </span>
    </li>
  );
}
