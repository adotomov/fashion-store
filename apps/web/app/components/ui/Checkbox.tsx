import type { InputHTMLAttributes } from "react";

import { cn } from "../../lib/utils/cn";

type CheckboxProps = InputHTMLAttributes<HTMLInputElement> & {
  label?: string;
};

export function Checkbox({ className, label, id, ...props }: CheckboxProps) {
  const input = (
    <input
      type="checkbox"
      id={id}
      className={cn(
        "h-4 w-4 rounded-sm border border-stone-400 text-stone-900 accent-stone-900 focus:outline-none focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-stone-900",
        className,
      )}
      {...props}
    />
  );

  if (!label) return input;

  return (
    <label htmlFor={id} className="flex items-center gap-2 text-sm text-stone-700">
      {input}
      {label}
    </label>
  );
}
