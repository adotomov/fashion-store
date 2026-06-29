import { cn } from "../../lib/utils/cn";

export type ColorOption = {
  name: string;
  /** any valid CSS color value */
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
        "h-8 w-8 rounded-full border-2 transition-all",
        selected ? "border-stone-900" : "border-transparent hover:border-stone-300",
      )}
    >
      <span className="block h-full w-full rounded-full border border-stone-300" style={{ backgroundColor: color.hex }} />
    </button>
  );
}
