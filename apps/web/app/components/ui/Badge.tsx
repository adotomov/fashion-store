import { type VariantProps, cva } from "class-variance-authority";
import type { HTMLAttributes } from "react";

import { cn } from "../../lib/utils/cn";

const badgeStyles = cva("inline-flex items-center rounded-full px-2.5 py-1 text-xs font-medium", {
  variants: {
    variant: {
      neutral: "bg-stone-100 text-stone-700",
      brand: "bg-stone-300 text-stone-900",
      accent: "bg-clay-50 text-clay-600",
      success: "bg-sage-50 text-sage-600",
      danger: "bg-danger-50 text-danger-600",
    },
  },
  defaultVariants: {
    variant: "neutral",
  },
});

type BadgeProps = HTMLAttributes<HTMLSpanElement> & VariantProps<typeof badgeStyles>;

export function Badge({ className, variant, ...props }: BadgeProps) {
  return <span className={cn(badgeStyles({ variant }), className)} {...props} />;
}
