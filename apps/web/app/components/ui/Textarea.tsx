import type { TextareaHTMLAttributes } from "react";

import { cn } from "../../lib/utils/cn";

type TextareaProps = TextareaHTMLAttributes<HTMLTextAreaElement> & {
  invalid?: boolean;
};

export function Textarea({ className, invalid, ...props }: TextareaProps) {
  return (
    <textarea
      className={cn(
        "min-h-28 w-full rounded-sm border border-stone-300 bg-white px-3.5 py-2.5 text-sm text-stone-900 placeholder:text-stone-400 transition-colors focus:border-stone-900 focus:outline-none disabled:cursor-not-allowed disabled:bg-stone-50 disabled:text-stone-400",
        invalid && "border-danger-500 focus:border-danger-500",
        className,
      )}
      {...props}
    />
  );
}
