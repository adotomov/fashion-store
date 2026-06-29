import type { SelectHTMLAttributes } from "react";

import { cn } from "../../lib/utils/cn";
import { Icon } from "./Icon";

type SelectProps = SelectHTMLAttributes<HTMLSelectElement>;

export function Select({ className, children, ...props }: SelectProps) {
  return (
    <div className="relative">
      <select
        className={cn(
          "h-11 w-full appearance-none rounded-sm border border-stone-300 bg-white px-3.5 pr-9 text-sm text-stone-900 transition-colors focus:border-stone-900 focus:outline-none disabled:cursor-not-allowed disabled:bg-stone-50 disabled:text-stone-400",
          className,
        )}
        {...props}
      >
        {children}
      </select>
      <Icon
        name="chevronDown"
        size={16}
        className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-stone-500"
      />
    </div>
  );
}
