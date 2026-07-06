import { Fragment, useEffect, useState } from "react";

import { useAdminPermissions } from "../../features/admin/AdminPermissionsContext";

import { EmptyState } from "../../components/admin/EmptyState";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { FormField } from "../../components/ui/FormField";
import { Icon } from "../../components/ui/Icon";
import { Input } from "../../components/ui/Input";
import { Modal } from "../../components/ui/Modal";
import { Select } from "../../components/ui/Select";
import { Text } from "../../components/ui/Text";
import {
  type AdminOrder,
  type AdminOrderStatus,
  getAdminOrder,
  listAdminOrders,
  updateOrderFulfillment,
} from "../../lib/api/admin-orders";
import { formatMoney } from "../../lib/money/money";

export const handle = { title: "Orders" };

const statusVariant: Record<AdminOrderStatus, "neutral" | "brand" | "accent" | "success" | "danger"> = {
  pending: "neutral",
  paid: "accent",
  shipped: "brand",
  delivered: "success",
  cancelled: "danger",
};

const paymentMethodLabels: Record<string, string> = {
  cash_on_delivery: "Cash on Delivery",
  card_on_easybox: "Card on EasyBox",
  card_online: "Card Online",
};

function formatStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function formatDate(value: string): string {
  return new Date(value).toLocaleString(undefined, { dateStyle: "medium", timeStyle: "short" });
}

export default function AdminOrders() {
  const { isReadOnly } = useAdminPermissions();
  const [orders, setOrders] = useState<AdminOrder[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [statusFilter, setStatusFilter] = useState("");

  const [fulfillmentOrder, setFulfillmentOrder] = useState<AdminOrder | null>(null);
  const [fulfillmentStatus, setFulfillmentStatus] = useState<AdminOrderStatus>("pending");
  const [carrier, setCarrier] = useState("");
  const [trackingNumber, setTrackingNumber] = useState("");
  const [shipmentStatus, setShipmentStatus] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  async function refresh() {
    try {
      setOrders(await listAdminOrders(statusFilter ? { status: statusFilter } : undefined));
    } catch {
      setError("Could not load orders.");
    }
  }

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [statusFilter]);

  async function toggleExpand(order: AdminOrder) {
    const isExpanding = expandedId !== order.id;
    setExpandedId(isExpanding ? order.id : null);
    if (isExpanding && !order.viewed_by_admin_at) {
      try {
        const viewed = await getAdminOrder(order.id);
        setOrders((prev) => prev?.map((o) => (o.id === viewed.id ? viewed : o)) ?? prev);
      } catch {
        // viewing-state update failing shouldn't block expanding the row
      }
    }
  }

  function openFulfillmentModal(order: AdminOrder) {
    setFulfillmentOrder(order);
    setFulfillmentStatus(order.status);
    setCarrier(order.carrier ?? "");
    setTrackingNumber(order.tracking_number ?? "");
    setShipmentStatus(order.shipment_status ?? "");
    setSaveError(null);
  }

  async function handleSaveFulfillment() {
    if (!fulfillmentOrder) return;
    setIsSaving(true);
    setSaveError(null);
    try {
      const updated = await updateOrderFulfillment(fulfillmentOrder.id, {
        status: fulfillmentStatus,
        carrier,
        tracking_number: trackingNumber,
        shipment_status: shipmentStatus,
      });
      setOrders((prev) => prev?.map((o) => (o.id === updated.id ? updated : o)) ?? prev);
      setFulfillmentOrder(null);
    } catch {
      setSaveError("Could not save changes. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <FormField label="Status" htmlFor="status-filter" className="w-48">
          <Select id="status-filter" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
            <option value="">All statuses</option>
            <option value="pending">Pending</option>
            <option value="paid">Paid</option>
            <option value="shipped">Shipped</option>
            <option value="delivered">Delivered</option>
            <option value="cancelled">Cancelled</option>
          </Select>
        </FormField>
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {orders === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : orders.length === 0 ? (
        <EmptyState icon="invoices" title="No orders yet" description="Placed orders will show up here." />
      ) : (
        <div className="overflow-hidden rounded-sm border border-stone-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
              <tr>
                <th className="px-4 py-3 font-medium">Order</th>
                <th className="px-4 py-3 font-medium">Customer</th>
                <th className="px-4 py-3 font-medium">Placed</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Payment</th>
                <th className="px-4 py-3 font-medium">Total</th>
                <th className="px-4 py-3 font-medium">Tracking</th>
                <th className="px-4 py-3 font-medium" />
              </tr>
            </thead>
            <tbody>
              {orders.map((order) => {
                const isExpanded = expandedId === order.id;
                const isUnviewed = !order.viewed_by_admin_at;
                return (
                  <Fragment key={order.id}>
                    <tr className="border-b border-stone-100 last:border-0">
                      <td className="px-4 py-3 font-medium text-stone-900">
                        <span className="flex items-center gap-2">
                          {isUnviewed && <span className="h-2 w-2 rounded-full bg-clay-500" aria-label="Unviewed" />}
                          {order.order_number}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-stone-600">
                        <div>{order.contact_name}</div>
                        <div className="text-xs text-stone-400">{order.contact_email}</div>
                      </td>
                      <td className="px-4 py-3 text-stone-600">{formatDate(order.placed_at)}</td>
                      <td className="px-4 py-3">
                        <Badge variant={statusVariant[order.status]}>{formatStatus(order.status)}</Badge>
                      </td>
                      <td className="px-4 py-3 text-stone-600">{paymentMethodLabels[order.payment_method] ?? order.payment_method}</td>
                      <td className="px-4 py-3 font-medium text-stone-900">{formatMoney(order.total)}</td>
                      <td className="px-4 py-3 text-stone-600">
                        {order.tracking_number ? (
                          <span>
                            {order.carrier} — {order.tracking_number}
                          </span>
                        ) : (
                          <span className="text-stone-400">—</span>
                        )}
                      </td>
                      <td className="px-4 py-3 text-right">
                        <div className="flex justify-end gap-1">
                          <Button variant="ghost" size="sm" onClick={() => openFulfillmentModal(order)} disabled={isReadOnly}>
                            Update
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => toggleExpand(order)}
                            aria-label={isExpanded ? "Hide order details" : "Show order details"}
                          >
                            <Icon name={isExpanded ? "chevronDown" : "chevronRight"} size={16} />
                          </Button>
                        </div>
                      </td>
                    </tr>
                    {isExpanded && (
                      <tr className="border-b border-stone-100 bg-stone-50 last:border-0">
                        <td colSpan={8} className="px-4 py-4">
                          <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
                            <div>
                              <Text size="xs" tone="muted" className="uppercase tracking-wide">
                                Items
                              </Text>
                              <div className="mt-2 flex flex-col gap-1">
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
                            </div>
                            <div>
                              <Text size="xs" tone="muted" className="uppercase tracking-wide">
                                Shipping Address
                              </Text>
                              <div className="mt-2 text-sm text-stone-700">
                                <div>{order.shipping_address.recipient_name}</div>
                                <div>{order.shipping_address.line1}</div>
                                {order.shipping_address.line2 && <div>{order.shipping_address.line2}</div>}
                                <div>
                                  {order.shipping_address.city}
                                  {order.shipping_address.region ? `, ${order.shipping_address.region}` : ""}{" "}
                                  {order.shipping_address.postal_code}
                                </div>
                                <div>{order.shipping_address.country_code}</div>
                                {order.shipping_address.phone && <div className="mt-1">{order.shipping_address.phone}</div>}
                              </div>
                            </div>
                            <div>
                              <Text size="xs" tone="muted" className="uppercase tracking-wide">
                                Payment
                              </Text>
                              <div className="mt-2 text-sm text-stone-700">
                                {order.payment ? (
                                  <>
                                    <div>{order.payment.status === "succeeded" ? "Succeeded" : "Failed"}</div>
                                    {order.payment.provider_reference && (
                                      <div className="text-xs text-stone-400">{order.payment.provider_reference}</div>
                                    )}
                                  </>
                                ) : (
                                  <div className="text-stone-400">Settled at delivery</div>
                                )}
                                {order.shipment_status && (
                                  <div className="mt-3">
                                    <Text size="xs" tone="muted" className="uppercase tracking-wide">
                                      Shipment Status
                                    </Text>
                                    <div>{order.shipment_status}</div>
                                  </div>
                                )}
                              </div>
                            </div>
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
      )}

      <Modal open={fulfillmentOrder !== null} onClose={() => setFulfillmentOrder(null)} title="Update Order">
        <div className="flex flex-col gap-4">
          {saveError && (
            <Text size="sm" tone="danger">
              {saveError}
            </Text>
          )}
          <FormField label="Status" htmlFor="fulfillment-status">
            <Select
              id="fulfillment-status"
              value={fulfillmentStatus}
              onChange={(e) => setFulfillmentStatus(e.target.value as AdminOrderStatus)}
            >
              <option value="pending">Pending</option>
              <option value="paid">Paid</option>
              <option value="shipped">Shipped</option>
              <option value="delivered">Delivered</option>
              <option value="cancelled">Cancelled</option>
            </Select>
          </FormField>
          <FormField label="Carrier" htmlFor="fulfillment-carrier" hint="e.g. Speedy, EasyBox">
            <Input id="fulfillment-carrier" value={carrier} onChange={(e) => setCarrier(e.target.value)} />
          </FormField>
          <FormField label="Tracking number" htmlFor="fulfillment-tracking">
            <Input id="fulfillment-tracking" value={trackingNumber} onChange={(e) => setTrackingNumber(e.target.value)} />
          </FormField>
          <FormField label="Shipment status" htmlFor="fulfillment-shipment-status" hint="e.g. Label created, In transit, Delivered">
            <Input id="fulfillment-shipment-status" value={shipmentStatus} onChange={(e) => setShipmentStatus(e.target.value)} />
          </FormField>
        </div>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setFulfillmentOrder(null)} disabled={isSaving}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSaveFulfillment} disabled={isSaving || isReadOnly}>
            {isSaving ? "Saving…" : "Save"}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
