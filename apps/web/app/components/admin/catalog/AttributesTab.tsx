import { useEffect, useState } from "react";

import { useAdminPermissions } from "../../../features/admin/AdminPermissionsContext";

import { EmptyState } from "../EmptyState";
import { TranslationFields } from "../TranslationFields";
import { Badge } from "../../ui/Badge";
import { Button } from "../../ui/Button";
import { Card } from "../../ui/Card";
import { FormField } from "../../ui/FormField";
import { Icon } from "../../ui/Icon";
import { Input } from "../../ui/Input";
import { Modal } from "../../ui/Modal";
import { Text } from "../../ui/Text";
import {
  type Attribute,
  addAttributeValue,
  createAttribute,
  deleteAttribute,
  deleteAttributeValue,
  listAttributes,
} from "../../../lib/api/attributes";

export function AttributesTab() {
  const { isReadOnly } = useAdminPermissions();
  const [attributes, setAttributes] = useState<Attribute[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [newName, setNewName] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [newValueByAttribute, setNewValueByAttribute] = useState<Record<string, string>>({});

  async function refresh() {
    try {
      setAttributes(await listAttributes());
    } catch {
      setError("Could not load attributes.");
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
      await createAttribute(newName.trim());
      setIsModalOpen(false);
      await refresh();
    } catch {
      setSaveError("Could not create attribute. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDeleteAttribute(attribute: Attribute) {
    if (!window.confirm(`Delete attribute "${attribute.name}" and all its values?`)) {
      return;
    }
    try {
      await deleteAttribute(attribute.id);
      await refresh();
    } catch {
      setError("Could not delete attribute.");
    }
  }

  async function handleAddValue(attribute: Attribute) {
    const value = newValueByAttribute[attribute.id]?.trim();
    if (!value) return;
    try {
      await addAttributeValue(attribute.id, value);
      setNewValueByAttribute((prev) => ({ ...prev, [attribute.id]: "" }));
      await refresh();
    } catch {
      setError("Could not add value. It may already exist.");
    }
  }

  async function handleDeleteValue(attribute: Attribute, valueId: string) {
    try {
      await deleteAttributeValue(attribute.id, valueId);
      await refresh();
    } catch {
      setError("Could not delete value.");
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

      {attributes === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : attributes.length === 0 ? (
        <EmptyState
          icon="catalog"
          title="No attributes yet"
          description="Create attributes like Size or Color, then add their values."
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {attributes.map((attribute) => (
            <Card key={attribute.id} className="p-5">
              <div className="flex items-center justify-between">
                <Text className="font-medium">{attribute.name}</Text>
                <Button
                  variant="ghost"
                  size="sm"
                  aria-label="Delete attribute"
                  title="Delete attribute"
                  onClick={() => handleDeleteAttribute(attribute)}
                  disabled={isReadOnly}
                  className="text-danger-600 hover:bg-danger-50"
                >
                  <Icon name="trash" size={15} />
                </Button>
              </div>

              <div className="mt-3 flex flex-wrap gap-2">
                {attribute.values.length === 0 && (
                  <Text size="xs" tone="muted">
                    No values yet
                  </Text>
                )}
                {attribute.values.map((value) => (
                  <Badge key={value.id} variant="neutral" className="gap-1.5">
                    {value.value}
                    <button
                      type="button"
                      aria-label={`Remove ${value.value}`}
                      onClick={() => handleDeleteValue(attribute, value.id)}
                      disabled={isReadOnly}
                      className="text-stone-400 hover:text-danger-600 disabled:pointer-events-none disabled:opacity-30"
                    >
                      <Icon name="close" size={12} />
                    </button>
                  </Badge>
                ))}
              </div>

              <div className="mt-3 flex gap-2">
                <Input
                  placeholder="Add a value"
                  value={newValueByAttribute[attribute.id] ?? ""}
                  onChange={(e) =>
                    setNewValueByAttribute((prev) => ({ ...prev, [attribute.id]: e.target.value }))
                  }
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleAddValue(attribute);
                  }}
                  disabled={isReadOnly}
                  className="h-9 text-sm"
                />
                <Button variant="outline" size="sm" onClick={() => handleAddValue(attribute)} disabled={isReadOnly}>
                  Add
                </Button>
              </div>

              <div className="mt-3">
                <TranslationFields entityType="attribute" entityId={attribute.id} fields={[{ key: "name", label: "Name" }]} />
              </div>
            </Card>
          ))}
        </div>
      )}

      <Modal open={isModalOpen} onClose={() => setIsModalOpen(false)} title="Create Attribute">
        <FormField label="Name" htmlFor="attribute-name" error={saveError ?? undefined}>
          <Input
            id="attribute-name"
            value={newName}
            onChange={(e) => setNewName(e.target.value)}
            placeholder="Size"
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
    </div>
  );
}
