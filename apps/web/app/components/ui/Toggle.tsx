import type { ButtonHTMLAttributes } from "react";

import { cn } from "../../lib/utils/cn";

type ToggleProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "onClick" | "onChange"> & {
  checked: boolean;
  onChange: (checked: boolean) => void;
};

export function Toggle({ checked, onChange, className, disabled, ...props }: ToggleProps) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative inline-flex h-6 w-11 shrink-0 items-center rounded-full transition-colors focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-stone-900 disabled:cursor-not-allowed disabled:opacity-50",
        checked ? "bg-stone-900" : "bg-stone-300",
        className,
      )}
      {...props}
    >
      <span
        className={cn(
          "inline-block h-5 w-5 transform rounded-full bg-white transition-transform",
          checked ? "translate-x-5" : "translate-x-0.5",
        )}
      />
    </button>
  );
}
