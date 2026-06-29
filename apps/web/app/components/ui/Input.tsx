import type { InputHTMLAttributes } from "react";

import { cn } from "../../lib/utils/cn";

type InputProps = InputHTMLAttributes<HTMLInputElement> & {
  invalid?: boolean;
};

export function Input({ className, invalid, ...props }: InputProps) {
  return (
    <input
      className={cn(
        "h-11 w-full rounded-sm border border-stone-300 bg-white px-3.5 text-sm text-stone-900 placeholder:text-stone-400 transition-colors focus:border-stone-900 focus:outline-none disabled:cursor-not-allowed disabled:bg-stone-50 disabled:text-stone-400",
        invalid && "border-danger-500 focus:border-danger-500",
        className,
      )}
      {...props}
    />
  );
}
