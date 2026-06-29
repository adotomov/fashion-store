import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { getStoreSettings, resolveImageUrl } from "../../lib/api/storefront";

const DEFAULT_STORE_NAME = "MAISON";

type StoreBranding = {
  storeName: string;
  logoUrl: string | null;
  // Re-fetches the public store-settings endpoint — called after the admin
  // Store Settings page saves a change, so the rest of the already-loaded
  // app (header, footer, admin sidebar) picks it up without a full reload.
  refresh: () => void;
};

const StoreSettingsContext = createContext<StoreBranding>({
  storeName: DEFAULT_STORE_NAME,
  logoUrl: null,
  refresh: () => {},
});

export function StoreSettingsProvider({ children }: { children: ReactNode }) {
  const [storeName, setStoreName] = useState(DEFAULT_STORE_NAME);
  const [logoUrl, setLogoUrl] = useState<string | null>(null);

  function refresh() {
    getStoreSettings()
      .then((settings) => {
        setStoreName(settings.store_name || DEFAULT_STORE_NAME);
        setLogoUrl(settings.logo_url ? resolveImageUrl(settings.logo_url) : null);
      })
      .catch(() => {});
  }

  useEffect(() => {
    refresh();
  }, []);

  return (
    <StoreSettingsContext.Provider value={{ storeName, logoUrl, refresh }}>{children}</StoreSettingsContext.Provider>
  );
}

export function useStoreBranding(): StoreBranding {
  return useContext(StoreSettingsContext);
}
