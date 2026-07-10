import { useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";

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
  type AdminAdjustableMovementType,
  type InventoryItem,
  type InventoryMovement,
  adjustStock,
  createInventoryItem,
  listInventoryItems,
  listMovements,
} from "../../lib/api/inventory";
import { getProduct, listProducts, type ProductVariant } from "../../lib/api/products";
import { listCategories } from "../../lib/api/categories";
import { cn } from "../../lib/utils/cn";

export const handle = { title: "Inventory" };

const dateFormatter = new Intl.DateTimeFormat("en-US", { dateStyle: "medium", timeStyle: "short" });

function formatDate(value: string): string {
  return dateFormatter.format(new Date(value));
}

function variantLabel(variant: ProductVariant): string {
  return variant.attributes.map((a) => a.value).join(" / ") || "Default";
}

export default function AdminInventory() {
  const { isReadOnly } = useAdminPermissions();
  const [searchParams, setSearchParams] = useSearchParams();
  const [items, setItems] = useState<InventoryItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [variantOptions, setVariantOptions] = useState<
    { id: string; label: string; categoryIdentifier: string }[]
  >([]);
  const [isLoadingVariants, setIsLoadingVariants] = useState(false);
  const [selectedVariantId, setSelectedVariantId] = useState("");
  // Set when arriving via a deep link from the product editor (an "Assign
  // SKU" shortcut next to a specific variant) — locks the variant field to
  // that one instead of making the admin re-find it in the dropdown.
  const [lockedVariantLabel, setLockedVariantLabel] = useState<string | null>(null);
  const [newSku, setNewSku] = useState("");
  // When set, SKUs are composed as `${skuPrefix}-${newSku}` and newSku holds
  // only the free-text suffix (max 8 chars). Empty = plain free-form SKU
  // (fallback for variants whose product has no category identifier).
  const [skuPrefix, setSkuPrefix] = useState("");
  const [newInitialQuantity, setNewInitialQuantity] = useState("0");
  const [createError, setCreateError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const highlightItemId = searchParams.get("highlightItemId");
  const highlightedRowRef = useRef<HTMLTableRowElement | null>(null);

  const [adjustItem, setAdjustItem] = useState<InventoryItem | null>(null);
  const [adjustType, setAdjustType] = useState<AdminAdjustableMovementType>("admin_adjustment");
  const [adjustDelta, setAdjustDelta] = useState("0");
  const [adjustNote, setAdjustNote] = useState("");
  const [adjustError, setAdjustError] = useState<string | null>(null);

  const [movementsItem, setMovementsItem] = useState<InventoryItem | null>(null);
  const [movements, setMovements] = useState<InventoryMovement[] | null>(null);

  async function refresh() {
    try {
      setItems(await listInventoryItems());
    } catch {
      setError("Could not load inventory items.");
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  // Deep link from the product editor's "Assign SKU" shortcut: open the
  // modal pre-locked to that variant without making the admin re-select it.
  useEffect(() => {
    const assignVariantId = searchParams.get("assignVariantId");
    if (!assignVariantId) return;

    const productName = searchParams.get("productName") ?? "";
    const variantLabelParam = searchParams.get("variantLabel") ?? "";
    setSelectedVariantId(assignVariantId);
    setLockedVariantLabel(productName ? `${productName} — ${variantLabelParam || "Default"}` : variantLabelParam);
    setNewSku("");
    setSkuPrefix(searchParams.get("categoryIdentifier") ?? "");
    setNewInitialQuantity("0");
    setCreateError(null);
    setIsCreateOpen(true);

    setSearchParams(
      (prev) => {
        const next = new URLSearchParams(prev);
        next.delete("assignVariantId");
        next.delete("productName");
        next.delete("variantLabel");
        next.delete("categoryIdentifier");
        return next;
      },
      { replace: true },
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!highlightItemId || !items) return;
    highlightedRowRef.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, [highlightItemId, items]);

  async function openCreateModal() {
    setSelectedVariantId("");
    setLockedVariantLabel(null);
    setNewSku("");
    setSkuPrefix("");
    setNewInitialQuantity("0");
    setCreateError(null);
    setIsCreateOpen(true);
    setIsLoadingVariants(true);
    try {
      const [products, categories] = await Promise.all([listProducts(), listCategories()]);
      const identifierByCategoryId = new Map(categories.map((c) => [c.id, c.internal_identifier]));
      const detailed = await Promise.all(products.map((p) => getProduct(p.id)));
      const options = detailed.flatMap((product) => {
        const categoryIdentifier = identifierByCategoryId.get(product.category_ids?.[0] ?? "") ?? "";
        return (product.variants ?? []).map((variant) => ({
          id: variant.id,
          label: `${product.name} — ${variantLabel(variant)}`,
          categoryIdentifier,
        }));
      });
      setVariantOptions(options);
    } catch {
      setCreateError("Could not load variants.");
    } finally {
      setIsLoadingVariants(false);
    }
  }

  async function handleCreate() {
    if (!selectedVariantId) {
      setCreateError("Choose a variant.");
      return;
    }
    const suffix = newSku.trim();
    if (!suffix) {
      setCreateError(skuPrefix ? "SKU suffix is required." : "SKU is required.");
      return;
    }
    if (skuPrefix && suffix.length > 8) {
      setCreateError("SKU suffix can be at most 8 characters.");
      return;
    }
    const composedSku = skuPrefix ? `${skuPrefix}-${suffix}` : suffix;
    setIsSaving(true);
    setCreateError(null);
    try {
      await createInventoryItem(selectedVariantId, composedSku, Number(newInitialQuantity) || 0);
      setIsCreateOpen(false);
      setLockedVariantLabel(null);
      await refresh();
    } catch {
      setCreateError("Could not create inventory item. The variant may already have one, or the SKU is taken.");
    } finally {
      setIsSaving(false);
    }
  }

  function openAdjustModal(item: InventoryItem) {
    setAdjustItem(item);
    setAdjustType("admin_adjustment");
    setAdjustDelta("0");
    setAdjustNote("");
    setAdjustError(null);
  }

  async function handleAdjust() {
    if (!adjustItem) return;
    const delta = Number(adjustDelta);
    if (!delta) {
      setAdjustError("Quantity delta cannot be zero.");
      return;
    }
    try {
      await adjustStock(adjustItem.id, adjustType, delta, adjustNote);
      setAdjustItem(null);
      await refresh();
    } catch {
      setAdjustError("Could not adjust stock. The result may go below zero.");
    }
  }

  async function openMovementsModal(item: InventoryItem) {
    setMovementsItem(item);
    setMovements(null);
    try {
      setMovements(await listMovements(item.id));
    } catch {
      setMovements([]);
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-end">
        <Button variant="primary" onClick={openCreateModal} disabled={isReadOnly}>
          <Icon name="plus" size={16} />
          Assign SKU
        </Button>
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {items === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : items.length === 0 ? (
        <EmptyState
          icon="inventory"
          title="No inventory items yet"
          description="Assign a SKU to a product variant to start tracking stock."
        />
      ) : (
        <div className="overflow-hidden rounded-sm border border-stone-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
              <tr>
                <th className="px-4 py-3 font-medium">Product</th>
                <th className="px-4 py-3 font-medium">Variant</th>
                <th className="px-4 py-3 font-medium">SKU</th>
                <th className="px-4 py-3 font-medium">On Hand</th>
                <th className="px-4 py-3 font-medium">Reserved</th>
                <th className="px-4 py-3 font-medium">Available</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr
                  key={item.id}
                  ref={item.id === highlightItemId ? highlightedRowRef : undefined}
                  className={cn(
                    "border-b border-stone-100 last:border-0",
                    item.id === highlightItemId && "bg-clay-50 ring-1 ring-inset ring-clay-300",
                  )}
                >
                  <td className="px-4 py-3 font-medium text-stone-900">{item.product_name}</td>
                  <td className="px-4 py-3 text-stone-600">{item.variant_label || "Default"}</td>
                  <td className="px-4 py-3 font-mono text-xs text-stone-500">{item.sku}</td>
                  <td className="px-4 py-3 text-stone-600">{item.quantity_on_hand}</td>
                  <td className="px-4 py-3 text-stone-600">{item.quantity_reserved}</td>
                  <td className="px-4 py-3">
                    <Badge variant={item.quantity_available > 0 ? "success" : "danger"}>
                      {item.quantity_available}
                    </Badge>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex gap-1">
                      <Button variant="ghost" size="sm" onClick={() => openAdjustModal(item)} disabled={isReadOnly}>
                        Adjust
                      </Button>
                      <Button variant="ghost" size="sm" onClick={() => openMovementsModal(item)}>
                        History
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal
        open={isCreateOpen}
        onClose={() => {
          setIsCreateOpen(false);
          setLockedVariantLabel(null);
        }}
        title="Assign SKU"
      >
        <div className="flex flex-col gap-4">
          <FormField label="Variant" htmlFor="variant" error={createError ?? undefined}>
            {lockedVariantLabel !== null ? (
              <Text size="sm" className="rounded-sm border border-stone-200 bg-stone-50 px-3 py-2.5 font-medium">
                {lockedVariantLabel}
              </Text>
            ) : (
              <Select
                id="variant"
                value={selectedVariantId}
                onChange={(e) => {
                  setSelectedVariantId(e.target.value);
                  const opt = variantOptions.find((o) => o.id === e.target.value);
                  setSkuPrefix(opt?.categoryIdentifier ?? "");
                  setNewSku("");
                }}
                disabled={isLoadingVariants}
              >
                <option value="">{isLoadingVariants ? "Loading…" : "Select a variant"}</option>
                {variantOptions.map((opt) => (
                  <option key={opt.id} value={opt.id}>
                    {opt.label}
                  </option>
                ))}
              </Select>
            )}
          </FormField>
          <FormField
            label="SKU"
            htmlFor="sku"
            hint={skuPrefix ? `Final SKU: ${skuPrefix}-${newSku.trim() || "…"}` : undefined}
          >
            {skuPrefix ? (
              <div className="flex items-stretch">
                <span className="inline-flex shrink-0 items-center whitespace-nowrap rounded-l-sm border border-r-0 border-stone-300 bg-stone-50 px-3 font-mono text-sm text-stone-600">
                  {skuPrefix}-
                </span>
                <Input
                  id="sku"
                  className="rounded-l-none"
                  value={newSku}
                  onChange={(e) => setNewSku(e.target.value)}
                  maxLength={8}
                  placeholder="ABC-RD"
                />
              </div>
            ) : (
              <Input id="sku" value={newSku} onChange={(e) => setNewSku(e.target.value)} placeholder="DRESS-S-BLK" />
            )}
          </FormField>
          <FormField label="Initial quantity" htmlFor="initial-quantity">
            <Input
              id="initial-quantity"
              type="number"
              min="0"
              value={newInitialQuantity}
              onChange={(e) => setNewInitialQuantity(e.target.value)}
            />
          </FormField>
        </div>
        <div className="mt-6 flex justify-end gap-3">
          <Button
            variant="outline"
            onClick={() => {
              setIsCreateOpen(false);
              setLockedVariantLabel(null);
            }}
            disabled={isSaving}
          >
            Cancel
          </Button>
          <Button variant="primary" onClick={handleCreate} disabled={isSaving || isReadOnly}>
            {isSaving ? "Saving…" : "Save"}
          </Button>
        </div>
      </Modal>

      <Modal open={adjustItem !== null} onClose={() => setAdjustItem(null)} title="Adjust Stock">
        <div className="flex flex-col gap-4">
          {adjustError && (
            <Text size="sm" tone="danger">
              {adjustError}
            </Text>
          )}
          <FormField label="Reason" htmlFor="adjust-type">
            <Select
              id="adjust-type"
              value={adjustType}
              onChange={(e) => setAdjustType(e.target.value as AdminAdjustableMovementType)}
            >
              <option value="admin_adjustment">Admin adjustment</option>
              <option value="return">Return</option>
              <option value="manual_correction">Manual correction</option>
            </Select>
          </FormField>
          <FormField label="Quantity delta" htmlFor="adjust-delta" hint="Positive to add stock, negative to remove">
            <Input id="adjust-delta" type="number" value={adjustDelta} onChange={(e) => setAdjustDelta(e.target.value)} />
          </FormField>
          <FormField label="Note" htmlFor="adjust-note">
            <Input id="adjust-note" value={adjustNote} onChange={(e) => setAdjustNote(e.target.value)} />
          </FormField>
        </div>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setAdjustItem(null)}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleAdjust} disabled={isReadOnly}>
            Save
          </Button>
        </div>
      </Modal>

      <Modal open={movementsItem !== null} onClose={() => setMovementsItem(null)} title="Movement History">
        {movements === null ? (
          <Text size="sm" tone="muted">
            Loading…
          </Text>
        ) : movements.length === 0 ? (
          <Text size="sm" tone="muted">
            No movements yet.
          </Text>
        ) : (
          <ul className="flex max-h-96 flex-col gap-2 overflow-y-auto">
            {movements.map((m) => (
              <li key={m.id} className="rounded-sm border border-stone-200 p-3 text-sm">
                <div className="flex items-center justify-between">
                  <Badge variant="neutral">{m.type}</Badge>
                  <span className={m.quantity_delta >= 0 ? "text-sage-600" : "text-danger-600"}>
                    {m.quantity_delta >= 0 ? "+" : ""}
                    {m.quantity_delta}
                  </span>
                </div>
                {m.note && <Text size="xs" tone="muted" className="mt-1">{m.note}</Text>}
                <Text size="xs" tone="muted" className="mt-1">
                  {formatDate(m.created_at)}
                </Text>
              </li>
            ))}
          </ul>
        )}
      </Modal>
    </div>
  );
}
