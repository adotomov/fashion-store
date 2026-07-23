import { useEffect, useRef, useState } from "react";

import { useAdminPermissions } from "../../../features/admin/AdminPermissionsContext";

import { EmptyState } from "../EmptyState";
import { TranslationFields } from "../TranslationFields";
import { Button } from "../../ui/Button";
import { FormField } from "../../ui/FormField";
import { Icon } from "../../ui/Icon";
import { Input } from "../../ui/Input";
import { Modal } from "../../ui/Modal";
import { Pagination } from "../../ui/Pagination";
import { Select } from "../../ui/Select";
import { Text } from "../../ui/Text";
import { usePagination } from "../../../lib/usePagination";
import {
  type Category,
  createCategory,
  deleteCategory,
  deleteCategoryThumbnail,
  listCategories,
  loadCategoryThumbnailBlobUrl,
  updateCategory,
  uploadCategoryThumbnail,
} from "../../../lib/api/categories";
import { type ProductType, listProductTypes } from "../../../lib/api/product-types";

const PAGE_SIZE = 20;

export function CategoriesTab() {
  const { isReadOnly } = useAdminPermissions();
  const [categories, setCategories] = useState<Category[] | null>(null);
  const [productTypes, setProductTypes] = useState<ProductType[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [newName, setNewName] = useState("");
  const [newParentId, setNewParentId] = useState("");
  const [newProductTypeId, setNewProductTypeId] = useState("");
  const [newInternalIdentifier, setNewInternalIdentifier] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const [thumbnailPreview, setThumbnailPreview] = useState<string | null>(null);
  const [isThumbnailBusy, setIsThumbnailBusy] = useState(false);
  const [thumbnailError, setThumbnailError] = useState<string | null>(null);
  const thumbnailInputRef = useRef<HTMLInputElement>(null);
  const { page, totalPages, pageItems, setPage } = usePagination(categories ?? [], PAGE_SIZE);

  async function refresh() {
    try {
      setCategories(await listCategories());
    } catch {
      setError("Could not load categories.");
    }
  }

  useEffect(() => {
    refresh();
    listProductTypes().then(setProductTypes).catch(() => {});
  }, []);

  useEffect(() => {
    if (!editingCategory?.image_url) {
      setThumbnailPreview(null);
      return;
    }
    let cancelled = false;
    let url: string | null = null;
    loadCategoryThumbnailBlobUrl(editingCategory.id)
      .then((loaded) => {
        if (cancelled) return;
        url = loaded;
        setThumbnailPreview(loaded);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
      if (url) URL.revokeObjectURL(url);
    };
  }, [editingCategory?.id, editingCategory?.image_url]);

  function openCreateModal() {
    setEditingCategory(null);
    setNewName("");
    setNewParentId("");
    setNewProductTypeId("");
    setNewInternalIdentifier("");
    setSaveError(null);
    setThumbnailError(null);
    setIsModalOpen(true);
  }

  function openEditModal(category: Category) {
    setEditingCategory(category);
    setNewName(category.name);
    setNewParentId(category.parent_id ?? "");
    setNewProductTypeId(category.product_type_id);
    setNewInternalIdentifier(category.internal_identifier ?? "");
    setSaveError(null);
    setThumbnailError(null);
    setIsModalOpen(true);
  }

  async function handleSave() {
    if (!newName.trim()) {
      setSaveError("Name is required");
      return;
    }
    if (!newProductTypeId) {
      setSaveError("Product type is required");
      return;
    }
    if (!newInternalIdentifier.trim()) {
      setSaveError("Internal identifier is required");
      return;
    }
    setIsSaving(true);
    setSaveError(null);
    try {
      if (editingCategory) {
        await updateCategory(editingCategory.id, {
          name: newName.trim(),
          parent_id: newParentId || null,
          product_type_id: newProductTypeId,
          internal_identifier: newInternalIdentifier.trim(),
        });
      } else {
        await createCategory(
          newName.trim(),
          newProductTypeId,
          newParentId || undefined,
          newInternalIdentifier.trim(),
        );
      }
      setIsModalOpen(false);
      await refresh();
    } catch {
      setSaveError(editingCategory ? "Could not save changes. Try again." : "Could not create category. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleThumbnailSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file || !editingCategory) return;
    setIsThumbnailBusy(true);
    setThumbnailError(null);
    try {
      const updated = await uploadCategoryThumbnail(editingCategory.id, file);
      setEditingCategory(updated);
      await refresh();
    } catch {
      setThumbnailError("Could not upload image.");
    } finally {
      setIsThumbnailBusy(false);
      if (thumbnailInputRef.current) thumbnailInputRef.current.value = "";
    }
  }

  async function handleThumbnailRemove() {
    if (!editingCategory) return;
    setIsThumbnailBusy(true);
    setThumbnailError(null);
    try {
      const updated = await deleteCategoryThumbnail(editingCategory.id);
      setEditingCategory(updated);
      await refresh();
    } catch {
      setThumbnailError("Could not remove image.");
    } finally {
      setIsThumbnailBusy(false);
    }
  }

  async function handleDelete(category: Category) {
    if (!window.confirm(`Delete category "${category.name}"? This cannot be undone.`)) {
      return;
    }
    try {
      await deleteCategory(category.id);
      await refresh();
    } catch {
      setError("Could not delete category. It may have subcategories.");
    }
  }

  function parentName(category: Category): string {
    if (!category.parent_id) return "—";
    return categories?.find((c) => c.id === category.parent_id)?.name ?? "—";
  }

  function productTypeName(category: Category): string {
    return productTypes.find((t) => t.id === category.product_type_id)?.name ?? "—";
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

      {categories === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : categories.length === 0 ? (
        <EmptyState icon="catalog" title="No categories yet" description="Create your first category." />
      ) : (
        <>
        <div className="overflow-hidden rounded-sm border border-stone-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
              <tr>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Identifier</th>
                <th className="px-4 py-3 font-medium">Slug</th>
                <th className="px-4 py-3 font-medium">Type</th>
                <th className="px-4 py-3 font-medium">Parent</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {pageItems.map((category) => (
                <tr key={category.id} className="border-b border-stone-100 last:border-0">
                  <td className="px-4 py-3 font-medium text-stone-900">{category.name}</td>
                  <td className="px-4 py-3 font-mono text-xs text-stone-600">
                    {category.internal_identifier || <span className="text-stone-400">—</span>}
                  </td>
                  <td className="px-4 py-3 font-mono text-xs text-stone-500">{category.slug}</td>
                  <td className="px-4 py-3 text-stone-600">{productTypeName(category)}</td>
                  <td className="px-4 py-3 text-stone-600">{parentName(category)}</td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Edit category"
                        title="Edit category"
                        disabled={isReadOnly}
                        onClick={() => openEditModal(category)}
                      >
                        <Icon name="pencil" size={15} />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Delete category"
                        title="Delete category"
                        onClick={() => handleDelete(category)}
                        disabled={isReadOnly}
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
        <Pagination page={page} totalPages={totalPages} onPageChange={setPage} className="mt-4" />
        </>
      )}

      <Modal
        open={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={editingCategory ? "Edit Category" : "Create Category"}
      >
        <div className="flex flex-col gap-4">
          <FormField label="Name" htmlFor="category-name" error={saveError ?? undefined}>
            <Input
              id="category-name"
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              placeholder="Dresses"
              autoFocus
            />
          </FormField>
          <FormField
            label="Internal identifier"
            htmlFor="category-identifier"
            hint="Required — used as the SKU prefix for this category's products (e.g. DR-01)"
          >
            <Input
              id="category-identifier"
              value={newInternalIdentifier}
              onChange={(e) => setNewInternalIdentifier(e.target.value)}
              placeholder="DR-01"
            />
          </FormField>
          <FormField label="Product type" htmlFor="category-product-type">
            <Select
              id="category-product-type"
              value={newProductTypeId}
              onChange={(e) => setNewProductTypeId(e.target.value)}
            >
              <option value="">Choose a type…</option>
              {productTypes.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </Select>
          </FormField>
          <FormField label="Parent category" htmlFor="category-parent" hint="Optional">
            <Select id="category-parent" value={newParentId} onChange={(e) => setNewParentId(e.target.value)}>
              <option value="">None</option>
              {categories
                ?.filter((c) => c.id !== editingCategory?.id)
                .map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
            </Select>
          </FormField>
          <FormField
            label="Thumbnail"
            htmlFor="category-thumbnail"
            hint={
              editingCategory
                ? "Optional — shown as the small photo in the storefront nav menu"
                : "Save the category first, then come back to upload a thumbnail"
            }
            error={thumbnailError ?? undefined}
          >
            {editingCategory ? (
              <div className="flex items-center gap-3">
                <div className="flex h-16 w-16 items-center justify-center overflow-hidden rounded-sm border border-stone-200 bg-stone-50">
                  {thumbnailPreview ? (
                    <img src={thumbnailPreview} alt="" className="h-full w-full object-cover" />
                  ) : (
                    <Icon name="catalog" size={18} className="text-stone-400" />
                  )}
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    type="button"
                    disabled={isThumbnailBusy || isReadOnly}
                    onClick={() => thumbnailInputRef.current?.click()}
                  >
                    {isThumbnailBusy ? "Uploading…" : thumbnailPreview ? "Replace" : "Upload"}
                  </Button>
                  {thumbnailPreview && (
                    <Button
                      variant="ghost"
                      size="sm"
                      type="button"
                      disabled={isThumbnailBusy || isReadOnly}
                      onClick={handleThumbnailRemove}
                      className="text-danger-600 hover:bg-danger-50"
                    >
                      Remove
                    </Button>
                  )}
                </div>
                <input
                  ref={thumbnailInputRef}
                  id="category-thumbnail"
                  type="file"
                  accept="image/*"
                  onChange={handleThumbnailSelected}
                  disabled={isThumbnailBusy}
                  className="hidden"
                />
              </div>
            ) : (
              <Text size="sm" tone="muted">
                Not available yet
              </Text>
            )}
          </FormField>

          <TranslationFields entityType="category" entityId={editingCategory?.id} fields={[{ key: "name", label: "Name" }]} />
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setIsModalOpen(false)} disabled={isSaving}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={isSaving || isReadOnly}>
            {isSaving ? "Saving…" : "Save"}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
