import type { ReactNode } from "react";

import { cn } from "../../lib/utils/cn";

type AccordionProps = {
  open: boolean;
  children: ReactNode;
  className?: string;
};

// A simple controlled expand/collapse section — the caller owns the open
// state (usually tied to a Toggle) rather than this component managing it.
export function Accordion({ open, children, className }: AccordionProps) {
  if (!open) return null;
  return <div className={cn("border-t border-stone-200 px-6 py-5", className)}>{children}</div>;
}
