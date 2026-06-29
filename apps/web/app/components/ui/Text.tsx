import type { ElementType, ReactNode } from "react";

import { cn } from "../../lib/utils/cn";

type HeadingProps = {
  as?: "h1" | "h2" | "h3" | "h4";
  size?: "xl" | "lg" | "md" | "sm";
  className?: string;
  children: ReactNode;
};

const headingSizes: Record<NonNullable<HeadingProps["size"]>, string> = {
  xl: "text-4xl md:text-5xl",
  lg: "text-3xl md:text-4xl",
  md: "text-2xl md:text-3xl",
  sm: "text-xl md:text-2xl",
};

// Display headings use the serif brand typeface; everything else stays on
// the sans body font for readability.
export function Heading({ as = "h2", size = "md", className, children }: HeadingProps) {
  const Component = as;
  return (
    <Component
      className={cn("font-display font-medium tracking-tight text-stone-900", headingSizes[size], className)}
    >
      {children}
    </Component>
  );
}

type TextProps = {
  as?: ElementType;
  size?: "lg" | "md" | "sm" | "xs";
  tone?: "default" | "muted" | "accent" | "danger";
  className?: string;
  children: ReactNode;
};

const textSizes: Record<NonNullable<TextProps["size"]>, string> = {
  lg: "text-lg",
  md: "text-base",
  sm: "text-sm",
  xs: "text-xs",
};

const textTones: Record<NonNullable<TextProps["tone"]>, string> = {
  default: "text-stone-900",
  muted: "text-stone-500",
  accent: "text-clay-600",
  danger: "text-danger-600",
};

export function Text({ as = "p", size = "md", tone = "default", className, children }: TextProps) {
  const Component = as;
  return <Component className={cn(textSizes[size], textTones[tone], className)}>{children}</Component>;
}

// Small uppercase label used for eyebrow text, section labels, and
// category tags throughout the storefront.
export function Eyebrow({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <span className={cn("text-xs font-medium uppercase tracking-[0.15em] text-stone-500", className)}>
      {children}
    </span>
  );
}
