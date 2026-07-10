import { useEffect, useState } from "react";
import { Link, useBlocker, useParams } from "react-router";

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
import { type TaxGroup, listTaxGroups } from "../../lib/api/tax-groups";

export const handle = { title: "Product" };

// Serialized snapshot of the staged form fields, used to detect unsaved
// edits. Arrays are sorted so reordering an assignment doesn't read as a
// change. Variants/media aren't included — they save immediately via their
// own actions and never sit in the staged-but-unsaved state.
type FormValues = {
  name: string;
  description: string;
  status: ProductStatus;
  priceAmount: string;
  categoryId: string;
  taxGroupId: string;
  catalogIds: string[];
  attributeIds: string[];
};

function formSnapshot(v: FormValues): string {
  return JSON.stringify({
    name: v.name,
    description: v.description,
    status: v.status,
    priceAmount: v.priceAmount,
    categoryId: v.categoryId,
    taxGroupId: v.taxGroupId,
    catalogIds: [...v.catalogIds].sort(),
    attributeIds: [...v.attributeIds].sort(),
  });
}

export default function ProductDetail() {
  const { isReadOnly } = useAdminPermissions();
  const { id } = useParams<{ id: string }>();
  const [product, setProduct] = useState<Product | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [catalogs, setCatalogs] = useState<Catalog[]>([]);
  const [attributes, setAttributes] = useState<Attribute[]>([]);
  const [taxGroups, setTaxGroups] = useState<TaxGroup[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [nameError, setNameError] = useState<string | null>(null);
  // Snapshot of the last-saved form state; compared against the live fields
  // below to decide whether there are unsaved changes to warn about.
  const [baseline, setBaseline] = useState<string | null>(null);

  // Staged form state — none of this hits the API until "Save Changes" is
  // clicked. Variants/media stay self-contained with their own immediate
  // actions, since they're distinct sub-resource operations, not form fields.
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [status, setStatus] = useState<ProductStatus>("draft");
  const [priceAmount, setPriceAmount] = useState("0");
  const [selectedCategoryId, setSelectedCategoryId] = useState("");
  const [selectedTaxGroupId, setSelectedTaxGroupId] = useState("");
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
      setStatus(loaded.status);
      setPriceAmount((loaded.base_price.amount / 100).toFixed(2));
      setSelectedCategoryId(loaded.category_ids?.[0] ?? "");
      setSelectedTaxGroupId(loaded.tax_group_id ?? "");
      setSelectedCatalogIds(loaded.catalog_ids ?? []);
      setSelectedAttributeIds((loaded.attributes ?? []).map((a) => a.id));
      setBaseline(
        formSnapshot({
          name: loaded.name,
          description: loaded.description,
          status: loaded.status,
          priceAmount: (loaded.base_price.amount / 100).toFixed(2),
          categoryId: loaded.category_ids?.[0] ?? "",
          taxGroupId: loaded.tax_group_id ?? "",
          catalogIds: loaded.catalog_ids ?? [],
          attributeIds: (loaded.attributes ?? []).map((a) => a.id),
        }),
      );
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
    listTaxGroups().then(setTaxGroups).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);

  async function handleSave() {
    if (!id) return;
    setIsSaving(true);
    setError(null);
    setNameError(null);
    setSaved(false);
    try {
      await Promise.all([
        updateProduct(id, {
          name,
          description,
          status,
          base_price: { amount: Math.round(Number(priceAmount) * 100), currency: product?.base_price.currency ?? "EUR" },
          tax_group_id: selectedTaxGroupId,
        }),
        setProductCategories(id, selectedCategoryId ? [selectedCategoryId] : []),
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

  const variants = product?.variants ?? [];
  const variantsMissingInventory = variants.filter((v) => !v.inventory_item_id);

  // The selected category's identifier prefixes variant SKUs; passed to the
  // inventory "Assign SKU" screen so it can pre-fill the read-only prefix.
  const selectedCategoryIdentifier =
    categories.find((c) => c.id === selectedCategoryId)?.internal_identifier ?? "";

  // A product may only go Active once it's fully set up. Attributes and
  // categories reflect the *staged* selections (what the product will be
  // after Save); variants/SKUs reflect server state since those persist
  // immediately.
  const activationRequirements = [
    { met: Number(priceAmount) > 0, label: "Base price greater than 0" },
    { met: selectedAttributeIds.length > 0, label: "At least one attribute assigned" },
    { met: selectedCategoryId !== "", label: "Assigned to a category" },
    { met: selectedTaxGroupId !== "", label: "Tax group assigned" },
    { met: variants.length > 0, label: "At least one variant created" },
    { met: variants.length > 0 && variantsMissingInventory.length === 0, label: "SKU assigned to every variant" },
  ];
  const canActivate = activationRequirements.every((r) => r.met);

  // Unsaved-changes tracking. baseline is null until the product first loads;
  // once set, any divergence in the staged fields means there's work to save.
  const isDirty =
    baseline !== null &&
    baseline !==
      formSnapshot({
        name,
        description,
        status,
        priceAmount,
        categoryId: selectedCategoryId,
        taxGroupId: selectedTaxGroupId,
        catalogIds: selectedCatalogIds,
        attributeIds: selectedAttributeIds,
      });

  // Block in-app navigation (sidebar, back arrow, tab switches) while there
  // are unsaved edits; the modal below lets the user confirm or cancel.
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) => isDirty && currentLocation.pathname !== nextLocation.pathname,
  );

  // Native browser guard for full page unloads (refresh, tab close, external
  // links) — useBlocker only covers client-side routing.
  useEffect(() => {
    if (!isDirty) return;
    function handleBeforeUnload(e: BeforeUnloadEvent) {
      e.preventDefault();
      e.returnValue = "";
    }
    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [isDirty]);

  function handleSaveClick() {
    if (!name.trim()) {
      setNameError("Name is required.");
      setError("Please fix the errors below before saving.");
      return;
    }
    if (status === "active" && !canActivate) {
      setError("This product can't be activated until all activation requirements are met.");
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
          {isDirty ? (
            <Text size="sm" tone="muted">
              Unsaved changes
            </Text>
          ) : saved ? (
            <Text size="sm" tone="muted">
              Saved
            </Text>
          ) : null}
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
            <FormField label="Name" htmlFor="name" error={nameError ?? undefined}>
              <Input
                id="name"
                value={name}
                onChange={(e) => {
                  setName(e.target.value);
                  if (nameError) setNameError(null);
                }}
              />
            </FormField>
            <FormField label="Description" htmlFor="description">
              <Textarea id="description" value={description} onChange={(e) => setDescription(e.target.value)} />
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
                  <option value="active" disabled={!canActivate && status !== "active"}>
                    Active
                  </option>
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

            <FormField
              label="Tax group"
              htmlFor="tax-group"
              hint="VAT group used on invoices. Manage under Invoices & Taxes → Tax."
            >
              <Select
                id="tax-group"
                value={selectedTaxGroupId}
                onChange={(e) => setSelectedTaxGroupId(e.target.value)}
              >
                <option value="">No tax group</option>
                {taxGroups.map((g) => (
                  <option key={g.id} value={g.id}>
                    {g.identifier} — {g.vat_rate}% VAT
                  </option>
                ))}
              </Select>
            </FormField>

            {status !== "active" && !canActivate && (
              <div className="rounded-sm border border-stone-200 bg-stone-50 p-4">
                <Text size="xs" className="font-medium text-stone-700">
                  Complete these before activating:
                </Text>
                <ul className="mt-2 flex flex-col gap-1.5">
                  {activationRequirements.map((req) => (
                    <li key={req.label} className="flex items-center gap-2">
                      <Icon
                        name={req.met ? "check" : "close"}
                        size={14}
                        className={req.met ? "text-sage-600" : "text-stone-400"}
                      />
                      <Text size="xs" className={req.met ? "text-stone-500 line-through" : "text-stone-700"}>
                        {req.label}
                      </Text>
                    </li>
                  ))}
                </ul>
                {variants.length > 0 && variantsMissingInventory.length > 0 && (
                  <ul className="mt-3 flex flex-col gap-2 border-t border-stone-200 pt-3">
                    {variantsMissingInventory.map((variant) => (
                      <li key={variant.id} className="flex items-center justify-between gap-3">
                        <Text size="xs" tone="muted">
                          {variantDisplayLabel(variant)}
                        </Text>
                        <a
                          href={`/admin/inventory?assignVariantId=${variant.id}&productName=${encodeURIComponent(product.name)}&variantLabel=${encodeURIComponent(variantDisplayLabel(variant))}&categoryIdentifier=${encodeURIComponent(selectedCategoryIdentifier)}`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="shrink-0 text-xs font-medium text-clay-600 hover:underline"
                        >
                          Assign SKU
                        </a>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            )}
          </div>
        </Card>
      </section>

      <section>
        <Eyebrow>Category</Eyebrow>
        <Card className="mt-3 p-6">
          <FormField
            label="Category"
            htmlFor="category"
            hint={
              selectedCategoryIdentifier
                ? `Variant SKUs are prefixed with this category's identifier: ${selectedCategoryIdentifier}`
                : "A product belongs to a single category. Its identifier prefixes variant SKUs."
            }
          >
            <Select id="category" value={selectedCategoryId} onChange={(e) => setSelectedCategoryId(e.target.value)}>
              <option value="">No category</option>
              {categories.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                  {c.internal_identifier ? ` (${c.internal_identifier})` : ""}
                </option>
              ))}
            </Select>
          </FormField>
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
            categoryIdentifier={selectedCategoryIdentifier}
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
        open={blocker.state === "blocked"}
        onClose={() => blocker.reset?.()}
        title="Unsaved changes"
      >
        <Text size="sm" tone="muted">
          You have unsaved changes on this product. If you leave now, your changes will be lost.
        </Text>
        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => blocker.reset?.()}>
            Stay on page
          </Button>
          <Button variant="primary" onClick={() => blocker.proceed?.()}>
            Leave without saving
          </Button>
        </div>
      </Modal>
    </div>
  );
}
