import { useEffect, useState } from "react";

import { useAdminPermissions } from "../../../features/admin/AdminPermissionsContext";

import { EmptyState } from "../EmptyState";
import { TranslationFields } from "../TranslationFields";
import { Button } from "../../ui/Button";
import { FormField } from "../../ui/FormField";
import { Icon } from "../../ui/Icon";
import { Input } from "../../ui/Input";
import { Modal } from "../../ui/Modal";
import { Select } from "../../ui/Select";
import { Text } from "../../ui/Text";
import {
  type Catalog,
  type CatalogStatus,
  createCatalog,
  deleteCatalog,
  downloadCatalogExport,
  listCatalogs,
  updateCatalog,
} from "../../../lib/api/catalogs";

const dateFormatter = new Intl.DateTimeFormat("en-US", { dateStyle: "medium", timeStyle: "short" });

function formatDate(value: string): string {
  return dateFormatter.format(new Date(value));
}

export function CatalogsTab() {
  const { isReadOnly } = useAdminPermissions();
  const [catalogs, setCatalogs] = useState<Catalog[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [translatingCatalog, setTranslatingCatalog] = useState<Catalog | null>(null);

  async function refresh() {
    try {
      const data = await listCatalogs();
      setCatalogs(data);
    } catch {
      setError("Could not load catalogs.");
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
      await createCatalog(newName.trim());
      setIsModalOpen(false);
      await refresh();
    } catch {
      setSaveError("Could not create catalog. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleStatusChange(catalog: Catalog, status: CatalogStatus) {
    setCatalogs((prev) => prev?.map((c) => (c.id === catalog.id ? { ...c, status } : c)) ?? prev);
    try {
      await updateCatalog(catalog.id, { status });
    } catch {
      setError("Could not update status.");
      await refresh();
    }
  }

  async function handleDelete(catalog: Catalog) {
    if (!window.confirm(`Delete catalog "${catalog.name}"? This cannot be undone.`)) {
      return;
    }
    try {
      await deleteCatalog(catalog.id);
      await refresh();
    } catch {
      setError("Could not delete catalog.");
    }
  }

  async function handleExport(catalog: Catalog, format: "csv" | "json") {
    try {
      await downloadCatalogExport(catalog.id, format);
    } catch {
      setError("Could not export catalog.");
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

      {catalogs === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : catalogs.length === 0 ? (
        <EmptyState
          icon="catalog"
          title="No catalogs yet"
          description="Create your first catalog to start organizing products."
        />
      ) : (
        <div className="overflow-hidden rounded-sm border border-stone-200 bg-white">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
              <tr>
                <th className="px-4 py-3 font-medium">Catalog ID</th>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 font-medium">Last Updated</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {catalogs.map((catalog) => (
                <tr key={catalog.id} className="border-b border-stone-100 last:border-0">
                  <td className="px-4 py-3 font-mono text-xs text-stone-500" title={catalog.id}>
                    {catalog.id.slice(0, 8)}
                  </td>
                  <td className="px-4 py-3 font-medium text-stone-900">{catalog.name}</td>
                  <td className="px-4 py-3 text-stone-600">{formatDate(catalog.created_at)}</td>
                  <td className="px-4 py-3 text-stone-600">{formatDate(catalog.updated_at)}</td>
                  <td className="px-4 py-3">
                    <Select
                      value={catalog.status}
                      disabled={isReadOnly}
                      onChange={(e) => handleStatusChange(catalog, e.target.value as CatalogStatus)}
                      className="h-9 w-32 text-xs"
                    >
                      <option value="draft">Draft</option>
                      <option value="active">Active</option>
                      <option value="disabled">Disabled</option>
                    </Select>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Export CSV"
                        title="Export CSV"
                        onClick={() => handleExport(catalog, "csv")}
                      >
                        CSV
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Export JSON"
                        title="Export JSON"
                        onClick={() => handleExport(catalog, "json")}
                      >
                        JSON
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Translate catalog"
                        title="Translate"
                        disabled={isReadOnly}
                        onClick={() => setTranslatingCatalog(catalog)}
                      >
                        <Icon name="globe" size={15} />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label="Delete catalog"
                        title="Delete catalog"
                        onClick={() => handleDelete(catalog)}
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
      )}

      <Modal open={isModalOpen} onClose={() => setIsModalOpen(false)} title="Create Catalog">
        <FormField label="Name" htmlFor="catalog-name" error={saveError ?? undefined}>
          <Input
            id="catalog-name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Summer Collection"
            autoFocus
          />
        </FormField>

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setIsModalOpen(false)} disabled={isSaving}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={isSaving || isReadOnly}>
            {isSaving ? "Saving…" : "Save"}
          </Button>
        </div>
      </Modal>

      <Modal
        open={translatingCatalog !== null}
        onClose={() => setTranslatingCatalog(null)}
        title={`Translate "${translatingCatalog?.name ?? ""}"`}
      >
        {translatingCatalog && (
          <TranslationFields entityType="catalog" entityId={translatingCatalog.id} fields={[{ key: "name", label: "Name" }]} />
        )}
      </Modal>
    </div>
  );
}
