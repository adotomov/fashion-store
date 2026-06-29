import { type VariantProps, cva } from "class-variance-authority";
import type { ButtonHTMLAttributes } from "react";

import { cn } from "../../lib/utils/cn";

export const buttonStyles = cva(
  "inline-flex items-center justify-center gap-2 rounded-sm font-medium tracking-wide transition-colors disabled:cursor-not-allowed disabled:opacity-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-stone-900",
  {
    variants: {
      variant: {
        primary: "bg-stone-900 text-white hover:bg-stone-800",
        secondary: "bg-stone-200 text-stone-900 hover:bg-stone-300",
        outline: "border border-stone-300 bg-transparent text-stone-900 hover:bg-stone-50",
        ghost: "bg-transparent text-stone-900 hover:bg-stone-100",
        accent: "bg-clay-500 text-white hover:bg-clay-600",
        danger: "bg-danger-500 text-white hover:bg-danger-600",
      },
      size: {
        sm: "h-9 px-3 text-sm",
        md: "h-11 px-5 text-sm",
        lg: "h-14 px-7 text-base",
        icon: "h-10 w-10",
      },
    },
    defaultVariants: {
      variant: "primary",
      size: "md",
    },
  },
);

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & VariantProps<typeof buttonStyles>;

export function Button({ className, variant, size, ...props }: ButtonProps) {
  return <button className={cn(buttonStyles({ variant, size }), className)} {...props} />;
}
