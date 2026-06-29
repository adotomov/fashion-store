import { cn } from "../../lib/utils/cn";
import { Icon } from "./Icon";

type RatingProps = {
  value: number;
  count?: number;
  className?: string;
};

export function Rating({ value, count, className }: RatingProps) {
  return (
    <div className={cn("inline-flex items-center gap-1.5", className)}>
      <div className="flex items-center gap-0.5">
        {Array.from({ length: 5 }, (_, i) => (
          <Icon
            key={i}
            name="star"
            size={14}
            className={i < Math.round(value) ? "fill-clay-500 text-clay-500" : "text-stone-300"}
          />
        ))}
      </div>
      {count !== undefined && <span className="text-xs text-stone-500">({count})</span>}
    </div>
  );
}
