import { useEffect, useMemo, useState } from "react";
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
  listStorefrontProducts,
  resolveImageUrl,
} from "../lib/api/storefront";
import { loadShopFilterState, saveShopFilterState } from "../lib/shopFilterState";

export const handle = { title: "Shop" };

export default function Shop() {
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const navigate = useNavigate();
  const { locale } = useLanguage();
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
    setSelectedTypeSlugs(searchParams.getAll("type"));
    setSelectedCategoryIds(searchParams.getAll("category_id"));
    setSelectedCatalogId(searchParams.get("catalog_id") ?? undefined);
    setSelectedAttributeValueIds(searchParams.getAll("attribute_value_id"));
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
    });
  }, [selectedTypeSlugs, selectedCategoryIds, selectedCatalogId, selectedAttributeValueIds]);

  // Stable string keys so effects don't re-fire every render over a fresh
  // array reference.
  const categoryKey = selectedCategoryIds.slice().sort().join(",");
  const attributeKey = selectedAttributeValueIds.slice().sort().join(",");

  useEffect(() => {
    getNav(locale)
      .then(setNavTypes)
      .catch(() => {});
  }, [locale]);

  useEffect(() => {
    listFacets({ categoryIds: selectedCategoryIds, catalogId: selectedCatalogId, locale })
      .then(setFacets)
      .catch(() => setFacets([]));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [categoryKey, selectedCatalogId, locale]);

  useEffect(() => {
    setProducts(null);
    listStorefrontProducts({
      categoryIds: selectedCategoryIds,
      catalogId: selectedCatalogId,
      attributeValueIds: selectedAttributeValueIds,
      q: searchQuery || undefined,
      locale,
    })
      .then(setProducts)
      .catch(() => setError("Could not load products."));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [categoryKey, selectedCatalogId, attributeKey, searchQuery, locale]);

  // Only types that own at least one selected category stay "active" — lets
  // us drive both the Type and Category groups from the same source of truth.
  const categoriesByType = useMemo(() => {
    const map = new Map<string, NavType["categories"]>();
    for (const type of navTypes) map.set(type.slug, type.categories);
    return map;
  }, [navTypes]);

  const visibleCategories = useMemo(() => {
    if (selectedTypeSlugs.length === 0) {
      return navTypes.flatMap((t) => t.categories);
    }
    return selectedTypeSlugs.flatMap((slug) => categoriesByType.get(slug) ?? []);
  }, [navTypes, selectedTypeSlugs, categoriesByType]);

  function toggleInList(list: string[], value: string): string[] {
    return list.includes(value) ? list.filter((v) => v !== value) : [...list, value];
  }

  function handleToggle(groupId: string, optionId: string) {
    if (groupId === "type") {
      const type = navTypes.find((t) => t.slug === optionId);
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
    } else {
      setSelectedAttributeValueIds((prev) => toggleInList(prev, optionId));
    }
  }

  function handleClear() {
    setSelectedTypeSlugs([]);
    setSelectedCategoryIds([]);
    setSelectedCatalogId(undefined);
    setSelectedAttributeValueIds([]);
  }

  const filterGroups: FilterGroup[] = [
    {
      id: "type",
      label: "Type",
      type: "checkbox",
      options: navTypes.map((t) => ({ id: t.slug, label: t.name })),
    },
    {
      id: "category",
      label: "Category",
      type: "checkbox",
      options: visibleCategories.map((c) => ({ id: c.id, label: c.name })),
    },
    ...facets.map(
      (facet): FilterGroup => ({
        id: facet.attribute_id,
        label: facet.attribute_name,
        type: "checkbox",
        options: facet.values.map((v) => ({ id: v.id, label: v.value })),
      }),
    ),
  ];

  const selected: Record<string, string[]> = {
    type: selectedTypeSlugs,
    category: selectedCategoryIds,
  };
  for (const facet of facets) {
    selected[facet.attribute_id] = selectedAttributeValueIds.filter((id) =>
      facet.values.some((v) => v.id === id),
    );
  }

  const heading = searchQuery
    ? `Search results for "${searchQuery}"`
    : selectedCategoryIds.length === 1
      ? visibleCategories.find((c) => c.id === selectedCategoryIds[0])?.name ?? "Shop"
      : selectedTypeSlugs.length === 1
        ? navTypes.find((t) => t.slug === selectedTypeSlugs[0])?.name ?? "Shop"
        : "All Products";

  function clearSearch() {
    const next = new URLSearchParams(searchParams);
    next.delete("q");
    navigate(`/shop${next.toString() ? `?${next.toString()}` : ""}`);
  }

  const breadcrumbItems = [
    { label: "Home", href: "/" },
    ...(selectedTypeSlugs.length === 1
      ? [{ label: navTypes.find((t) => t.slug === selectedTypeSlugs[0])?.name ?? selectedTypeSlugs[0], href: `/shop?type=${selectedTypeSlugs[0]}` }]
      : []),
    ...(selectedCategoryIds.length === 1 ? [{ label: heading }] : selectedTypeSlugs.length !== 1 ? [{ label: "Shop" }] : []),
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
                  Filters
                </Button>
                <Text size="sm" tone="muted">
                  {products ? `${products.length} item${products.length === 1 ? "" : "s"}` : "Loading…"}
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
                  No products match these filters yet.
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
                      outOfStock={!product.in_stock}
                      isWishlisted={isAuthenticated && isWishlisted(product.id)}
                      onToggleWishlist={isAuthenticated ? () => toggle(product.id) : undefined}
                    />
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </main>
      <Footer />

      <Modal open={mobileFiltersOpen} onClose={() => setMobileFiltersOpen(false)} title="Filters">
        <FilterPanel groups={filterGroups} selected={selected} onToggle={handleToggle} onClear={handleClear} />
      </Modal>
    </div>
  );
}
