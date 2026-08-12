import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { type Language, getResolvedLocale, getUiStrings, listEnabledLanguages } from "../../lib/api/storefront";

const STORAGE_KEY = "store_locale";
const DEFAULT_LOCALE = "en";

type LanguageState = {
  locale: string;
  languages: Language[];
  setLocale: (code: string) => void;
  // applyAccountLocale sets the language from a signed-in user's account
  // preference. Unlike setLocale it does NOT persist — an explicit manual choice
  // still wins — but it overrides geo/browser detection (account wins over geo).
  applyAccountLocale: (code: string) => void;
  // Looks up a static UI string by key, falling back to the caller-supplied
  // default text — covers both keys not yet seeded server-side and locales
  // that haven't translated that key yet.
  t: (key: string, fallback: string) => string;
};

const LanguageContext = createContext<LanguageState>({
  locale: DEFAULT_LOCALE,
  languages: [],
  setLocale: () => {},
  applyAccountLocale: () => {},
  t: (_key, fallback) => fallback,
});

// The visitor's explicit, remembered choice — null when they've never picked one,
// in which case we auto-detect from location/browser via the API.
function readStoredLocale(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(STORAGE_KEY);
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState(DEFAULT_LOCALE);
  const [languages, setLanguages] = useState<Language[]>([]);
  const [strings, setStrings] = useState<Record<string, string>>({});

  useEffect(() => {
    listEnabledLanguages()
      .then(setLanguages)
      .catch(() => {});

    // Precedence: an explicit remembered choice wins; otherwise the server
    // resolves the locale from geo + browser, falling back to the store default.
    const stored = readStoredLocale();
    if (stored) {
      setLocaleState(stored);
      return;
    }
    getResolvedLocale()
      .then(({ locale: resolved }) => setLocaleState(resolved || DEFAULT_LOCALE))
      .catch(() => setLocaleState(DEFAULT_LOCALE));
  }, []);

  useEffect(() => {
    getUiStrings(locale)
      .then(setStrings)
      .catch(() => {});
    if (typeof document !== "undefined") {
      document.documentElement.lang = locale;
    }
  }, [locale]);

  function setLocale(code: string) {
    setLocaleState(code);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, code);
    }
  }

  function applyAccountLocale(code: string) {
    if (!code) return;
    // Respect an explicit manual choice; otherwise the account language wins
    // over geo/browser detection.
    if (typeof window !== "undefined" && window.localStorage.getItem(STORAGE_KEY)) return;
    setLocaleState(code);
  }

  function t(key: string, fallback: string): string {
    return strings[key] ?? fallback;
  }

  return (
    <LanguageContext.Provider value={{ locale, languages, setLocale, applyAccountLocale, t }}>
      {children}
    </LanguageContext.Provider>
  );
}

export function useLanguage(): LanguageState {
  return useContext(LanguageContext);
}
