import type { HTMLAttributes } from "react";

import { cn } from "../../lib/utils/cn";

export function Card({ className, ...props }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn("rounded-sm border border-stone-200 bg-white", className)}
      {...props}
    />
  );
}
