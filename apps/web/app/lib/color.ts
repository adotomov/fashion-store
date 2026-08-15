// The built-in "Multicolor" swatch is stored with this sentinel in place of a
// real hex. It represents multi-colored garments (prints, patterns) and renders
// as a rainbow wheel instead of a solid fill. Kept in sync with the backend
// domain.MulticolorHex sentinel; it is intentionally not a valid hex so it can
// only be seeded, never created through the admin color picker.
export const MULTICOLOR_HEX = "multicolor";

export function isMulticolor(hex?: string | null): boolean {
  return hex === MULTICOLOR_HEX;
}

// CSS `background` value for a color swatch: a rainbow conic-gradient wheel for
// the built-in multicolor value, otherwise the solid picked color. Use with the
// shorthand `background` property (not `backgroundColor`) so the gradient works.
export function swatchBackground(hex?: string | null): string {
  return isMulticolor(hex)
    ? "conic-gradient(from 90deg, #ef4444, #f59e0b, #eab308, #22c55e, #3b82f6, #8b5cf6, #ec4899, #ef4444)"
    : (hex ?? "transparent");
}
