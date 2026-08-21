import { useEffect, useState } from "react";

import { Link } from "react-router";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { buttonStyles } from "../components/ui/Button";
import { Icon } from "../components/ui/Icon";
import { Text } from "../components/ui/Text";
import { useLanguage } from "../features/i18n/LanguageContext";
import { useStoreBranding } from "../features/store-settings/StoreSettingsContext";
import { getStorefrontLegalContent } from "../lib/api/store-documents";
import { type StorefrontStoreSettings, getStoreSettings, resolveImageUrl } from "../lib/api/storefront";
import { buildMeta } from "../lib/seo/meta";

export function meta() {
  return buildMeta({
    title: "About Us",
    description: "Learn about Verani — our story, our values, and the curated fashion we bring to Bulgaria.",
    path: "/about",
  });
}

export default function About() {
  const { locale, t } = useLanguage();
  const { storeName } = useStoreBranding();
  const [settings, setSettings] = useState<StorefrontStoreSettings | null>(null);
  const [content, setContent] = useState<string | null>(null);

  useEffect(() => {
    getStoreSettings()
      .then(setSettings)
      .catch(() => setSettings(null));
  }, []);

  useEffect(() => {
    let cancelled = false;
    setContent(null);
    getStorefrontLegalContent("about", locale)
      .then((r) => {
        if (!cancelled) setContent(r.content_md);
      })
      .catch(() => {
        if (!cancelled) setContent("");
      });
    return () => {
      cancelled = true;
    };
  }, [locale]);

  const coverUrl = settings?.about_cover_url ? resolveImageUrl(settings.about_cover_url) : null;

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        {/* Cover photo */}
        <div className="relative h-64 w-full overflow-hidden bg-stone-100 sm:h-80 lg:h-[28rem]">
          {coverUrl ? (
            <img src={coverUrl} alt={storeName} className="h-full w-full object-cover" />
          ) : (
            <div className="h-full w-full bg-gradient-to-br from-stone-200 via-stone-100 to-clay-100" />
          )}
        </div>

        <div className="mx-auto max-w-3xl px-4 py-14 sm:px-6 lg:px-8">
          {content === null ? (
            <Text tone="muted">{t("common.loading", "Loading…")}</Text>
          ) : content.trim() ? (
            <div className="prose prose-stone max-w-none">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
            </div>
          ) : (
            <Text className="whitespace-pre-line leading-relaxed" tone="muted">
              {t("about.unavailable", "Our story is coming soon.")}
            </Text>
          )}

          <div className="mt-12 flex justify-center border-t border-stone-200 pt-10">
            <Link to="/help/contact" className={buttonStyles({ variant: "primary", size: "lg" })}>
              {t("about.contact_cta", "Contact Us")}
              <Icon name="chevronRight" size={18} />
            </Link>
          </div>
        </div>
      </main>
      <Footer />
    </div>
  );
}
