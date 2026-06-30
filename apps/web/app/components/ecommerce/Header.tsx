import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link, useLocation, useNavigate } from "react-router";

import { useAuth } from "../../features/auth/AuthContext";
import { UserMenu } from "../../features/auth/UserMenu";
import { useCart } from "../../features/cart/CartContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { useStoreBranding } from "../../features/store-settings/StoreSettingsContext";
import { useWishlist } from "../../features/wishlist/WishlistContext";
import { type NavType, getNav, resolveImageUrl } from "../../lib/api/storefront";
import { cn } from "../../lib/utils/cn";
import { Icon } from "../ui/Icon";
import { Text } from "../ui/Text";
import { LanguageSelector } from "./LanguageSelector";

type HeaderProps = {
  className?: string;
};

export function Header({ className }: HeaderProps) {
  const { isAuthenticated } = useAuth();
  const { itemCount: cartCount } = useCart();
  const { count: wishlistCount } = useWishlist();
  const { storeName, logoUrl } = useStoreBranding();
  const { locale } = useLanguage();
  const location = useLocation();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const [navTypes, setNavTypes] = useState<NavType[]>([]);
  const [openTypeId, setOpenTypeId] = useState<string | null>(null);
  const [expandedMobileTypeId, setExpandedMobileTypeId] = useState<string | null>(null);
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const searchInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    getNav(locale)
      .then(setNavTypes)
      .catch(() => {});
  }, [locale]);

  useEffect(() => {
    if (searchOpen) searchInputRef.current?.focus();
  }, [searchOpen]);

  function handleSearchSubmit(e: FormEvent) {
    e.preventDefault();
    const q = searchQuery.trim();
    setSearchOpen(false);
    setSearchQuery("");
    navigate(q ? `/shop?q=${encodeURIComponent(q)}` : "/shop", { state: { resetFilters: true } });
  }

  return (
    <header
      className={cn("relative z-30 border-b border-stone-200 bg-white", className)}
      onMouseLeave={() => setOpenTypeId(null)}
    >
      <div className="mx-auto flex h-20 max-w-7xl items-center justify-between gap-4 px-4 sm:px-6 lg:px-8">
        <button
          type="button"
          className="-ml-2 p-2 lg:hidden"
          aria-label="Toggle menu"
          onClick={() => setMenuOpen((open) => !open)}
        >
          <Icon name={menuOpen ? "close" : "menu"} size={22} />
        </button>

        <Link to="/" className="flex items-center gap-2">
          {logoUrl && <img src={logoUrl} alt={storeName} className="h-8 w-auto object-contain" />}
          <span className="font-display text-2xl font-medium tracking-wide text-stone-900">{storeName}</span>
        </Link>

        <nav className="hidden items-center gap-8 lg:flex">
          <Link
            to="/shop"
            state={{ resetFilters: true }}
            className="py-7 text-sm font-medium text-stone-700 transition-colors hover:text-stone-900"
          >
            Shop All
          </Link>
          {navTypes.map((type) => (
            <div key={type.id} onMouseEnter={() => setOpenTypeId(type.id)}>
              <Link
                to={`/shop?type=${type.slug}`}
                state={{ resetFilters: true }}
                className={cn(
                  "flex items-center gap-1 py-7 text-sm font-medium text-stone-700 transition-colors hover:text-stone-900",
                  openTypeId === type.id && "text-stone-900",
                )}
              >
                {type.name}
                {type.categories.length > 0 && <Icon name="chevronDown" size={14} />}
              </Link>
            </div>
          ))}
        </nav>

        <div className="flex items-center gap-1">
          <LanguageSelector />
          <div className="relative">
            {searchOpen ? (
              <form onSubmit={handleSearchSubmit} className="flex items-center">
                <input
                  ref={searchInputRef}
                  type="search"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  onBlur={() => {
                    if (!searchQuery) setSearchOpen(false);
                  }}
                  placeholder="Search products…"
                  aria-label="Search products"
                  className="h-9 w-44 rounded-sm border border-stone-300 px-3 text-sm focus:border-stone-900 focus:outline-none sm:w-56"
                />
              </form>
            ) : (
              <button
                type="button"
                aria-label="Search"
                className="rounded-sm p-2.5 hover:bg-stone-50"
                onClick={() => setSearchOpen(true)}
              >
                <Icon name="search" size={20} />
              </button>
            )}
          </div>
          {isAuthenticated ? (
            <UserMenu />
          ) : (
            <Link
              to="/login"
              state={{ from: { pathname: location.pathname, search: location.search } }}
              aria-label="Account"
              className="rounded-sm p-2.5 hover:bg-stone-50"
            >
              <Icon name="profile" size={20} />
            </Link>
          )}
          {isAuthenticated && (
            <Link to="/wishlist" aria-label="Wishlist" className="relative rounded-sm p-2.5 hover:bg-stone-50">
              <Icon name="wishlist" size={20} />
              {wishlistCount > 0 && <CountBadge count={wishlistCount} />}
            </Link>
          )}
          <Link to="/cart" aria-label="Cart" className="relative rounded-sm p-2.5 hover:bg-stone-50">
            <Icon name="cart" size={20} />
            {cartCount > 0 && <CountBadge count={cartCount} />}
          </Link>
        </div>
      </div>

      {/* Desktop mega-menu: a category grid with small photos, dropped down
          beneath the nav item currently being hovered. */}
      {navTypes.map(
        (type) =>
          type.categories.length > 0 && (
            <div
              key={type.id}
              className={cn(
                "absolute inset-x-0 top-full border-b border-stone-200 bg-white shadow-lg",
                openTypeId === type.id ? "block" : "hidden",
              )}
              onMouseEnter={() => setOpenTypeId(type.id)}
            >
              <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
                <div className="grid grid-cols-2 gap-6 sm:grid-cols-4 md:grid-cols-6">
                  {type.categories.map((category) => (
                    <Link
                      key={category.id}
                      to={`/shop?type=${type.slug}&category_id=${category.id}`}
                      state={{ resetFilters: true }}
                      onClick={() => setOpenTypeId(null)}
                      className="group flex flex-col gap-2.5"
                    >
                      <CategoryThumbnail category={category} />
                      <Text size="sm" className="text-center font-medium group-hover:text-clay-600">
                        {category.name}
                      </Text>
                    </Link>
                  ))}
                </div>
              </div>
            </div>
          ),
      )}

      {menuOpen && (
        <nav className="flex flex-col gap-1 border-t border-stone-200 px-4 py-3 lg:hidden">
          <Link
            to="/shop"
            state={{ resetFilters: true }}
            onClick={() => setMenuOpen(false)}
            className="rounded-sm px-2 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50"
          >
            Shop All
          </Link>
          {navTypes.map((type) => (
            <div key={type.id}>
              <div className="flex items-center justify-between">
                <Link
                  to={`/shop?type=${type.slug}`}
                  state={{ resetFilters: true }}
                  onClick={() => setMenuOpen(false)}
                  className="flex-1 rounded-sm px-2 py-2.5 text-sm font-medium text-stone-700 hover:bg-stone-50"
                >
                  {type.name}
                </Link>
                {type.categories.length > 0 && (
                  <button
                    type="button"
                    aria-label={`Toggle ${type.name} categories`}
                    className="p-2.5"
                    onClick={() => setExpandedMobileTypeId((id) => (id === type.id ? null : type.id))}
                  >
                    <Icon
                      name="chevronDown"
                      size={16}
                      className={cn("transition-transform", expandedMobileTypeId === type.id && "rotate-180")}
                    />
                  </button>
                )}
              </div>
              {expandedMobileTypeId === type.id && (
                <div className="ml-4 flex flex-col gap-1 border-l border-stone-200 pl-3">
                  {type.categories.map((category) => (
                    <Link
                      key={category.id}
                      to={`/shop?type=${type.slug}&category_id=${category.id}`}
                      state={{ resetFilters: true }}
                      onClick={() => setMenuOpen(false)}
                      className="rounded-sm px-2 py-2 text-sm text-stone-600 hover:bg-stone-50"
                    >
                      {category.name}
                    </Link>
                  ))}
                </div>
              )}
            </div>
          ))}
        </nav>
      )}
    </header>
  );
}

function CategoryThumbnail({ category }: { category: NavType["categories"][number] }) {
  if (category.image_url) {
    return (
      <span className="block aspect-square w-full overflow-hidden rounded-sm bg-stone-100">
        <img
          src={resolveImageUrl(category.image_url)}
          alt={category.name}
          className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-105"
        />
      </span>
    );
  }

  return (
    <span className="flex aspect-square w-full items-center justify-center rounded-sm bg-gradient-to-br from-stone-100 to-stone-200 transition-colors group-hover:from-clay-50 group-hover:to-clay-100">
      <span className="font-display text-2xl text-stone-400 group-hover:text-clay-500">
        {category.name.charAt(0).toUpperCase()}
      </span>
    </span>
  );
}

function CountBadge({ count }: { count: number }) {
  return (
    <span className="absolute right-1 top-1 flex h-4 w-4 items-center justify-center rounded-full bg-clay-500 text-[10px] font-medium text-white">
      {count > 9 ? "9+" : count}
    </span>
  );
}
