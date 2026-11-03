import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { getStoreSettings, resolveImageUrl } from "../../lib/api/storefront";

const DEFAULT_STORE_NAME = "Fashion Store";

type StoreBranding = {
  storeName: string;
  logoUrl: string | null;
  // The store's configured BCP-47 locale (e.g. "bg-BG"), set under Settings →
  // Identity. Drives store-wide, locale-dependent formatting.
  storeLocale: string;
  // Social profile URLs shown in the footer — empty string means "not set",
  // in which case the footer hides that icon.
  facebookUrl: string;
  instagramUrl: string;
  // Re-fetches the public store-settings endpoint — called after the admin
  // Store Settings page saves a change, so the rest of the already-loaded
  // app (header, footer, admin sidebar) picks it up without a full reload.
  // The free-shipping threshold in EUR minor units; null until the store settings have loaded.
  freeShippingThresholdMinor: number | null;
  refresh: () => void;
};

const StoreSettingsContext = createContext<StoreBranding>({
  storeName: DEFAULT_STORE_NAME,
  logoUrl: null,
  storeLocale: "",
  facebookUrl: "",
  instagramUrl: "",
  freeShippingThresholdMinor: null,
  refresh: () => {},
});

export function StoreSettingsProvider({ children }: { children: ReactNode }) {
  const [storeName, setStoreName] = useState(DEFAULT_STORE_NAME);
  const [logoUrl, setLogoUrl] = useState<string | null>(null);
  const [storeLocale, setStoreLocale] = useState("");
  const [facebookUrl, setFacebookUrl] = useState("");
  const [instagramUrl, setInstagramUrl] = useState("");
  const [freeShippingThresholdMinor, setFreeShippingThresholdMinor] = useState<number | null>(null);

  function refresh() {
    getStoreSettings()
      .then((settings) => {
        setStoreName(settings.store_name || DEFAULT_STORE_NAME);
        setLogoUrl(settings.logo_url ? resolveImageUrl(settings.logo_url) : null);
        setStoreLocale(settings.locale ?? "");
        setFacebookUrl(settings.facebook_url ?? "");
        setInstagramUrl(settings.instagram_url ?? "");
        setFreeShippingThresholdMinor(settings.free_shipping_threshold_minor ?? null);
      })
      .catch(() => {});
  }

  useEffect(() => {
    refresh();
  }, []);

  // Point the browser favicon at the store's uploaded logo once branding
  // loads. Falls back to the static /favicon.ico when no logo is configured.
  useEffect(() => {
    if (typeof document === "undefined") return;
    let link = document.querySelector<HTMLLinkElement>("link[rel~='icon']");
    if (!link) {
      link = document.createElement("link");
      link.rel = "icon";
      document.head.appendChild(link);
    }
    link.href = logoUrl ?? "/favicon.ico";
  }, [logoUrl]);

  return (
    <StoreSettingsContext.Provider value={{ storeName, logoUrl, storeLocale, facebookUrl, instagramUrl, freeShippingThresholdMinor, refresh }}>
      {children}
    </StoreSettingsContext.Provider>
  );
}

export function useStoreBranding(): StoreBranding {
  return useContext(StoreSettingsContext);
}
