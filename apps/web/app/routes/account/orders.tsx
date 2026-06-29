import { Fragment, useEffect, useState } from "react";

import { EmptyState } from "../../components/admin/EmptyState";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Icon } from "../../components/ui/Icon";
import { Text } from "../../components/ui/Text";
import { type Order, type OrderStatus, listOrders } from "../../lib/api/orders";
import { formatMoney } from "../../lib/money/money";

export const handle = { title: "My Orders" };

const statusVariant: Record<OrderStatus, "neutral" | "brand" | "accent" | "success" | "danger"> = {
  pending: "neutral",
  paid: "accent",
  shipped: "brand",
  delivered: "success",
  cancelled: "danger",
};

function formatStatus(status: OrderStatus): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(undefined, { year: "numeric", month: "short", day: "numeric" });
}

function formatShipmentStatus(status: string): string {
  return status.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function Orders() {
  const [orders, setOrders] = useState<Order[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);

  useEffect(() => {
    listOrders()
      .then(setOrders)
      .catch(() => setError("Could not load your orders."));
  }, []);

  if (error) {
    return (
      <Text size="sm" tone="danger">
        {error}
      </Text>
    );
  }

  if (orders === null) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  if (orders.length === 0) {
    return <EmptyState icon="inventory" title="No orders yet" description="Your placed orders will show up here." />;
  }

  return (
    <div className="overflow-hidden rounded-sm border border-stone-200 bg-white">
      <table className="w-full text-left text-sm">
        <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
          <tr>
            <th className="px-4 py-3 font-medium">Order</th>
            <th className="px-4 py-3 font-medium">Date</th>
            <th className="px-4 py-3 font-medium">Status</th>
            <th className="px-4 py-3 font-medium">Items</th>
            <th className="px-4 py-3 font-medium">Total</th>
            <th className="px-4 py-3 font-medium" />
          </tr>
        </thead>
        <tbody>
          {orders.map((order) => {
            const isExpanded = expandedId === order.id;
            return (
              <Fragment key={order.id}>
                <tr className="border-b border-stone-100 last:border-0">
                  <td className="px-4 py-3 font-medium text-stone-900">{order.order_number}</td>
                  <td className="px-4 py-3 text-stone-600">{formatDate(order.placed_at)}</td>
                  <td className="px-4 py-3">
                    <Badge variant={statusVariant[order.status]}>{formatStatus(order.status)}</Badge>
                  </td>
                  <td className="px-4 py-3 text-stone-600">{order.items.length}</td>
                  <td className="px-4 py-3 font-medium text-stone-900">{formatMoney(order.total)}</td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setExpandedId(isExpanded ? null : order.id)}
                      aria-label={isExpanded ? "Hide order items" : "Show order items"}
                    >
                      <Icon name={isExpanded ? "chevronDown" : "chevronRight"} size={16} />
                    </Button>
                  </td>
                </tr>
                {isExpanded && (
                  <tr className="border-b border-stone-100 bg-stone-50 last:border-0">
                    <td colSpan={6} className="px-4 py-3">
                      {order.tracking_number && (
                        <div className="mb-3 flex flex-wrap items-center gap-2 text-sm">
                          <Text size="sm" className="font-medium">
                            {order.carrier ?? "Carrier"} — {order.tracking_number}
                          </Text>
                          {order.shipment_status && <Badge variant="brand">{formatShipmentStatus(order.shipment_status)}</Badge>}
                        </div>
                      )}
                      <div className="flex flex-col gap-2">
                        {order.items.map((item) => (
                          <div key={item.id} className="flex items-center justify-between text-sm">
                            <span className="text-stone-700">
                              {item.product_name}
                              {item.variant_label ? ` — ${item.variant_label}` : ""}
                              <span className="ml-2 text-stone-400">× {item.quantity}</span>
                            </span>
                            <span className="text-stone-600">{formatMoney(item.unit_price)}</span>
                          </div>
                        ))}
                      </div>
                    </td>
                  </tr>
                )}
              </Fragment>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
