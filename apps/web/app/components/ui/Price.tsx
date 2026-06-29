import { type Money, formatMoney } from "../../lib/money/money";
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
  const hasDiscount = compareAtPrice !== undefined && compareAtPrice.amount > price.amount;
  const percentOff = hasDiscount
    ? Math.round(((compareAtPrice!.amount - price.amount) / compareAtPrice!.amount) * 100)
    : 0;

  return (
    <span className={cn("inline-flex items-baseline gap-2", className)}>
      <span className={cn("font-medium text-stone-900", sizes[size])}>{formatMoney(price)}</span>
      {hasDiscount && (
        <>
          <span className="text-sm text-stone-400 line-through">{formatMoney(compareAtPrice!)}</span>
          <Badge variant="accent">-{percentOff}%</Badge>
        </>
      )}
    </span>
  );
}
