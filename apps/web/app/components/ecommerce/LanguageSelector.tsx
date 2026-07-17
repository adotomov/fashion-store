import { useEffect, useRef, useState } from "react";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { cn } from "../../lib/utils/cn";
import { Icon } from "../ui/Icon";

// Only renders the dropdown once the store has more than one enabled
// language — a single-language store has nothing to switch between.
export function LanguageSelector() {
  const { locale, languages, setLocale } = useLanguage();
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  // Close on an outside pointer press or Escape. We deliberately don't drive
  // this off the trigger's `onBlur`: on macOS Firefox and Safari a <button>
  // isn't focused when clicked (Chrome focuses it), so focus/blur-based
  // open/close logic misfires in those browsers.
  useEffect(() => {
    if (!open) return;

    function onPointerDown(e: PointerEvent) {
      if (!containerRef.current?.contains(e.target as Node)) setOpen(false);
    }
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "Escape") setOpen(false);
    }

    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  if (languages.length < 2) return null;

  const current = languages.find((l) => l.code === locale) ?? languages[0];

  return (
    <div ref={containerRef} className="relative">
      <button
        type="button"
        aria-label="Choose language"
        aria-haspopup="true"
        aria-expanded={open}
        className="flex items-center gap-1 rounded-sm p-2 text-sm font-medium text-stone-700 hover:bg-stone-50 sm:p-2.5"
        onClick={() => setOpen((o) => !o)}
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
                // preventDefault keeps the press from moving focus (and firing a
                // blur teardown) before the click lands — otherwise the option's
                // onClick can be lost mid-selection.
                onMouseDown={(e) => e.preventDefault()}
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
