import { Button } from "./Button";
import { Icon } from "./Icon";
import { Text } from "./Text";

type PaginationProps = {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  className?: string;
};

// Build a compact windowed page list around the current page, inserting
// "ellipsis" markers where pages are skipped, e.g. 1 … 4 5 [6] 7 8 … 20.
function pageWindow(page: number, totalPages: number): (number | "ellipsis")[] {
  const MAX = 7; // first, last, current, 2 neighbours each side, 2 ellipses
  if (totalPages <= MAX) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }
  const pages: (number | "ellipsis")[] = [1];
  const start = Math.max(2, page - 1);
  const end = Math.min(totalPages - 1, page + 1);
  if (start > 2) pages.push("ellipsis");
  for (let p = start; p <= end; p++) pages.push(p);
  if (end < totalPages - 1) pages.push("ellipsis");
  pages.push(totalPages);
  return pages;
}

// Numbered pager with previous/next arrows. Presentational — the caller owns
// the page state (client-side slicing or a server refetch). Renders nothing
// for a single page.
export function Pagination({ page, totalPages, onPageChange, className }: PaginationProps) {
  if (totalPages <= 1) return null;

  return (
    <nav
      className={`flex items-center justify-center gap-1.5 ${className ?? ""}`}
      aria-label="Pagination"
    >
      <Button
        variant="ghost"
        size="sm"
        aria-label="Previous page"
        disabled={page <= 1}
        onClick={() => onPageChange(page - 1)}
      >
        <Icon name="chevronLeft" size={16} />
      </Button>

      {pageWindow(page, totalPages).map((entry, i) =>
        entry === "ellipsis" ? (
          <Text key={`e${i}`} size="sm" tone="muted" className="px-1 select-none">
            …
          </Text>
        ) : (
          <Button
            key={entry}
            variant={entry === page ? "primary" : "ghost"}
            size="sm"
            aria-label={`Page ${entry}`}
            aria-current={entry === page ? "page" : undefined}
            onClick={() => onPageChange(entry)}
          >
            {entry}
          </Button>
        ),
      )}

      <Button
        variant="ghost"
        size="sm"
        aria-label="Next page"
        disabled={page >= totalPages}
        onClick={() => onPageChange(page + 1)}
      >
        <Icon name="chevronRight" size={16} />
      </Button>
    </nav>
  );
}
