import { Text } from "../../ui/Text";

export type BarListItem = {
  label: string;
  count: number;
};

type BarListProps = {
  items: BarListItem[];
  emptyMessage?: string;
};

// Lightweight horizontal bar list for breakdowns (status/city/country/etc)
// — avoids pulling in a charting library for a handful of ranked bars.
export function BarList({ items, emptyMessage = "No data yet." }: BarListProps) {
  if (items.length === 0) {
    return (
      <Text size="sm" tone="muted">
        {emptyMessage}
      </Text>
    );
  }

  const max = Math.max(...items.map((i) => i.count), 1);

  return (
    <div className="flex flex-col gap-2.5">
      {items.map((item) => (
        <div key={item.label} className="flex items-center gap-3">
          <Text size="sm" className="w-28 shrink-0 truncate text-stone-700">
            {item.label}
          </Text>
          <div className="h-2 flex-1 overflow-hidden rounded-full bg-stone-100">
            <div
              className="h-full rounded-full bg-clay-500"
              style={{ width: `${Math.max((item.count / max) * 100, 4)}%` }}
            />
          </div>
          <Text size="sm" className="w-10 shrink-0 text-right font-medium text-stone-900">
            {item.count}
          </Text>
        </div>
      ))}
    </div>
  );
}
