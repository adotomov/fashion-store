import { Text } from "../../ui/Text";

export type DailyChartPoint = {
  date: string;
  value: number;
};

type DailyChartProps = {
  points: DailyChartPoint[];
  emptyMessage?: string;
  formatValue?: (value: number) => string;
};

const dateFormatter = new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric" });

// Lightweight daily bar chart (no charting library) — bars sized by CSS
// height percentage relative to the period's max value.
export function DailyChart({ points, emptyMessage = "No data yet.", formatValue }: DailyChartProps) {
  if (points.length === 0) {
    return (
      <Text size="sm" tone="muted">
        {emptyMessage}
      </Text>
    );
  }

  const max = Math.max(...points.map((p) => p.value), 1);

  return (
    <div className="flex h-32 items-end gap-1">
      {points.map((point) => (
        <div key={point.date} className="group relative flex flex-1 flex-col items-center justify-end gap-1">
          <div className="pointer-events-none absolute -top-7 z-10 hidden whitespace-nowrap rounded-sm bg-stone-900 px-2 py-1 text-xs text-white group-hover:block">
            {dateFormatter.format(new Date(point.date))}: {formatValue ? formatValue(point.value) : point.value}
          </div>
          <div
            className="w-full rounded-sm bg-clay-400 transition-colors group-hover:bg-clay-600"
            style={{ height: `${Math.max((point.value / max) * 100, point.value > 0 ? 4 : 1)}%` }}
          />
        </div>
      ))}
    </div>
  );
}
