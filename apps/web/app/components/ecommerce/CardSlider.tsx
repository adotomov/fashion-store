import { type ReactNode, useEffect, useState } from "react";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { Heading } from "../ui/Text";
import { Icon } from "../ui/Icon";

// How many cards are visible at each breakpoint. Tracked in state (not pure
// CSS) because the circular window has to know how many items it shows to
// compute which slice is on screen.
function visibleCountFor(width: number): number {
  if (width >= 1024) return 4;
  if (width >= 640) return 3;
  return 2;
}

export function useVisibleCount(): number {
  const [count, setCount] = useState(4);

  useEffect(() => {
    function update() {
      setCount(visibleCountFor(window.innerWidth));
    }
    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, []);

  return count;
}

type CardSliderProps<T> = {
  // Optional: when omitted the strip is headless (used by the department
  // banners) and the arrows sit alone at the top-right.
  title?: string;
  items: T[];
  getKey: (item: T) => string;
  renderItem: (item: T) => ReactNode;
};

// A horizontal strip of cards that shows a fixed number at a time and loops
// endlessly left/right — shared by "Shop by Category" (category tiles) and
// "Best in its category" (product cards).
export function CardSlider<T>({ title, items, getKey, renderItem }: CardSliderProps<T>) {
  const { t } = useLanguage();
  const visibleCount = useVisibleCount();
  const [offset, setOffset] = useState(0);
  const total = items.length;
  const canSlide = total > visibleCount;

  // Circular window: wrap around the end so the strip never runs out.
  const visible = Array.from(
    { length: Math.min(visibleCount, total) },
    (_, i) => items[(offset + i) % total],
  );

  function slide(delta: number) {
    setOffset((current) => (current + delta + total) % total);
  }

  return (
    <div>
      {(title || canSlide) && (
        <div className="flex items-center justify-between gap-4">
          {title ? <Heading as="h3" size="sm">{title}</Heading> : <span />}
          {canSlide && (
            <div className="flex items-center gap-1.5">
              <SlideButton
                direction="left"
                label={t("home.slider_prev", "Previous")}
                onClick={() => slide(-1)}
              />
              <SlideButton
                direction="right"
                label={t("home.slider_next", "Next")}
                onClick={() => slide(1)}
              />
            </div>
          )}
        </div>
      )}

      {/* Always lay out `visibleCount` columns so a group with fewer items
          leaves empty cells rather than stretching its cards across the row. */}
      <div className={`${title || canSlide ? "mt-4" : ""} grid gap-4 sm:gap-6`} style={{ gridTemplateColumns: `repeat(${visibleCount}, minmax(0, 1fr))` }}>
        {visible.map((item) => (
          <div key={getKey(item)}>{renderItem(item)}</div>
        ))}
      </div>
    </div>
  );
}

function SlideButton({
  direction,
  label,
  onClick,
}: {
  direction: "left" | "right";
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      onClick={onClick}
      className="flex h-9 w-9 items-center justify-center rounded-full border border-stone-300 text-stone-700 transition-colors hover:border-stone-900 hover:text-stone-900"
    >
      <Icon name={direction === "left" ? "chevronLeft" : "chevronRight"} size={18} />
    </button>
  );
}
