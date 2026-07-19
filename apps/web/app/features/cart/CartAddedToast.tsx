import { useEffect, useState } from "react";
import { Link } from "react-router";

import { buttonStyles } from "../../components/ui/Button";
import { Icon } from "../../components/ui/Icon";
import { Text } from "../../components/ui/Text";
import { cn } from "../../lib/utils/cn";
import { useLanguage } from "../i18n/LanguageContext";

const VISIBLE_MS = 5000;
const EXIT_MS = 300;

/**
 * Transient confirmation that slides up from the bottom of the screen whenever
 * an item is added to the cart, with a shortcut into the cart. Auto-dismisses
 * after 5s. Driven by a monotonically increasing `nonce` from CartContext so
 * repeated adds re-trigger (and restart the timer) even while one is on screen.
 */
export function CartAddedToast({ nonce }: { nonce: number }) {
  const { t } = useLanguage();
  // `mounted` keeps the node in the DOM through the exit transition; `shown`
  // toggles the slide/opacity so the same element can animate in and out.
  const [mounted, setMounted] = useState(false);
  const [shown, setShown] = useState(false);

  useEffect(() => {
    if (nonce === 0) return; // no add yet on this mount
    setMounted(true);
    const raf = requestAnimationFrame(() => setShown(true));
    const hideTimer = setTimeout(() => setShown(false), VISIBLE_MS);
    const unmountTimer = setTimeout(() => setMounted(false), VISIBLE_MS + EXIT_MS);
    return () => {
      cancelAnimationFrame(raf);
      clearTimeout(hideTimer);
      clearTimeout(unmountTimer);
    };
  }, [nonce]);

  if (!mounted) return null;

  return (
    <div
      role="status"
      aria-live="polite"
      className="pointer-events-none fixed inset-x-0 bottom-0 z-50 flex justify-center px-4 pb-[max(1rem,env(safe-area-inset-bottom))]"
    >
      <div
        className={cn(
          "pointer-events-auto flex w-full max-w-sm items-center gap-3 rounded-lg border border-stone-200 bg-white px-4 py-3 shadow-lg transition-all duration-300 ease-out",
          shown ? "translate-y-0 opacity-100" : "translate-y-6 opacity-0",
        )}
      >
        <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-emerald-50 text-emerald-600">
          <Icon name="check" size={18} />
        </span>
        <Text size="sm" className="min-w-0 flex-1 font-medium">
          {t("cart.added_toast", "Added to your cart")}
        </Text>
        <Link
          to="/cart"
          onClick={() => setShown(false)}
          className={cn(buttonStyles({ variant: "primary", size: "sm" }), "shrink-0 whitespace-nowrap")}
        >
          {t("cart.open_cart", "Open Cart")}
        </Link>
      </div>
    </div>
  );
}
