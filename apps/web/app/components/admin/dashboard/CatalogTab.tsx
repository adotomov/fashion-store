import { useEffect, useState } from "react";

import { EmptyState } from "../EmptyState";
import { Badge } from "../../ui/Badge";
import { Card } from "../../ui/Card";
import { Eyebrow, Text } from "../../ui/Text";
import { type CatalogStats, getCatalogStats } from "../../../lib/api/products";

export function CatalogTab() {
  const [stats, setStats] = useState<CatalogStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getCatalogStats()
      .then(setStats)
      .catch(() => setError("Could not load catalog stats."));
  }, []);

  if (error) {
    return (
      <Text size="sm" tone="danger">
        {error}
      </Text>
    );
  }

  if (!stats) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-3 lg:grid-cols-5">
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            Total products
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.total_products}
          </Text>
        </Card>
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            Active
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.active_products}
          </Text>
        </Card>
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            Draft
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.draft_products}
          </Text>
        </Card>
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            Archived
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.archived_products}
          </Text>
        </Card>
        <Card className="p-5">
          <Text size="sm" className="font-medium text-stone-600">
            Variants
          </Text>
          <Text as="span" size="lg" className="mt-2 block font-display text-3xl font-medium text-stone-900">
            {stats.total_variants}
          </Text>
        </Card>
      </div>

      <section>
        <Eyebrow>Best sellers</Eyebrow>
        <Card className="mt-3 overflow-hidden p-0">
          {stats.top_products.length === 0 ? (
            <div className="p-6">
              <EmptyState
                icon="catalog"
                title="No sales yet"
                description="Best sellers appear here once orders come in with line items linked to products."
              />
            </div>
          ) : (
            <table className="w-full text-left text-sm">
              <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
                <tr>
                  <th className="px-4 py-3 font-medium">Rank</th>
                  <th className="px-4 py-3 font-medium">Product</th>
                  <th className="px-4 py-3 font-medium">Units sold</th>
                  <th className="px-4 py-3 font-medium">Orders</th>
                </tr>
              </thead>
              <tbody>
                {stats.top_products.map((p, i) => (
                  <tr key={p.product_id} className="border-b border-stone-100 last:border-0">
                    <td className="px-4 py-3 text-stone-500">
                      {i === 0 ? <Badge variant="success">#1</Badge> : `#${i + 1}`}
                    </td>
                    <td className="px-4 py-3 font-medium text-stone-900">{p.product_name}</td>
                    <td className="px-4 py-3 text-stone-600">{p.quantity_sold}</td>
                    <td className="px-4 py-3 text-stone-600">{p.order_count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
      </section>
    </div>
  );
}
