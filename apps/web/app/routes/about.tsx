import { useEffect, useState } from "react";

import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { Heading, Text } from "../components/ui/Text";
import { type StorefrontStoreSettings, getStoreSettings, resolveImageUrl } from "../lib/api/storefront";

export const handle = { title: "About" };

export default function About() {
  const [settings, setSettings] = useState<StorefrontStoreSettings | null>(null);

  useEffect(() => {
    getStoreSettings()
      .then(setSettings)
      .catch(() => setSettings(null));
  }, []);

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
          {settings?.logo_url && (
            <img
              src={resolveImageUrl(settings.logo_url)}
              alt={settings.store_name}
              className="mb-8 h-12 w-auto object-contain"
            />
          )}
          <Heading as="h1" size="lg">
            About {settings?.store_name ?? "Us"}
          </Heading>
          <Text className="mt-6 whitespace-pre-line leading-relaxed" tone="muted">
            {settings?.company_description ?? "More information about us is coming soon."}
          </Text>
        </div>
      </main>
      <Footer />
    </div>
  );
}
