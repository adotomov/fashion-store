import { Link } from "react-router";

import type { Money } from "../../lib/money/money";
import { cn } from "../../lib/utils/cn";
import { Badge } from "../ui/Badge";
import { Button } from "../ui/Button";
import { Icon } from "../ui/Icon";
import { Price } from "../ui/Price";
import { Text } from "../ui/Text";

export type ProductCardProps = {
  href: string;
  image?: { src: string; alt: string };
  title: string;
  price: Money;
  compareAtPrice?: Money;
  badge?: string;
  outOfStock?: boolean;
  isWishlisted?: boolean;
  onToggleWishlist?: () => void;
  className?: string;
};

export function ProductCard({
  href,
  image,
  title,
  price,
  compareAtPrice,
  badge,
  outOfStock = false,
  isWishlisted = false,
  onToggleWishlist,
  className,
}: ProductCardProps) {
  return (
    <div className={cn("group relative flex flex-col", className)}>
      <div className="relative aspect-[3/4] w-full overflow-hidden rounded-sm bg-stone-50">
        <Link to={href}>
          {image ? (
            <img
              src={image.src}
              alt={image.alt}
              className={cn(
                "h-full w-full object-cover transition-transform duration-300 group-hover:scale-105",
                outOfStock && "opacity-60",
              )}
            />
          ) : (
            <span className="flex h-full w-full items-center justify-center bg-gradient-to-br from-stone-100 to-stone-200 transition-colors group-hover:from-clay-50 group-hover:to-clay-100">
              <span className="font-display text-4xl text-stone-400 group-hover:text-clay-500">
                {title.charAt(0).toUpperCase()}
              </span>
            </span>
          )}
        </Link>

        {outOfStock ? (
          <Badge variant="danger" className="absolute left-3 top-3">
            Out of Stock
          </Badge>
        ) : (
          badge && <Badge variant="accent" className="absolute left-3 top-3">{badge}</Badge>
        )}

        <Button
          variant="outline"
          size="icon"
          aria-pressed={isWishlisted}
          aria-label="Add to wishlist"
          onClick={onToggleWishlist}
          className="absolute right-3 top-3 border-none bg-white/90 hover:bg-white"
        >
          <Icon name="wishlist" size={16} className={isWishlisted ? "fill-clay-500 text-clay-500" : undefined} />
        </Button>
      </div>

      <Link to={href} className="mt-3">
        <Text size="sm" className="line-clamp-2 font-medium">
          {title}
        </Text>
        <Price price={price} compareAtPrice={compareAtPrice} size="sm" className="mt-1.5" />
      </Link>
    </div>
  );
}
