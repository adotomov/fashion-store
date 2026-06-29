import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { type Language, getUiStrings, listEnabledLanguages } from "../../lib/api/storefront";

const STORAGE_KEY = "store_locale";
const DEFAULT_LOCALE = "en";

type LanguageState = {
  locale: string;
  languages: Language[];
  setLocale: (code: string) => void;
  // Looks up a static UI string by key, falling back to the caller-supplied
  // default text — covers both keys not yet seeded server-side and locales
  // that haven't translated that key yet.
  t: (key: string, fallback: string) => string;
};

const LanguageContext = createContext<LanguageState>({
  locale: DEFAULT_LOCALE,
  languages: [],
  setLocale: () => {},
  t: (_key, fallback) => fallback,
});

function readStoredLocale(): string {
  if (typeof window === "undefined") return DEFAULT_LOCALE;
  return window.localStorage.getItem(STORAGE_KEY) ?? DEFAULT_LOCALE;
}

export function LanguageProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState(DEFAULT_LOCALE);
  const [languages, setLanguages] = useState<Language[]>([]);
  const [strings, setStrings] = useState<Record<string, string>>({});

  useEffect(() => {
    setLocaleState(readStoredLocale());
    listEnabledLanguages()
      .then(setLanguages)
      .catch(() => {});
  }, []);

  useEffect(() => {
    getUiStrings(locale)
      .then(setStrings)
      .catch(() => {});
  }, [locale]);

  function setLocale(code: string) {
    setLocaleState(code);
    if (typeof window !== "undefined") {
      window.localStorage.setItem(STORAGE_KEY, code);
    }
  }

  function t(key: string, fallback: string): string {
    return strings[key] ?? fallback;
  }

  return <LanguageContext.Provider value={{ locale, languages, setLocale, t }}>{children}</LanguageContext.Provider>;
}

export function useLanguage(): LanguageState {
  return useContext(LanguageContext);
}
