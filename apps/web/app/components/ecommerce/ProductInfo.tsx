import { useState } from "react";

import type { Money } from "../../lib/money/money";
import { cn } from "../../lib/utils/cn";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { type ColorOption, ColorSwatch } from "../ui/ColorSwatch";
import { Icon } from "../ui/Icon";
import { Price } from "../ui/Price";
import { Heading, Text } from "../ui/Text";

export type ProductVariant = {
  label: string;
  available?: boolean;
};

export type ProductInfoProps = {
  title: string;
  description: string;
  tags?: string[];
  price: Money;
  compareAtPrice?: Money;
  variants?: ProductVariant[];
  colors?: ColorOption[];
  isWishlisted?: boolean;
  onAddToCart?: (selection: { variant?: string; color?: string }) => void;
  onToggleWishlist?: () => void;
  className?: string;
};

export function ProductInfo({
  title,
  description,
  tags,
  price,
  compareAtPrice,
  variants,
  colors,
  isWishlisted = false,
  onAddToCart,
  onToggleWishlist,
  className,
}: ProductInfoProps) {
  const [selectedVariant, setSelectedVariant] = useState<string | undefined>(
    variants?.find((v) => v.available !== false)?.label,
  );
  const [selectedColor, setSelectedColor] = useState<ColorOption | undefined>(colors?.[0]);

  return (
    <div className={cn("flex flex-col gap-6", className)}>
      <div>
        {tags && tags.length > 0 && (
          <div className="mb-3 flex flex-wrap gap-2">
            {tags.map((tag) => (
              <Badge key={tag} variant="neutral">
                {tag}
              </Badge>
            ))}
          </div>
        )}
        <Heading as="h1" size="lg">
          {title}
        </Heading>
        <Price price={price} compareAtPrice={compareAtPrice} size="lg" className="mt-3" />
      </div>

      <Text tone="muted">{description}</Text>

      {variants && variants.length > 0 && (
        <div>
          <Text size="sm" className="mb-2 font-medium">
            Size
          </Text>
          <div className="flex flex-wrap gap-2">
            {variants.map((variant) => {
              const isSelected = selectedVariant === variant.label;
              const isAvailable = variant.available !== false;
              return (
                <button
                  key={variant.label}
                  type="button"
                  disabled={!isAvailable}
                  aria-pressed={isSelected}
                  onClick={() => setSelectedVariant(variant.label)}
                  className={cn(
                    "h-11 min-w-11 rounded-sm border px-3 text-sm font-medium transition-colors",
                    isSelected
                      ? "border-stone-900 bg-stone-900 text-white"
                      : "border-stone-300 text-stone-900 hover:border-stone-900",
                    !isAvailable && "cursor-not-allowed border-stone-200 text-stone-300 line-through hover:border-stone-200",
                  )}
                >
                  {variant.label}
                </button>
              );
            })}
          </div>
        </div>
      )}

      {colors && colors.length > 0 && (
        <div>
          <Text size="sm" className="mb-2 font-medium">
            Color{selectedColor ? `: ${selectedColor.name}` : ""}
          </Text>
          <div className="flex flex-wrap gap-3">
            {colors.map((color) => (
              <ColorSwatch
                key={color.name}
                color={color}
                selected={selectedColor?.name === color.name}
                onSelect={setSelectedColor}
              />
            ))}
          </div>
        </div>
      )}

      <div className="flex gap-3">
        <Button
          variant="primary"
          size="lg"
          className="flex-1"
          onClick={() => onAddToCart?.({ variant: selectedVariant, color: selectedColor?.name })}
        >
          <Icon name="cart" size={18} />
          Add to Cart
        </Button>
        <Button
          variant="outline"
          size="lg"
          aria-pressed={isWishlisted}
          aria-label="Add to wishlist"
          onClick={onToggleWishlist}
        >
          <Icon
            name="wishlist"
            size={18}
            className={isWishlisted ? "fill-clay-500 text-clay-500" : undefined}
          />
        </Button>
      </div>
    </div>
  );
}
