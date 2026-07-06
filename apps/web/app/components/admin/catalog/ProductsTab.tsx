import { useEffect, useState } from "react";
import { useNavigate } from "react-router";

import { useAdminPermissions } from "../../../features/admin/AdminPermissionsContext";

import { EmptyState } from "../EmptyState";
import { Badge } from "../../ui/Badge";
import { Button } from "../../ui/Button";
import { FormField } from "../../ui/FormField";
import { Icon } from "../../ui/Icon";
import { Input } from "../../ui/Input";
import { Modal } from "../../ui/Modal";
import { Price } from "../../ui/Price";
import { Text } from "../../ui/Text";
import { type Product, createProduct, deleteProduct, listProducts } from "../../../lib/api/products";

const dateFormatter = new Intl.DateTimeFormat("en-US", { dateStyle: "medium", timeStyle: "short" });

function formatDate(value: string): string {
  return dateFormatter.format(new Date(value));
}

const statusVariant = {
  draft: "neutral",
  active: "success",
  archived: "danger",
} as const;

export function ProductsTab() {
  const { isReadOnly } = useAdminPermissions();
  const navigate = useNavigate();
  const [products, setProducts] = useState<Product[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  async function refresh() {
    try {
      setProducts(await listProducts());
    } catch {
      setError("Could not load products.");
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
      const product = await createProduct(newName.trim());
      setIsModalOpen(false);
      navigate(`/admin/products/${product.id}`);
    } catch {
      setSaveError("Could not create product. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete(product: Product, e: React.MouseEvent) {
    e.stopPropagation();
    if (!window.confirm(`Delete product "${product.name}"? This cannot be undone.`)) {
      return;
    }
    try {
      await deleteProduct(product.id);
      await refresh();
    } catch {
      setError("Could not delete product.");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-end">
        <Button variant="primary" onClick={openCreateModal} disabled={isReadOnly}>
          <Icon name="plus" size={16} />
          Create
        </Button>
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {products === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : products.length === 0 ? (
        <EmptyState icon="catalog" title="No products yet" description="Create your first product to get started." />
      ) : (
        <div className="overflow-hidden rounded-sm border border-stone-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
              <tr>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Price</th>
                <th className="px-4 py-3 font-medium">Variants</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {products.map((product) => (
                <tr
                  key={product.id}
                  onClick={() => navigate(`/admin/products/${product.id}`)}
                  className="cursor-pointer border-b border-stone-100 last:border-0 hover:bg-stone-50"
                >
                  <td className="px-4 py-3 font-medium text-stone-900">{product.name}</td>
                  <td className="px-4 py-3">
                    <Badge variant={statusVariant[product.status]}>{product.status}</Badge>
                  </td>
                  <td className="px-4 py-3">
                    <Price price={product.base_price} size="sm" />
                  </td>
                  <td className="px-4 py-3 text-stone-600">{product.variant_count}</td>
                  <td className="px-4 py-3 text-stone-600">{formatDate(product.created_at)}</td>
                  <td className="px-4 py-3">
                    <Button
                      variant="ghost"
                      size="sm"
                      aria-label="Delete product"
                      title="Delete product"
                      onClick={(e) => handleDelete(product, e)}
                      disabled={isReadOnly}
                      className="text-danger-600 hover:bg-danger-50"
                    >
                      <Icon name="trash" size={15} />
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <Modal open={isModalOpen} onClose={() => setIsModalOpen(false)} title="Create Product">
        <FormField label="Name" htmlFor="product-name" error={saveError ?? undefined}>
          <Input
            id="product-name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Silk Wrap Dress"
            autoFocus
          />
        </FormField>

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setIsModalOpen(false)} disabled={isSaving}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={isSaving || isReadOnly}>
            {isSaving ? "Saving…" : "Save & Continue"}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
