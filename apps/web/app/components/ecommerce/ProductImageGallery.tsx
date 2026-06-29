import { useState } from "react";

import { cn } from "../../lib/utils/cn";

export type ProductImage = {
  src: string;
  alt: string;
};

type ProductImageGalleryProps = {
  /** spotlight image */
  main: ProductImage;
  /** 1 to 5 additional images shown as clickable thumbnails */
  thumbnails: ProductImage[];
  className?: string;
};

export function ProductImageGallery({ main, thumbnails, className }: ProductImageGalleryProps) {
  const images = [main, ...thumbnails.slice(0, 5)];
  const [activeIndex, setActiveIndex] = useState(0);
  const active = images[activeIndex] ?? main;

  return (
    <div className={cn("flex flex-col gap-3", className)}>
      <div className="aspect-square w-full overflow-hidden rounded-sm bg-stone-50">
        <img src={active.src} alt={active.alt} className="h-full w-full object-cover" />
      </div>

      {images.length > 1 && (
        <div className="flex gap-3">
          {images.map((image, index) => (
            <button
              key={image.src + index}
              type="button"
              aria-label={`Show image: ${image.alt}`}
              aria-current={index === activeIndex}
              onClick={() => setActiveIndex(index)}
              className={cn(
                "h-20 w-20 shrink-0 overflow-hidden rounded-sm border-2 bg-stone-50 transition-colors",
                index === activeIndex ? "border-stone-900" : "border-transparent hover:border-stone-300",
              )}
            >
              <img src={image.src} alt={image.alt} className="h-full w-full object-cover" />
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
