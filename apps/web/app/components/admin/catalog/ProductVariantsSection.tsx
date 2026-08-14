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
import {
  type ProductVariant,
  type VariantDimensions,
  createVariant,
  deleteVariant,
  updateVariant,
} from "../../../lib/api/products";

const EMPTY_DIMENSIONS = { weight: "", length: "", width: "", height: "" };
type DimensionFields = typeof EMPTY_DIMENSIONS;

// toDimensions turns the four free-text fields into the integer grams/cm the
// API expects; blank fields become 0 (unset).
function toDimensions(d: DimensionFields): VariantDimensions {
  return {
    weightGrams: Math.max(0, Math.round(Number(d.weight) || 0)),
    lengthCm: Math.max(0, Math.round(Number(d.length) || 0)),
    widthCm: Math.max(0, Math.round(Number(d.width) || 0)),
    heightCm: Math.max(0, Math.round(Number(d.height) || 0)),
  };
}

function dimensionsFromVariant(v: ProductVariant): DimensionFields {
  return {
    weight: v.weight_grams ? String(v.weight_grams) : "",
    length: v.length_cm ? String(v.length_cm) : "",
    width: v.width_cm ? String(v.width_cm) : "",
    height: v.height_cm ? String(v.height_cm) : "",
  };
}

type ProductVariantsSectionProps = {
  productId: string;
  productName: string;
  variants: ProductVariant[];
  attributes: Attribute[];
  // The product's category identifier (e.g. "DR-01"), passed to the "Assign
  // SKU" screen so it can pre-fill the read-only SKU prefix. Empty when the
  // product has no category or the category has no identifier.
  categoryIdentifier: string;
  onChange: () => void;
};

export function variantDisplayLabel(variant: ProductVariant): string {
  return variant.attributes.map((a) => a.value).join(" / ") || "Default";
}

// DimensionInputs renders the shared weight (g) + L/W/H (cm) number fields used
// by both the add-variant form and the inline shipping editor.
function DimensionInputs({
  values,
  onChange,
  disabled,
}: {
  values: DimensionFields;
  onChange: (next: DimensionFields) => void;
  disabled?: boolean;
}) {
  const fields: { key: keyof DimensionFields; label: string }[] = [
    { key: "weight", label: "Weight (g)" },
    { key: "length", label: "Length (cm)" },
    { key: "width", label: "Width (cm)" },
    { key: "height", label: "Height (cm)" },
  ];
  return (
    <>
      {fields.map((f) => (
        <div key={f.key} className="w-24">
          <Text size="xs" tone="muted" className="mb-1">
            {f.label}
          </Text>
          <Input
            type="number"
            min="0"
            step="1"
            placeholder="0"
            value={values[f.key]}
            onChange={(e) => onChange({ ...values, [f.key]: e.target.value })}
            disabled={disabled}
            className="h-9 text-sm"
          />
        </div>
      ))}
    </>
  );
}

export function ProductVariantsSection({
  productId,
  productName,
  variants,
  attributes,
  categoryIdentifier,
  onChange,
}: ProductVariantsSectionProps) {
  const { isReadOnly } = useAdminPermissions();
  const [selection, setSelection] = useState<Record<string, string>>({});
  const [priceOverride, setPriceOverride] = useState("");
  const [dimensions, setDimensions] = useState<DimensionFields>(EMPTY_DIMENSIONS);
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  // The variant whose shipping attributes are being edited inline, plus its
  // working field values.
  const [editingId, setEditingId] = useState<string | null>(null);
  const [editDimensions, setEditDimensions] = useState<DimensionFields>(EMPTY_DIMENSIONS);
  const [isEditSaving, setIsEditSaving] = useState(false);

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
        toDimensions(dimensions),
      );
      setSelection({});
      setPriceOverride("");
      setDimensions(EMPTY_DIMENSIONS);
      onChange();
    } catch {
      setError("Could not create variant. It may already exist.");
    } finally {
      setIsSaving(false);
    }
  }

  function startEditing(variant: ProductVariant) {
    setEditingId(variant.id);
    setEditDimensions(dimensionsFromVariant(variant));
    setError(null);
  }

  async function handleSaveShipping(variant: ProductVariant) {
    setIsEditSaving(true);
    setError(null);
    try {
      await updateVariant(productId, variant.id, {
        attributeValueIds: variant.attribute_value_ids,
        // Preserve the existing price override — this inline editor only touches
        // shipping attributes.
        priceOverride: variant.price_override,
        clearPriceOverride: !variant.price_override,
        dimensions: toDimensions(editDimensions),
      });
      setEditingId(null);
      onChange();
    } catch {
      setError("Could not update variant shipping details.");
    } finally {
      setIsEditSaving(false);
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
            <li key={variant.id} className="rounded-sm border border-stone-200 px-4 py-3">
              <div className="flex items-center justify-between">
                <div className="flex flex-wrap items-center gap-2">
                  {variant.attributes.map((a) => (
                    <Badge key={a.id} variant="neutral">
                      {a.value}
                    </Badge>
                  ))}
                  {variant.price_override && <Price price={variant.price_override} size="sm" />}
                  {variant.weight_grams > 0 && (
                    <Text size="xs" tone="muted">
                      {variant.weight_grams} g
                    </Text>
                  )}
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label="Edit shipping details"
                    onClick={() => (editingId === variant.id ? setEditingId(null) : startEditing(variant))}
                    disabled={isReadOnly}
                    className="text-stone-600"
                  >
                    <Icon name="shipping" size={15} />
                  </Button>
                  {variant.inventory_item_id ? (
                    <Link
                      to={`/admin/inventory?highlightItemId=${variant.inventory_item_id}`}
                      className="text-sm font-medium text-stone-600 hover:underline"
                    >
                      View SKU
                    </Link>
                  ) : (
                    <Link
                      to={`/admin/inventory?assignVariantId=${variant.id}&productName=${encodeURIComponent(productName)}&variantLabel=${encodeURIComponent(variantDisplayLabel(variant))}&categoryIdentifier=${encodeURIComponent(categoryIdentifier)}`}
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
              </div>

              {editingId === variant.id && (
                <div className="mt-3 flex flex-wrap items-end gap-3 border-t border-stone-100 pt-3">
                  <DimensionInputs values={editDimensions} onChange={setEditDimensions} disabled={isReadOnly} />
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleSaveShipping(variant)}
                    disabled={isEditSaving || isReadOnly}
                  >
                    {isEditSaving ? "Saving…" : "Save"}
                  </Button>
                </div>
              )}
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
          <DimensionInputs values={dimensions} onChange={setDimensions} disabled={isReadOnly} />
          <Button variant="outline" size="sm" onClick={handleAddVariant} disabled={isSaving || isReadOnly}>
            {isSaving ? "Adding…" : "Add Variant"}
          </Button>
        </div>
      </div>
    </div>
  );
}
