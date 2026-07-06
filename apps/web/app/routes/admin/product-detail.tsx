import { useEffect, useState } from "react";
import { Link, useParams } from "react-router";

import { useAdminPermissions } from "../../features/admin/AdminPermissionsContext";

import { AssignmentSelector } from "../../components/admin/catalog/AssignmentSelector";
import { ProductMediaSection } from "../../components/admin/catalog/ProductMediaSection";
import { ProductVariantsSection, variantDisplayLabel } from "../../components/admin/catalog/ProductVariantsSection";
import { TranslationFields } from "../../components/admin/TranslationFields";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Icon } from "../../components/ui/Icon";
import { Input } from "../../components/ui/Input";
import { Modal } from "../../components/ui/Modal";
import { Select } from "../../components/ui/Select";
import { Textarea } from "../../components/ui/Textarea";
import { Eyebrow, Heading, Text } from "../../components/ui/Text";
import { type Attribute, listAttributes } from "../../lib/api/attributes";
import { type Catalog, listCatalogs } from "../../lib/api/catalogs";
import { type Category, listCategories } from "../../lib/api/categories";
import {
  type Product,
  type ProductStatus,
  getProduct,
  setProductAttributes,
  setProductCatalogs,
  setProductCategories,
  updateProduct,
} from "../../lib/api/products";

export const handle = { title: "Product" };

export default function ProductDetail() {
  const { isReadOnly } = useAdminPermissions();
  const { id } = useParams<{ id: string }>();
  const [product, setProduct] = useState<Product | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [catalogs, setCatalogs] = useState<Catalog[]>([]);
  const [attributes, setAttributes] = useState<Attribute[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [showInventoryWarning, setShowInventoryWarning] = useState(false);

  // Staged form state — none of this hits the API until "Save Changes" is
  // clicked. Variants/media stay self-contained with their own immediate
  // actions, since they're distinct sub-resource operations, not form fields.
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nksCode, setNksCode] = useState("");
  const [status, setStatus] = useState<ProductStatus>("draft");
  const [priceAmount, setPriceAmount] = useState("0");
  const [selectedCategoryIds, setSelectedCategoryIds] = useState<string[]>([]);
  const [selectedCatalogIds, setSelectedCatalogIds] = useState<string[]>([]);
  const [selectedAttributeIds, setSelectedAttributeIds] = useState<string[]>([]);

  // Full reload: re-syncs both the displayed product (variants/media/etc)
  // and the staged form fields from the server. Used on first load and
  // right after a successful Save.
  async function loadProduct() {
    if (!id) return;
    try {
      const loaded = await getProduct(id);
      setProduct(loaded);
      setName(loaded.name);
      setDescription(loaded.description);
      setNksCode(loaded.nks_code ?? "");
      setStatus(loaded.status);
      setPriceAmount((loaded.base_price.amount / 100).toFixed(2));
      setSelectedCategoryIds(loaded.category_ids ?? []);
      setSelectedCatalogIds(loaded.catalog_ids ?? []);
      setSelectedAttributeIds((loaded.attributes ?? []).map((a) => a.id));
    } catch {
      setError("Could not load product.");
    }
  }

  // Partial reload: used by Variants/Media after their own immediate API
  // calls. Only refreshes the displayed product data — it must NOT clobber
  // any in-progress, unsaved edits to the staged form fields above.
  async function refreshProductData() {
    if (!id) return;
    try {
      setProduct(await getProduct(id));
    } catch {
      setError("Could not refresh product.");
    }
  }

  useEffect(() => {
    loadProduct();
    listCategories().then(setCategories).catch(() => {});
    listCatalogs().then(setCatalogs).catch(() => {});
    listAttributes().then(setAttributes).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  async function handleSave() {
    if (!id) return;
    setIsSaving(true);
    setError(null);
    setSaved(false);
    try {
      await Promise.all([
        updateProduct(id, {
          name,
          description,
          nks_code: nksCode,
          status,
          base_price: { amount: Math.round(Number(priceAmount) * 100), currency: product?.base_price.currency ?? "EUR" },
        }),
        setProductCategories(id, selectedCategoryIds),
        setProductCatalogs(id, selectedCatalogIds),
        setProductAttributes(id, selectedAttributeIds),
      ]);
      await loadProduct();
      setSaved(true);
    } catch {
      setError("Could not save changes.");
    } finally {
      setIsSaving(false);
    }
  }

  // Only attributes explicitly selected as relevant for this product are
  // offered when composing variants — avoids showing every system-wide
  // attribute when only 2-3 matter here.
  const relevantAttributes = attributes.filter((attribute) => selectedAttributeIds.includes(attribute.id));

  const variantsMissingInventory = (product?.variants ?? []).filter((v) => !v.inventory_item_id);

  function handleSaveClick() {
    if (status === "active" && variantsMissingInventory.length > 0) {
      setShowInventoryWarning(true);
      return;
    }
    handleSave();
  }

  if (!id) return null;

  if (!product) {
    return (
      <Text size="sm" tone="muted">
        {error ?? "Loading…"}
      </Text>
    );
  }

  return (
    <div className="flex max-w-4xl flex-col gap-8">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <Link
            to="/admin/catalog?tab=products"
            aria-label="Back to products"
            className="rounded-sm p-1.5 text-stone-500 hover:bg-stone-100 hover:text-stone-900"
          >
            <Icon name="chevronLeft" size={18} />
          </Link>
          <div>
            <Heading as="h1" size="sm">
              {product.name}
            </Heading>
            <Text size="xs" tone="muted" className="mt-1">
              /{product.slug}
            </Text>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {saved && (
            <Text size="sm" tone="muted">
              Saved
            </Text>
          )}
          <Button variant="primary" onClick={handleSaveClick} disabled={isSaving || isReadOnly}>
            {isSaving ? "Saving…" : "Save Changes"}
          </Button>
        </div>
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      <section>
        <Eyebrow>Basics</Eyebrow>
        <Card className="mt-3 p-6">
          <div className="flex flex-col gap-4">
            <FormField label="Name" htmlFor="name">
              <Input id="name" value={name} onChange={(e) => setName(e.target.value)} />
            </FormField>
            <FormField label="Description" htmlFor="description">
              <Textarea id="description" value={description} onChange={(e) => setDescription(e.target.value)} />
            </FormField>
            <FormField label="НКС код (НАП)" htmlFor="nks-code" hint="Номенклатурен код на стоката — незадължително">
              <Input id="nks-code" value={nksCode} onChange={(e) => setNksCode(e.target.value)} placeholder="напр. 62.03.32" />
            </FormField>
            <TranslationFields
              entityType="product"
              entityId={product.id}
              fields={[
                { key: "name", label: "Name" },
                { key: "description", label: "Description" },
              ]}
            />
            <div className="grid grid-cols-2 gap-4">
              <FormField label="Status" htmlFor="status">
                <Select id="status" value={status} onChange={(e) => setStatus(e.target.value as ProductStatus)}>
                  <option value="draft">Draft</option>
                  <option value="active">Active</option>
                  <option value="archived">Archived</option>
                </Select>
              </FormField>
              <FormField label={`Price (${product.base_price.currency})`} htmlFor="price">
                <Input
                  id="price"
                  type="number"
                  step="0.01"
                  value={priceAmount}
                  onChange={(e) => setPriceAmount(e.target.value)}
                />
              </FormField>
            </div>
          </div>
        </Card>
      </section>

      <section>
        <Eyebrow>Categories</Eyebrow>
        <Card className="mt-3 p-6">
          <AssignmentSelector
            options={categories.map((c) => ({ id: c.id, label: c.name }))}
            selectedIds={selectedCategoryIds}
            onAdd={(catId) => setSelectedCategoryIds((prev) => [...prev, catId])}
            onRemove={(catId) => setSelectedCategoryIds((prev) => prev.filter((x) => x !== catId))}
            placeholder="Choose a category…"
            emptyMessage="No categories assigned yet."
          />
        </Card>
      </section>

      <section>
        <Eyebrow>Catalogs</Eyebrow>
        <Card className="mt-3 p-6">
          <AssignmentSelector
            options={catalogs.map((c) => ({ id: c.id, label: c.name }))}
            selectedIds={selectedCatalogIds}
            onAdd={(catId) => setSelectedCatalogIds((prev) => [...prev, catId])}
            onRemove={(catId) => setSelectedCatalogIds((prev) => prev.filter((x) => x !== catId))}
            placeholder="Choose a catalog…"
            emptyMessage="Not part of any catalog yet."
          />
        </Card>
      </section>

      <section>
        <Eyebrow>Product Attributes</Eyebrow>
        <Text size="xs" tone="muted" className="mt-1">
          Choose which attributes apply to this product (e.g. Size, Color). Variants below are built by picking a
          value per attribute, e.g. Size: M, Color: Blue.
        </Text>
        <Card className="mt-3 p-6">
          <AssignmentSelector
            options={attributes.map((a) => ({ id: a.id, label: a.name }))}
            selectedIds={selectedAttributeIds}
            onAdd={(attrId) => setSelectedAttributeIds((prev) => [...prev, attrId])}
            onRemove={(attrId) => setSelectedAttributeIds((prev) => prev.filter((x) => x !== attrId))}
            placeholder="Choose an attribute…"
            emptyMessage="No attributes selected yet."
          />
        </Card>
      </section>

      <section>
        <Eyebrow>Variants</Eyebrow>
        {relevantAttributes.length === 0 && (
          <Text size="xs" tone="muted" className="mt-1">
            Select at least one product attribute above to start building variants.
          </Text>
        )}
        <Card className="mt-3 p-6">
          <ProductVariantsSection
            productId={id}
            productName={product.name}
            variants={product.variants ?? []}
            attributes={relevantAttributes}
            onChange={refreshProductData}
          />
        </Card>
      </section>

      <section>
        <Eyebrow>Media</Eyebrow>
        <Card className="mt-3 p-6">
          <ProductMediaSection productId={id} media={product.media ?? []} onChange={refreshProductData} />
        </Card>
      </section>

      <Modal
        open={showInventoryWarning}
        onClose={() => setShowInventoryWarning(false)}
        title="Missing inventory"
      >
        <div className="flex flex-col gap-4">
          <Text size="sm" tone="muted">
            {variantsMissingInventory.length === 1
              ? "1 variant has no SKU assigned yet."
              : `${variantsMissingInventory.length} variants have no SKU assigned yet.`}{" "}
            Customers won't be able to tell these are out of stock until inventory is tracked.
          </Text>
          <ul className="flex flex-col gap-2">
            {variantsMissingInventory.map((variant) => (
              <li
                key={variant.id}
                className="flex items-center justify-between rounded-sm border border-stone-200 px-3 py-2"
              >
                <Text size="sm">{variantDisplayLabel(variant)}</Text>
                <a
                  href={`/admin/inventory?assignVariantId=${variant.id}&productName=${encodeURIComponent(product.name)}&variantLabel=${encodeURIComponent(variantDisplayLabel(variant))}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm font-medium text-clay-600 hover:underline"
                >
                  Assign SKU
                </a>
              </li>
            ))}
          </ul>
        </div>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setShowInventoryWarning(false)}>
            Cancel
          </Button>
          <Button
            variant="primary"
            onClick={() => {
              setShowInventoryWarning(false);
              handleSave();
            }}
          >
            Activate Anyway
          </Button>
        </div>
      </Modal>
    </div>
  );
}
