import { type ReactNode, useEffect, useState } from "react";
import { Link, useParams } from "react-router";

import { Breadcrumbs } from "../components/ecommerce/Breadcrumbs";
import { Footer } from "../components/ecommerce/Footer";
import { Header } from "../components/ecommerce/Header";
import { ProductImageGallery } from "../components/ecommerce/ProductImageGallery";
import { Badge } from "../components/ui/Badge";
import { Button } from "../components/ui/Button";
import { Icon } from "../components/ui/Icon";
import { Price } from "../components/ui/Price";
import { Heading, Text } from "../components/ui/Text";
import { useAuth } from "../features/auth/AuthContext";
import { useCart } from "../features/cart/CartContext";
import { useLanguage } from "../features/i18n/LanguageContext";
import { useWishlist } from "../features/wishlist/WishlistContext";
import { cn } from "../lib/utils/cn";
import {
  type StorefrontProductDetail,
  type StorefrontVariant,
  getStorefrontProduct,
  resolveImageUrl,
} from "../lib/api/storefront";

// Absent quantity_available means no inventory item has been assigned to
// the variant yet — treated the same as zero, since inventory is tracked
// per variant and an unassigned variant can't actually be fulfilled.
function isVariantInStock(variant: StorefrontVariant): boolean {
  return typeof variant.quantity_available === "number" && variant.quantity_available > 0;
}

export const handle = { title: "Product" };

export default function ProductDetail() {
  const { slug } = useParams<{ slug: string }>();
  const { locale } = useLanguage();
  const [product, setProduct] = useState<StorefrontProductDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  // attribute_id -> selected attribute_value_id
  const [selectedValues, setSelectedValues] = useState<Record<string, string>>({});
  const { isAuthenticated } = useAuth();
  const { isWishlisted, toggle } = useWishlist();
  const { addItem } = useCart();
  const [isAddingToCart, setIsAddingToCart] = useState(false);
  const [addToCartError, setAddToCartError] = useState<string | null>(null);
  const [addedToCart, setAddedToCart] = useState(false);

  useEffect(() => {
    if (!slug) return;
    setProduct(null);
    setError(null);
    getStorefrontProduct(slug, locale)
      .then((loaded) => {
        setProduct(loaded);
        const firstVariant = loaded.variants[0];
        if (firstVariant) {
          const initial: Record<string, string> = {};
          for (const value of firstVariant.attributes) initial[value.attribute_id] = value.id;
          setSelectedValues(initial);
        }
      })
      .catch(() => setError("Could not load this product."));
  }, [slug, locale]);

  if (error) {
    return (
      <Shell>
        <Text size="sm" tone="danger" className="py-16 text-center">
          {error}
        </Text>
      </Shell>
    );
  }

  if (!product) {
    return (
      <Shell>
        <Text size="sm" tone="muted" className="py-16 text-center">
          Loading…
        </Text>
      </Shell>
    );
  }

  // The values available for one attribute, drawn from every variant — used
  // to render that attribute's selectable pills.
  function valuesForAttribute(attributeId: string) {
    const seen = new Map<string, string>();
    for (const variant of product!.variants) {
      const match = variant.attributes.find((a) => a.attribute_id === attributeId);
      if (match) seen.set(match.id, match.value);
    }
    return Array.from(seen, ([id, value]) => ({ id, value }));
  }

  // Finds the variant matching the current selection, with one attribute
  // optionally overridden — used both for the actual selection and to
  // probe "what would happen if I picked this value instead" for disabling
  // out-of-stock options.
  function variantForSelection(overrides: Record<string, string>) {
    const candidate = { ...selectedValues, ...overrides };
    return product!.variants.find(
      (variant) =>
        variant.attributes.length === product!.attributes.length &&
        variant.attributes.every((a) => candidate[a.attribute_id] === a.id),
    );
  }

  // A value is disabled only when picking it resolves to a known,
  // out-of-stock variant — if no variant matches yet (still picking other
  // attributes), it stays selectable.
  function isValueOutOfStock(attributeId: string, valueId: string): boolean {
    const candidate = variantForSelection({ [attributeId]: valueId });
    return candidate !== undefined && !isVariantInStock(candidate);
  }

  const matchedVariant = variantForSelection({});
  const isOutOfStock = product.variants.length > 0 && (!matchedVariant || !isVariantInStock(matchedVariant));
  const price = matchedVariant?.price_override ?? product.base_price;
  const images = product.media.map((m) => ({ src: resolveImageUrl(m.url), alt: m.alt_text || product.name }));

  async function handleAddToCart() {
    if (!matchedVariant || isOutOfStock) return;
    setIsAddingToCart(true);
    setAddToCartError(null);
    setAddedToCart(false);
    try {
      await addItem(matchedVariant.id);
      setAddedToCart(true);
    } catch {
      setAddToCartError("Could not add this item to your cart.");
    } finally {
      setIsAddingToCart(false);
    }
  }

  return (
    <Shell>
      <div className="flex items-center gap-3">
        <Link
          to="/shop"
          aria-label="Back to products"
          className="rounded-sm p-1.5 text-stone-500 hover:bg-stone-100 hover:text-stone-900"
        >
          <Icon name="chevronLeft" size={18} />
        </Link>
        <Breadcrumbs
          items={[{ label: "Home", href: "/" }, { label: "Shop", href: "/shop" }, { label: product.name }]}
        />
      </div>

      <div className="mt-6 grid grid-cols-1 gap-10 lg:grid-cols-2">
        {images.length > 0 ? (
          <ProductImageGallery main={images[0]} thumbnails={images.slice(1)} />
        ) : (
          <div className="flex aspect-square w-full items-center justify-center rounded-sm bg-gradient-to-br from-stone-100 to-stone-200">
            <span className="font-display text-6xl text-stone-400">{product.name.charAt(0).toUpperCase()}</span>
          </div>
        )}

        <div className="flex flex-col gap-6">
          <div>
            <Heading as="h1" size="lg">
              {product.name}
            </Heading>
            <div className="mt-3 flex items-center gap-3">
              <Price price={price} compareAtPrice={product.compare_at_price} size="lg" />
              {isOutOfStock && <Badge variant="danger">Out of Stock</Badge>}
            </div>
          </div>

          {product.description && <Text tone="muted">{product.description}</Text>}

          {product.attributes.map((attribute) => (
            <div key={attribute.id}>
              <Text size="sm" className="mb-2 font-medium">
                {attribute.name}
              </Text>
              <div className="flex flex-wrap gap-2">
                {valuesForAttribute(attribute.id).map((value) => {
                  const isSelected = selectedValues[attribute.id] === value.id;
                  const outOfStock = isValueOutOfStock(attribute.id, value.id);
                  return (
                    <button
                      key={value.id}
                      type="button"
                      aria-pressed={isSelected}
                      disabled={outOfStock}
                      onClick={() => setSelectedValues((prev) => ({ ...prev, [attribute.id]: value.id }))}
                      className={cn(
                        "h-11 min-w-11 rounded-sm border px-3 text-sm font-medium transition-colors",
                        outOfStock
                          ? "cursor-not-allowed border-stone-200 text-stone-400"
                          : isSelected
                            ? "border-stone-900 bg-stone-900 text-white"
                            : "border-stone-300 text-stone-900 hover:border-stone-900",
                      )}
                    >
                      {value.value}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}

          <div className="flex gap-3">
            <Button
              variant="primary"
              size="lg"
              className="flex-1"
              disabled={!matchedVariant || isOutOfStock || isAddingToCart}
              onClick={handleAddToCart}
            >
              <Icon name="cart" size={18} />
              {isOutOfStock ? "Out of Stock" : isAddingToCart ? "Adding…" : addedToCart ? "Added" : "Add to Cart"}
            </Button>
            <Button
              variant="outline"
              size="lg"
              aria-pressed={isAuthenticated && isWishlisted(product.id)}
              aria-label="Add to wishlist"
              onClick={() => toggle(product.id)}
              disabled={!isAuthenticated}
            >
              <Icon
                name="wishlist"
                size={18}
                className={isAuthenticated && isWishlisted(product.id) ? "fill-clay-500 text-clay-500" : undefined}
              />
            </Button>
          </div>

          {addToCartError && (
            <Text size="sm" tone="danger">
              {addToCartError}
            </Text>
          )}
        </div>
      </div>
    </Shell>
  );
}

function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">{children}</div>
      </main>
      <Footer />
    </div>
  );
}
