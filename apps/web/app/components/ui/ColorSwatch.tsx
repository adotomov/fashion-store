import { swatchBackground } from "../../lib/color";
import { cn } from "../../lib/utils/cn";

export type ColorOption = {
  name: string;
  /** any valid CSS color value, or the "multicolor" sentinel for the wheel */
  hex: string;
};

type ColorSwatchProps = {
  color: ColorOption;
  selected?: boolean;
  onSelect?: (color: ColorOption) => void;
};

export function ColorSwatch({ color, selected, onSelect }: ColorSwatchProps) {
  return (
    <button
      type="button"
      title={color.name}
      aria-label={color.name}
      aria-pressed={selected}
      onClick={() => onSelect?.(color)}
      className={cn(
        "h-8 w-8 rounded-full ring-offset-2 ring-offset-white transition-all",
        selected
          ? "scale-110 ring-2 ring-stone-900"
          : "ring-1 ring-transparent hover:scale-105 hover:ring-stone-300",
      )}
    >
      <span className="block h-full w-full rounded-full border border-stone-300" style={{ background: swatchBackground(color.hex) }} />
    </button>
  );
}
