import { useStoreBranding } from "../../features/store-settings/StoreSettingsContext";
import { type Money, formatMoney, isBulgarianLocale, toBGN } from "../../lib/money/money";
import { cn } from "../../lib/utils/cn";
import { Badge } from "./Badge";

type PriceProps = {
  price: Money;
  compareAtPrice?: Money;
  size?: "sm" | "md" | "lg";
  className?: string;
};

const sizes: Record<NonNullable<PriceProps["size"]>, string> = {
  sm: "text-sm",
  md: "text-base",
  lg: "text-xl",
};

export function Price({ price, compareAtPrice, size = "md", className }: PriceProps) {
  const { storeLocale } = useStoreBranding();
  const hasDiscount = compareAtPrice !== undefined && compareAtPrice.amount > price.amount;
  const percentOff = hasDiscount
    ? Math.round(((compareAtPrice!.amount - price.amount) / compareAtPrice!.amount) * 100)
    : 0;

  // Bulgarian law requires EUR prices to be shown alongside their BGN
  // equivalent at the fixed peg. Only relevant for EUR-denominated prices.
  const showBGN = isBulgarianLocale(storeLocale) && price.currency === "EUR";

  return (
    <span className={cn("inline-flex flex-col", className)}>
      <span className="inline-flex items-baseline gap-2">
        <span className={cn("font-medium text-stone-900", sizes[size])}>{formatMoney(price)}</span>
        {hasDiscount && (
          <>
            <span className="text-sm text-stone-400 line-through">{formatMoney(compareAtPrice!)}</span>
            <Badge variant="accent">-{percentOff}%</Badge>
          </>
        )}
      </span>
      {showBGN && <span className="text-sm text-stone-500">{formatMoney(toBGN(price), "bg-BG")}</span>}
    </span>
  );
}
