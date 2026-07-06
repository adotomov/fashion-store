import { useState } from "react";
import { Link } from "react-router";

import { useAdminPermissions } from "../../../features/admin/AdminPermissionsContext";

import { Badge } from "../../ui/Badge";
import { Button } from "../../ui/Button";
import { Icon } from "../../ui/Icon";
import { Input } from "../../ui/Input";
import { Price } from "../../ui/Price";
import { Select } from "../../ui/Select";
import { Text } from "../../ui/Text";
import type { Attribute } from "../../../lib/api/attributes";
import { type ProductVariant, createVariant, deleteVariant } from "../../../lib/api/products";

type ProductVariantsSectionProps = {
  productId: string;
  productName: string;
  variants: ProductVariant[];
  attributes: Attribute[];
  onChange: () => void;
};

export function variantDisplayLabel(variant: ProductVariant): string {
  return variant.attributes.map((a) => a.value).join(" / ") || "Default";
}

export function ProductVariantsSection({
  productId,
  productName,
  variants,
  attributes,
  onChange,
}: ProductVariantsSectionProps) {
  const { isReadOnly } = useAdminPermissions();
  const [selection, setSelection] = useState<Record<string, string>>({});
  const [priceOverride, setPriceOverride] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  async function handleAddVariant() {
    const attributeValueIds = Object.values(selection).filter(Boolean);
    if (attributeValueIds.length === 0) {
      setError("Choose at least one attribute value to define the variant.");
      return;
    }

    setIsSaving(true);
    setError(null);
    try {
      await createVariant(
        productId,
        attributeValueIds,
        priceOverride ? { amount: Math.round(Number(priceOverride) * 100), currency: "EUR" } : undefined,
      );
      setSelection({});
      setPriceOverride("");
      onChange();
    } catch {
      setError("Could not create variant. It may already exist.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete(variantId: string) {
    if (!window.confirm("Delete this variant? Any linked inventory item will be removed too.")) return;
    try {
      await deleteVariant(productId, variantId);
      onChange();
    } catch {
      setError("Could not delete variant.");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {variants.length === 0 ? (
        <Text size="sm" tone="muted">
          No variants yet.
        </Text>
      ) : (
        <ul className="flex flex-col gap-2">
          {variants.map((variant) => (
            <li
              key={variant.id}
              className="flex items-center justify-between rounded-sm border border-stone-200 px-4 py-3"
            >
              <div className="flex flex-wrap items-center gap-2">
                {variant.attributes.map((a) => (
                  <Badge key={a.id} variant="neutral">
                    {a.value}
                  </Badge>
                ))}
                {variant.price_override && <Price price={variant.price_override} size="sm" />}
              </div>
              <div className="flex items-center gap-1">
                {variant.inventory_item_id ? (
                  <Link
                    to={`/admin/inventory?highlightItemId=${variant.inventory_item_id}`}
                    className="text-sm font-medium text-stone-600 hover:underline"
                  >
                    View SKU
                  </Link>
                ) : (
                  <Link
                    to={`/admin/inventory?assignVariantId=${variant.id}&productName=${encodeURIComponent(productName)}&variantLabel=${encodeURIComponent(variantDisplayLabel(variant))}`}
                    className="text-sm font-medium text-clay-600 hover:underline"
                  >
                    Assign SKU
                  </Link>
                )}
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label="Delete variant"
                  onClick={() => handleDelete(variant.id)}
                  disabled={isReadOnly}
                  className="text-danger-600 hover:bg-danger-50"
                >
                  <Icon name="trash" size={15} />
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}

      <div className="rounded-sm border border-dashed border-stone-300 p-4">
        <Text size="sm" className="mb-3 font-medium">
          Add variant
        </Text>
        <div className="flex flex-wrap items-end gap-3">
          {attributes.map((attribute) => (
            <div key={attribute.id} className="w-36">
              <Text size="xs" tone="muted" className="mb-1">
                {attribute.name}
              </Text>
              <Select
                value={selection[attribute.id] ?? ""}
                onChange={(e) => setSelection((prev) => ({ ...prev, [attribute.id]: e.target.value }))}
                disabled={isReadOnly}
                className="h-9 text-sm"
              >
                <option value="">—</option>
                {attribute.values.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.value}
                  </option>
                ))}
              </Select>
            </div>
          ))}
          <div className="w-32">
            <Text size="xs" tone="muted" className="mb-1">
              Price override
            </Text>
            <Input
              type="number"
              step="0.01"
              placeholder="Optional"
              value={priceOverride}
              onChange={(e) => setPriceOverride(e.target.value)}
              className="h-9 text-sm"
            />
          </div>
          <Button variant="outline" size="sm" onClick={handleAddVariant} disabled={isSaving || isReadOnly}>
            {isSaving ? "Adding…" : "Add Variant"}
          </Button>
        </div>
      </div>
    </div>
  );
}
