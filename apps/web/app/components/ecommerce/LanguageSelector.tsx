import { useState } from "react";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { cn } from "../../lib/utils/cn";
import { Icon } from "../ui/Icon";

// Only renders the dropdown once the store has more than one enabled
// language — a single-language store has nothing to switch between.
export function LanguageSelector() {
  const { locale, languages, setLocale } = useLanguage();
  const [open, setOpen] = useState(false);

  if (languages.length < 2) return null;

  const current = languages.find((l) => l.code === locale) ?? languages[0];

  return (
    <div className="relative">
      <button
        type="button"
        aria-label="Choose language"
        className="flex items-center gap-1 rounded-sm p-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50"
        onClick={() => setOpen((o) => !o)}
        onBlur={() => setOpen(false)}
      >
        {current.code.toUpperCase()}
        <Icon name="chevronDown" size={14} className={cn("transition-transform", open && "rotate-180")} />
      </button>
      {open && (
        <ul className="absolute right-0 top-full z-40 mt-1 min-w-[8rem] rounded-sm border border-stone-200 bg-white py-1 shadow-lg">
          {languages.map((lang) => (
            <li key={lang.code}>
              <button
                type="button"
                className={cn(
                  "block w-full px-4 py-2 text-left text-sm hover:bg-stone-50",
                  lang.code === locale ? "font-medium text-stone-900" : "text-stone-600",
                )}
                onClick={() => {
                  setLocale(lang.code);
                  setOpen(false);
                }}
              >
                {lang.name}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
