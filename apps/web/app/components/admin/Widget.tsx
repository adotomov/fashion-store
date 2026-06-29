import { Card } from "../ui/Card";
import { Icon, type IconName } from "../ui/Icon";
import { Text } from "../ui/Text";
import { cn } from "../../lib/utils/cn";

export type WidgetStat = {
  label: string;
  value: string | number;
};

export type WidgetProps = {
  title: string;
  icon: IconName;
  /** one stat (a single total) or several (e.g. 24h / 7d / 30d breakdowns) */
  stats: WidgetStat[];
  className?: string;
};

// Shared dashboard stat card. Add new dashboard metrics by rendering more
// <Widget> instances with whatever stats are relevant — no new component
// needed per metric.
export function Widget({ title, icon, stats, className }: WidgetProps) {
  return (
    <Card className={cn("p-5", className)}>
      <div className="flex items-center gap-2 text-stone-500">
        <Icon name={icon} size={16} />
        <Text size="sm" className="font-medium text-stone-600">
          {title}
        </Text>
      </div>

      <div className={cn("mt-4 flex", stats.length > 1 ? "justify-between gap-4" : "")}>
        {stats.map((stat) => (
          <div key={stat.label}>
            <Text as="span" size="lg" className="block font-display text-3xl font-medium text-stone-900">
              {stat.value}
            </Text>
            <Text size="xs" tone="muted" className="mt-1">
              {stat.label}
            </Text>
          </div>
        ))}
      </div>
    </Card>
  );
}
