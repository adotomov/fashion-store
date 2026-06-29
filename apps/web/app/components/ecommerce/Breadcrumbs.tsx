import { Link } from "react-router";

import { cn } from "../../lib/utils/cn";
import { Icon } from "../ui/Icon";

export type Breadcrumb = {
  label: string;
  href?: string;
};

type BreadcrumbsProps = {
  items: Breadcrumb[];
  className?: string;
};

export function Breadcrumbs({ items, className }: BreadcrumbsProps) {
  return (
    <nav aria-label="Breadcrumb" className={cn("flex items-center gap-1.5 text-sm", className)}>
      {items.map((item, index) => {
        const isLast = index === items.length - 1;
        return (
          <span key={item.label} className="flex items-center gap-1.5">
            {item.href && !isLast ? (
              <Link to={item.href} className="text-stone-500 hover:text-stone-900">
                {item.label}
              </Link>
            ) : (
              <span className={isLast ? "text-stone-900" : "text-stone-500"} aria-current={isLast ? "page" : undefined}>
                {item.label}
              </span>
            )}
            {!isLast && <Icon name="chevronRight" size={14} className="text-stone-400" />}
          </span>
        );
      })}
    </nav>
  );
}
