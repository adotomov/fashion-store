import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useSearchParams, useNavigate } from "react-router";

import { Breadcrumbs } from "../components/ecommerce/Breadcrumbs";
import { type FilterGroup, FilterPanel } from "../components/ecommerce/FilterPanel";
import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { ProductCard } from "../components/ecommerce/ProductCard";
import { useLanguage } from "../features/i18n/LanguageContext";
import { Button } from "../components/ui/Button";
import { Icon } from "../components/ui/Icon";
import { Modal } from "../components/ui/Modal";
import { Heading, Text } from "../components/ui/Text";
import { useAuth } from "../features/auth/AuthContext";
import { useWishlist } from "../features/wishlist/WishlistContext";
import {
  type AttributeFacet,
  type NavType,
  type StorefrontProduct,
  getNav,
  listFacets,
  listStorefrontProductsPage,
  resolveImageUrl,
} from "../lib/api/storefront";
import { Pagination } from "../components/ui/Pagination";
import { loadShopFilterState, saveShopFilterState } from "../lib/shopFilterState";

export const handle = { title: "Shop" };

// Storefront grid page size — the API caps a page at 30 regardless.
const PAGE_SIZE = 30;

export default function Shop() {
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const navigate = useNavigate();
  const { locale, t } = useLanguage();
  const { isAuthenticated } = useAuth();
  const { isWishlisted, toggle } = useWishlist();

  // Links that should start a fresh filter context (header nav, home page
  // category/collection tiles) pass resetFilters via location state. Without
  // it — e.g. landing back here from a product detail page's "Back" button —
  // we restore whatever was last selected from sessionStorage, since none of
  // that ever lived in the URL to begin with.
  const shouldReset = Boolean((location.state as { resetFilters?: boolean } | null)?.resetFilters);
  const restored = shouldReset ? null : loadShopFilterState();

  const [navTypes, setNavTypes] = useState<NavType[]>([]);
  const [facets, setFacets] = useState<AttributeFacet[]>([]);
  const [products, setProducts] = useState<StorefrontProduct[] | null>(null);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [error, setError] = useState<string | null>(null);
  const [mobileFiltersOpen, setMobileFiltersOpen] = useState(false);

  const [selectedTypeSlugs, setSelectedTypeSlugs] = useState<string[]>(
    () => restored?.typeSlugs ?? searchParams.getAll("type"),
  );
  const [selectedCategoryIds, setSelectedCategoryIds] = useState<string[]>(
    () => restored?.categoryIds ?? searchParams.getAll("category_id"),
  );
  const [selectedCatalogId, setSelectedCatalogId] = useState<string | undefined>(
    () => restored?.catalogId ?? searchParams.get("catalog_id") ?? undefined,
  );
  const [selectedAttributeValueIds, setSelectedAttributeValueIds] = useState<string[]>(
    () => restored?.attributeValueIds ?? searchParams.getAll("attribute_value_id"),
  );
  const [onSaleOnly, setOnSaleOnly] = useState<boolean>(
    () => restored?.onSale ?? searchParams.get("sale") === "true",
  );
  // Landing scoped to a type but without explicit categories (home department
  // banners / header nav → /shop?type=slug) needs the type expanded into its
  // category ids: the product query filters by category, not type, so without
  // this it would show every product. The expansion is deferred until the nav
  // has loaded (each type's categories come from there), so we flag it here.
  const [typeExpansionPending, setTypeExpansionPending] = useState<boolean>(
    () =>
      !restored &&
      searchParams.getAll("type").length > 0 &&
      searchParams.getAll("category_id").length === 0,
  );
  // A search query lives entirely in the URL (never persisted to
  // sessionStorage like the other filters) — it always reflects exactly
  // what the header search form was last submitted with.
  const searchQuery = searchParams.get("q") ?? "";

  // Only re-sync from the URL when the navigation that brought us here was
  // explicitly marked as a reset (header nav, home page tiles) — otherwise
  // this would wipe the sessionStorage-restored state right after mount,
  // since a "Back to Products" navigation lands on a bare /shop with no
  // query params at all.
  const searchParamsKey = searchParams.toString();
  useEffect(() => {
    if (!shouldReset) return;
    const urlTypes = searchParams.getAll("type");
    const urlCategoryIds = searchParams.getAll("category_id");
    setSelectedTypeSlugs(urlTypes);
    setSelectedCategoryIds(urlCategoryIds);
    setSelectedCatalogId(searchParams.get("catalog_id") ?? undefined);
    setSelectedAttributeValueIds(searchParams.getAll("attribute_value_id"));
    setOnSaleOnly(searchParams.get("sale") === "true");
    setTypeExpansionPending(urlTypes.length > 0 && urlCategoryIds.length === 0);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParamsKey, shouldReset]);

  // Keep the persisted snapshot current so a later visit to a product page
  // and back restores exactly this state.
  useEffect(() => {
    saveShopFilterState({
      typeSlugs: selectedTypeSlugs,
      categoryIds: selectedCategoryIds,
      catalogId: selectedCatalogId,
      attributeValueIds: selectedAttributeValueIds,
      onSale: onSaleOnly,
    });
  }, [selectedTypeSlugs, selectedCategoryIds, selectedCatalogId, selectedAttributeValueIds, onSaleOnly]);

  // Stable string keys so effects don't re-fire every render over a fresh
  // array reference.
  const categoryKey = selectedCategoryIds.slice().sort().join(",");
  const attributeKey = selectedAttributeValueIds.slice().sort().join(",");
  // One string identifying the current filter set — any change resets to page 1.
  const filterKey = [categoryKey, selectedCatalogId ?? "", attributeKey, String(onSaleOnly), searchQuery, locale].join("|");
  // Tracks which filterKey the fetch effect last acted on, so it can skip the
  // now-stale request when a filter change is about to reset the page.
  const fetchedFilterKey = useRef(filterKey);
  // Monotonic request id: only the latest fetch's result is applied, guarding
  // against out-of-order responses when pages are changed quickly.
  const reqSeq = useRef(0);
  const didMount = useRef(false);

  useEffect(() => {
    getNav(locale)
      .then(setNavTypes)
      // If the nav can't load we can't expand a type into its categories —
      // drop the pending flag so the product list falls back to unfiltered
      // rather than hanging on the loading skeleton forever.
      .catch(() => setTypeExpansionPending(false));
  }, [locale]);

  // Once the nav has loaded, expand any type-only scope into that type's
  // category ids so the product query filters correctly (see the state flag).
  useEffect(() => {
    if (!typeExpansionPending || navTypes.length === 0) return;
    setSelectedCategoryIds((prev) => {
      const ids = new Set(prev);
      for (const slug of selectedTypeSlugs) {
        navTypes.find((nt) => nt.slug === slug)?.categories.forEach((c) => ids.add(c.id));
      }
      return Array.from(ids);
    });
    setTypeExpansionPending(false);
  }, [typeExpansionPending, navTypes, selectedTypeSlugs]);

  useEffect(() => {
    listFacets({ categoryIds: selectedCategoryIds, catalogId: selectedCatalogId, locale })
      .then(setFacets)
      .catch(() => setFacets([]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [categoryKey, selectedCatalogId, locale]);

  // Reset to the first page whenever the filter set changes (but not on the
  // initial mount, where page is already 1). Defined before the fetch effect
  // so its setPage runs first in the commit.
  useEffect(() => {
    if (!didMount.current) {
      didMount.current = true;
      return;
    }
    setPage(1);
  }, [filterKey]);

  useEffect(() => {
    setProducts(null);
    // Hold the fetch until a pending type→category expansion resolves, so we
    // never briefly query unscoped (which would flash every product).
    if (typeExpansionPending) return;
    // A filter just changed while off page 1: the reset effect above is about
    // to set page = 1, which re-runs this fetch. Skip the stale in-between one.
    if (fetchedFilterKey.current !== filterKey && page !== 1) {
      fetchedFilterKey.current = filterKey;
      return;
    }
    fetchedFilterKey.current = filterKey;

    const seq = ++reqSeq.current;
    listStorefrontProductsPage({
      categoryIds: selectedCategoryIds,
      catalogId: selectedCatalogId,
      attributeValueIds: selectedAttributeValueIds,
      hasPromotion: onSaleOnly || undefined,
      q: searchQuery || undefined,
      page,
      pageSize: PAGE_SIZE,
      locale,
    })
      .then((res) => {
        if (seq !== reqSeq.current) return;
        setProducts(res.items);
        setTotal(res.total);
      })
      .catch(() => {
        if (seq === reqSeq.current) setError(t("shop.load_error", "Could not load products."));
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterKey, page, typeExpansionPending]);

  // Only types that own at least one selected category stay "active" — lets
  // us drive both the Type and Category groups from the same source of truth.
  const categoriesByType = useMemo(() => {
    const map = new Map<string, NavType["categories"]>();
    for (const type of navTypes) map.set(type.slug, type.categories);
    return map;
  }, [navTypes]);

  const visibleCategories = useMemo(() => {
    if (selectedTypeSlugs.length === 0) {
      return navTypes.flatMap((nt) => nt.categories);
    }
    return selectedTypeSlugs.flatMap((slug) => categoriesByType.get(slug) ?? []);
  }, [navTypes, selectedTypeSlugs, categoriesByType]);

  function toggleInList(list: string[], value: string): string[] {
    return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
  }

  function handleToggle(groupId: string, optionId: string) {
    if (groupId === "type") {
      const type = navTypes.find((nt) => nt.slug === optionId);
      const isCurrentlySelected = selectedTypeSlugs.includes(optionId);
      setSelectedTypeSlugs((prev) => toggleInList(prev, optionId));
      // Keep the category group in sync: adding a type pre-selects its
      // categories, removing it drops them, matching how arriving from
      // the nav (type click) scopes the page.
      if (type) {
        const categoryIds = type.categories.map((c) => c.id);
        setSelectedCategoryIds((prev) =>
          isCurrentlySelected
            ? prev.filter((id) => !categoryIds.includes(id))
            : Array.from(new Set([...prev, ...categoryIds])),
        );
      }
    } else if (groupId === "category") {
      setSelectedCategoryIds((prev) => toggleInList(prev, optionId));
    } else if (groupId === "offers") {
      setOnSaleOnly((prev) => !prev);
    } else {
      setSelectedAttributeValueIds((prev) => toggleInList(prev, optionId));
    }
  }

  function handleClear() {
    setSelectedTypeSlugs([]);
    setSelectedCategoryIds([]);
    setSelectedCatalogId(undefined);
    setSelectedAttributeValueIds([]);
    setOnSaleOnly(false);
  }

  const filterGroups: FilterGroup[] = [
    {
      id: "offers",
      label: t("shop.filter_offers", "Offers"),
      type: "checkbox",
      options: [{ id: "on_sale", label: t("shop.filter_on_sale", "On Sale") }],
    },
    {
      id: "type",
      label: t("shop.filter_type", "Type"),
      type: "checkbox",
      options: navTypes.map((nt) => ({ id: nt.slug, label: nt.name })),
    },
    {
      id: "category",
      label: t("shop.filter_category", "Category"),
      type: "checkbox",
      options: visibleCategories.map((c) => ({ id: c.id, label: c.name })),
    },
    ...facets.map((facet): FilterGroup =>
      facet.attribute_type === "color"
        ? {
            id: facet.attribute_id,
            label: facet.attribute_name,
            type: "color",
            options: facet.values.map((v) => ({ id: v.id, name: v.value, hex: v.color_hex ?? "#e7e5e4" })),
          }
        : {
            id: facet.attribute_id,
            label: facet.attribute_name,
            type: "checkbox",
            options: facet.values.map((v) => ({ id: v.id, label: v.value })),
          },
    ),
  ];

  const selected: Record<string, string[]> = {
    offers: onSaleOnly ? ["on_sale"] : [],
    type: selectedTypeSlugs,
    category: selectedCategoryIds,
  };
  for (const facet of facets) {
    selected[facet.attribute_id] = selectedAttributeValueIds.filter((id) =>
      facet.values.some((v) => v.id === id),
    );
  }

  const heading = searchQuery
    ? `${t("shop.search_results_for", "Search results for")} "${searchQuery}"`
    : selectedCategoryIds.length === 1
      ? visibleCategories.find((c) => c.id === selectedCategoryIds[0])?.name ?? t("nav.shop_all", "Shop All")
      : selectedTypeSlugs.length === 1
        ? navTypes.find((nt) => nt.slug === selectedTypeSlugs[0])?.name ?? t("nav.shop_all", "Shop All")
        : t("shop.all_products", "All Products");

  function clearSearch() {
    const next = new URLSearchParams(searchParams);
    next.delete("q");
    navigate(`/shop${next.toString() ? `?${next.toString()}` : ""}`);
  }

  const breadcrumbItems = [
    { label: "Home", href: "/" },
    ...(selectedTypeSlugs.length === 1
      ? [{ label: navTypes.find((nt) => nt.slug === selectedTypeSlugs[0])?.name ?? selectedTypeSlugs[0], href: `/shop?type=${selectedTypeSlugs[0]}` }]
      : []),
    ...(selectedCategoryIds.length === 1 ? [{ label: heading }] : selectedTypeSlugs.length !== 1 ? [{ label: t("nav.shop_all", "Shop All") }] : []),
  ];

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
          <Breadcrumbs items={breadcrumbItems} />
          <Heading as="h1" size="lg" className="mt-3">
            {heading}
          </Heading>

          {searchQuery && (
            <button
              type="button"
              onClick={clearSearch}
              className="mt-3 inline-flex items-center gap-1.5 rounded-full bg-stone-100 px-3 py-1 text-xs font-medium text-stone-700 hover:bg-stone-200"
            >
              Search: "{searchQuery}"
              <Icon name="close" size={12} />
            </button>
          )}

          {error && (
            <Text size="sm" tone="danger" className="mt-4">
              {error}
            </Text>
          )}

          <div className="mt-8 grid grid-cols-1 gap-10 lg:grid-cols-[260px_1fr]">
            <div className="hidden lg:block">
              <FilterPanel groups={filterGroups} selected={selected} onToggle={handleToggle} onClear={handleClear} />
            </div>

            <div>
              <div className="mb-4 flex items-center justify-between lg:justify-end">
                <Button variant="outline" size="sm" className="lg:hidden" onClick={() => setMobileFiltersOpen(true)}>
                  <Icon name="filters" size={16} />
                  {t("shop.filters", "Filters")}
                </Button>
                <Text size="sm" tone="muted">
                  {products ? `${total} item${total === 1 ? "" : "s"}` : t("common.loading", "Loading…")}
                </Text>
              </div>

              {products === null ? (
                <div className="grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-3">
                  {Array.from({ length: 6 }).map((_, i) => (
                    <div key={i} className="aspect-[3/4] animate-pulse rounded-sm bg-stone-100" />
                  ))}
                </div>
              ) : products.length === 0 ? (
                <Text size="sm" tone="muted" className="py-16 text-center">
                  {t("shop.no_products", "No products match these filters yet.")}
                </Text>
              ) : (
                <div className="grid grid-cols-2 gap-x-6 gap-y-10 sm:grid-cols-3">
                  {products.map((product) => (
                    <ProductCard
                      key={product.id}
                      href={`/shop/${product.slug}`}
                      image={product.image_url ? { src: resolveImageUrl(product.image_url), alt: product.name } : undefined}
                      title={product.name}
                      price={product.base_price}
                      compareAtPrice={product.compare_at_price}
                      promotionPrice={product.promotion_price}
                      promotionLabel={product.promotion_label}
                      outOfStock={!product.in_stock}
                      isWishlisted={isAuthenticated && isWishlisted(product.id)}
                      onToggleWishlist={isAuthenticated ? () => toggle(product.id) : undefined}
                    />
                  ))}
                </div>
              )}

              {products !== null && products.length > 0 && (
                <Pagination
                  page={page}
                  totalPages={Math.ceil(total / PAGE_SIZE)}
                  onPageChange={(p) => {
                    setPage(p);
                    window.scrollTo({ top: 0, behavior: "smooth" });
                  }}
                  className="mt-12"
                />
              )}
            </div>
          </div>
        </div>
      </main>
      <Footer />

      <Modal open={mobileFiltersOpen} onClose={() => setMobileFiltersOpen(false)} title={t("shop.filters", "Filters")}>
        <FilterPanel groups={filterGroups} selected={selected} onToggle={handleToggle} onClear={handleClear} />
      </Modal>
    </div>
  );
}
