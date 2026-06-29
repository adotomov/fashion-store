import { cn } from "../../lib/utils/cn";
import { Icon } from "./Icon";

type QuantityStepperProps = {
  quantity: number;
  onChange: (quantity: number) => void;
  min?: number;
  max?: number;
  className?: string;
};

export function QuantityStepper({ quantity, onChange, min = 1, max = 99, className }: QuantityStepperProps) {
  return (
    <div className={cn("inline-flex items-center rounded-sm border border-stone-300", className)}>
      <button
        type="button"
        aria-label="Decrease quantity"
        disabled={quantity <= min}
        onClick={() => onChange(Math.max(min, quantity - 1))}
        className="flex h-10 w-10 items-center justify-center text-stone-600 transition-colors hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-40"
      >
        <Icon name="minus" size={16} />
      </button>
      <span className="w-8 text-center text-sm font-medium text-stone-900">{quantity}</span>
      <button
        type="button"
        aria-label="Increase quantity"
        disabled={quantity >= max}
        onClick={() => onChange(Math.min(max, quantity + 1))}
        className="flex h-10 w-10 items-center justify-center text-stone-600 transition-colors hover:bg-stone-50 disabled:cursor-not-allowed disabled:opacity-40"
      >
        <Icon name="plus" size={16} />
      </button>
    </div>
  );
}
