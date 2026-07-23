import { useMemo, useState } from "react";

export type PaginationState<T> = {
  page: number;
  totalPages: number;
  pageItems: T[];
  setPage: (page: number) => void;
};

// Client-side pagination for a fully-loaded list: slices `items` into pages of
// `pageSize` and tracks the current page. The page is clamped to the available
// range, so deleting the last item on the final page falls back gracefully.
// Callers with filters should reset to page 1 when the filter changes.
export function usePagination<T>(items: T[], pageSize: number): PaginationState<T> {
  const [page, setPage] = useState(1);

  const totalPages = Math.max(1, Math.ceil(items.length / pageSize));
  const currentPage = Math.min(Math.max(1, page), totalPages);

  const pageItems = useMemo(
    () => items.slice((currentPage - 1) * pageSize, currentPage * pageSize),
    [items, currentPage, pageSize],
  );

  return { page: currentPage, totalPages, pageItems, setPage };
}
