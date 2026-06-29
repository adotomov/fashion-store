import { useEffect, useState } from "react";

import { EmptyState } from "../EmptyState";
import { TranslationFields } from "../TranslationFields";
import { Button } from "../../ui/Button";
import { FormField } from "../../ui/FormField";
import { Icon } from "../../ui/Icon";
import { Input } from "../../ui/Input";
import { Modal } from "../../ui/Modal";
import { Text } from "../../ui/Text";
import {
  type ProductType,
  createProductType,
  deleteProductType,
  listProductTypes,
} from "../../../lib/api/product-types";

export function ProductTypesTab() {
  const [productTypes, setProductTypes] = useState<ProductType[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [translatingType, setTranslatingType] = useState<ProductType | null>(null);

  async function refresh() {
    try {
      setProductTypes(await listProductTypes());
    } catch {
      setError("Could not load product types.");
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  function openCreateModal() {
    setNewName("");
    setSaveError(null);
    setIsModalOpen(true);
  }

  async function handleSave() {
    if (!newName.trim()) {
      setSaveError("Name is required");
      return;
    }
    setIsSaving(true);
    setSaveError(null);
    try {
      await createProductType(newName.trim());
      setIsModalOpen(false);
      await refresh();
    } catch {
      setSaveError("Could not create product type. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete(productType: ProductType) {
    if (!window.confirm(`Delete product type "${productType.name}"? This cannot be undone.`)) {
      return;
    }
    try {
      await deleteProductType(productType.id);
      await refresh();
    } catch {
      setError("Could not delete product type. It may still have categories assigned to it.");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <Text size="xs" tone="muted">
        Product types are the top-level storefront navigation menu (e.g. Jewellery, Clothing). Categories are
        assigned to a type and appear as its dropdown items.
      </Text>

      <div className="flex items-center justify-end">
        <Button variant="primary" onClick={openCreateModal}>
          <Icon name="plus" size={16} />
          Create
        </Button>
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {productTypes === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : productTypes.length === 0 ? (
        <EmptyState icon="catalog" title="No product types yet" description="Create your first product type." />
      ) : (
        <div className="overflow-hidden rounded-sm border border-stone-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
              <tr>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Slug</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {productTypes.map((productType) => (
                <tr key={productType.id} className="border-b border-stone-100 last:border-0">
                  <td className="px-4 py-3 font-medium text-stone-900">{productType.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-stone-500">{productType.slug}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Translate product type"
                        title="Translate"
                        onClick={() => setTranslatingType(productType)}
                      >
                        <Icon name="globe" size={15} />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Delete product type"
                        title="Delete product type"
                        onClick={() => handleDelete(productType)}
                        className="text-danger-600 hover:bg-danger-50"
                      >
                        <Icon name="trash" size={15} />
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal open={isModalOpen} onClose={() => setIsModalOpen(false)} title="Create Product Type">
        <FormField label="Name" htmlFor="product-type-name" error={saveError ?? undefined}>
          <Input
            id="product-type-name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Jewellery"
            autoFocus
          />
        </FormField>

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setIsModalOpen(false)} disabled={isSaving}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={isSaving}>
            {isSaving ? "Saving…" : "Save"}
          </Button>
        </div>
      </Modal>

      <Modal
        open={translatingType !== null}
        onClose={() => setTranslatingType(null)}
        title={`Translate "${translatingType?.name ?? ""}"`}
      >
        {translatingType && (
          <TranslationFields entityType="product_type" entityId={translatingType.id} fields={[{ key: "name", label: "Name" }]} />
        )}
      </Modal>
    </div>
  );
}
