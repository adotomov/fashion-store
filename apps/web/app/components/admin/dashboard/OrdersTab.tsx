import { useEffect, useState } from "react";

import { BarList } from "./BarList";
import { DailyChart } from "./DailyChart";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { Eyebrow, Text } from "../../ui/Text";
import { type OrderStats, type OrderStatsRange, getOrderStats } from "../../../lib/api/admin-orders";
import { COUNTRIES } from "../../../lib/data/countries";
import { formatMoney } from "../../../lib/money/money";

const ranges: { id: OrderStatsRange; label: string }[] = [
  { id: "7d", label: "Last 7 days" },
  { id: "30d", label: "Last 30 days" },
  { id: "90d", label: "Last 3 months" },
];

const statusLabels: Record<string, string> = {
  pending: "Pending",
  paid: "Paid",
  shipped: "Shipped",
  delivered: "Delivered",
  cancelled: "Cancelled",
};

const deliveryLabels: Record<string, string> = {
  speedy: "Speedy",
  easybox: "EasyBox",
};

function countryName(code: string): string {
  return COUNTRIES.find((c) => c.code === code)?.name ?? code;
}

export function OrdersTab() {
  const [range, setRange] = useState<OrderStatsRange>("7d");
  const [stats, setStats] = useState<OrderStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setStats(null);
    getOrderStats(range)
      .then(setStats)
      .catch(() => setError("Could not load order stats."));
  }, [range]);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center gap-2">
        {ranges.map((r) => (
          <Button key={r.id} variant={r.id === range ? "primary" : "outline"} size="sm" onClick={() => setRange(r.id)}>
            {r.label}
          </Button>
        ))}
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {!stats ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : (
        <>
          <div className="grid grid-cols-1 gap-5 sm:grid-cols-3">
            <Card className="p-5">
              <Text size="sm" className="font-medium text-stone-600">
                Orders
              </Text>
              <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
                {stats.order_count}
              </Text>
            </Card>
            <Card className="p-5">
              <Text size="sm" className="font-medium text-stone-600">
                Revenue
              </Text>
              <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
                {formatMoney(stats.revenue)}
              </Text>
            </Card>
            <Card className="p-5">
              <Text size="sm" className="font-medium text-stone-600">
                Avg. order value
              </Text>
              <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
                {formatMoney(stats.avg_order_value)}
              </Text>
            </Card>
          </div>

          <section>
            <Eyebrow>Orders per day</Eyebrow>
            <Card className="mt-3 p-6">
              <DailyChart points={stats.daily_counts.map((d) => ({ date: d.date, value: d.count }))} />
            </Card>
          </section>

          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
            <section>
              <Eyebrow>By status</Eyebrow>
              <Card className="mt-3 p-6">
                <BarList
                  items={stats.status_breakdown.map((b) => ({ label: statusLabels[b.label] ?? b.label, count: b.count }))}
                />
              </Card>
            </section>

            <section>
              <Eyebrow>By delivery method</Eyebrow>
              <Card className="mt-3 p-6">
                <BarList
                  items={stats.by_delivery_method.map((b) => ({ label: deliveryLabels[b.label] ?? b.label, count: b.count }))}
                />
              </Card>
            </section>

            <section>
              <Eyebrow>Top cities</Eyebrow>
              <Card className="mt-3 p-6">
                <BarList items={stats.by_city.map((b) => ({ label: b.label, count: b.count }))} />
              </Card>
            </section>

            <section>
              <Eyebrow>By country</Eyebrow>
              <Card className="mt-3 p-6">
                <BarList items={stats.by_country.map((b) => ({ label: countryName(b.label), count: b.count }))} />
              </Card>
            </section>
          </div>
        </>
      )}
    </div>
  );
}
