import { useEffect, useState } from "react";

import { useAdminPermissions } from "../../../features/admin/AdminPermissionsContext";

import { EmptyState } from "../EmptyState";
import { Tabs } from "../Tabs";
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

const subTabs = [
  { id: "default", label: "Default" },
  { id: "custom", label: "Custom" },
];

export function AttributesTab() {
  const { isReadOnly } = useAdminPermissions();
  const [attributes, setAttributes] = useState<Attribute[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [subTab, setSubTab] = useState<string>("default");

  // Create-attribute modal (custom attributes only).
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

  async function handleAddColor(attribute: Attribute, name: string, hex: string) {
    try {
      await addAttributeValue(attribute.id, name, hex);
      await refresh();
    } catch {
      setError("Could not add color. The name may already exist.");
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

  const defaultAttributes = (attributes ?? []).filter((a) => a.is_system);
  const customAttributes = (attributes ?? []).filter((a) => !a.is_system);

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-4">
        <Tabs tabs={subTabs} activeTab={subTab} onChange={setSubTab} />
        {subTab === "custom" && (
          <Button variant="primary" onClick={openCreateModal} disabled={isReadOnly} className="shrink-0">
            <Icon name="plus" size={16} />
            Create
          </Button>
        )}
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
      ) : subTab === "default" ? (
        defaultAttributes.length === 0 ? (
          <EmptyState
            icon="catalog"
            title="No default attributes"
            description="Built-in attributes like Color will appear here."
          />
        ) : (
          <div className="grid grid-cols-1 gap-4">
            {defaultAttributes.map((attribute) =>
              attribute.type === "color" ? (
                <ColorAttributeCard
                  key={attribute.id}
                  attribute={attribute}
                  isReadOnly={isReadOnly}
                  onAddColor={handleAddColor}
                  onDeleteValue={handleDeleteValue}
                />
              ) : (
                <TextAttributeCard
                  key={attribute.id}
                  attribute={attribute}
                  isReadOnly={isReadOnly}
                  deletable={false}
                  newValue={newValueByAttribute[attribute.id] ?? ""}
                  onNewValueChange={(v) =>
                    setNewValueByAttribute((prev) => ({ ...prev, [attribute.id]: v }))
                  }
                  onAddValue={() => handleAddValue(attribute)}
                  onDeleteValue={handleDeleteValue}
                  onDeleteAttribute={handleDeleteAttribute}
                />
              ),
            )}
          </div>
        )
      ) : customAttributes.length === 0 ? (
        <EmptyState
          icon="catalog"
          title="No custom attributes yet"
          description="Create attributes like Size or Material, then add their values."
        />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {customAttributes.map((attribute) => (
            <TextAttributeCard
              key={attribute.id}
              attribute={attribute}
              isReadOnly={isReadOnly}
              deletable
              newValue={newValueByAttribute[attribute.id] ?? ""}
              onNewValueChange={(v) => setNewValueByAttribute((prev) => ({ ...prev, [attribute.id]: v }))}
              onAddValue={() => handleAddValue(attribute)}
              onDeleteValue={handleDeleteValue}
              onDeleteAttribute={handleDeleteAttribute}
            />
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

type TextAttributeCardProps = {
  attribute: Attribute;
  isReadOnly: boolean;
  deletable: boolean;
  newValue: string;
  onNewValueChange: (value: string) => void;
  onAddValue: () => void;
  onDeleteValue: (attribute: Attribute, valueId: string) => void;
  onDeleteAttribute: (attribute: Attribute) => void;
};

function TextAttributeCard({
  attribute,
  isReadOnly,
  deletable,
  newValue,
  onNewValueChange,
  onAddValue,
  onDeleteValue,
  onDeleteAttribute,
}: TextAttributeCardProps) {
  return (
    <Card className="p-5">
      <div className="flex items-center justify-between">
        <Text className="font-medium">{attribute.name}</Text>
        {deletable && (
          <Button
            variant="ghost"
            size="sm"
            aria-label="Delete attribute"
            title="Delete attribute"
            onClick={() => onDeleteAttribute(attribute)}
            disabled={isReadOnly}
            className="text-danger-600 hover:bg-danger-50"
          >
            <Icon name="trash" size={15} />
          </Button>
        )}
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
              onClick={() => onDeleteValue(attribute, value.id)}
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
          value={newValue}
          onChange={(e) => onNewValueChange(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") onAddValue();
          }}
          disabled={isReadOnly}
          className="h-9 text-sm"
        />
        <Button variant="outline" size="sm" onClick={onAddValue} disabled={isReadOnly}>
          Add
        </Button>
      </div>

      <div className="mt-3">
        <TranslationFields entityType="attribute" entityId={attribute.id} fields={[{ key: "name", label: "Name" }]} />
      </div>
    </Card>
  );
}

type ColorAttributeCardProps = {
  attribute: Attribute;
  isReadOnly: boolean;
  onAddColor: (attribute: Attribute, name: string, hex: string) => void;
  onDeleteValue: (attribute: Attribute, valueId: string) => void;
};

function ColorAttributeCard({ attribute, isReadOnly, onAddColor, onDeleteValue }: ColorAttributeCardProps) {
  const [hex, setHex] = useState("#B2543C");
  const [name, setName] = useState("");

  function handleAdd() {
    if (!name.trim()) return;
    onAddColor(attribute, name.trim(), hex);
    setName("");
  }

  return (
    <Card className="p-5">
      <div className="flex items-center gap-2">
        <Text className="font-medium">{attribute.name}</Text>
        <Badge variant="neutral">Default</Badge>
      </div>
      <Text size="xs" tone="muted" className="mt-1">
        Pick a color from the palette and give it a name. Colors show as swatches on the storefront.
      </Text>

      <div className="mt-4 flex flex-wrap gap-4">
        {attribute.values.length === 0 && (
          <Text size="xs" tone="muted">
            No colors yet
          </Text>
        )}
        {attribute.values.map((value) => (
          <div key={value.id} className="flex flex-col items-center gap-1.5">
            <div className="relative">
              <span
                className="block h-9 w-9 rounded-full border border-stone-300"
                style={{ backgroundColor: value.color_hex ?? "transparent" }}
                title={value.color_hex ?? value.value}
              />
              <button
                type="button"
                aria-label={`Remove ${value.value}`}
                onClick={() => onDeleteValue(attribute, value.id)}
                disabled={isReadOnly}
                className="absolute -right-1.5 -top-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-white text-stone-400 shadow-sm ring-1 ring-stone-200 hover:text-danger-600 disabled:pointer-events-none disabled:opacity-30"
              >
                <Icon name="close" size={10} />
              </button>
            </div>
            <Text size="xs" tone="muted">
              {value.value}
            </Text>
          </div>
        ))}
      </div>

      <div className="mt-5 flex flex-wrap items-end gap-3 border-t border-stone-100 pt-4">
        <div>
          <Text size="xs" tone="muted" className="mb-1">
            Color
          </Text>
          <input
            type="color"
            value={hex}
            onChange={(e) => setHex(e.target.value)}
            disabled={isReadOnly}
            aria-label="Pick a color"
            className="h-9 w-12 cursor-pointer rounded-sm border border-stone-300 bg-white p-1 disabled:cursor-not-allowed"
          />
        </div>
        <div className="flex-1 min-w-40">
          <Text size="xs" tone="muted" className="mb-1">
            Name
          </Text>
          <Input
            placeholder="e.g. Clay"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") handleAdd();
            }}
            disabled={isReadOnly}
            className="h-9 text-sm"
          />
        </div>
        <Button variant="outline" size="sm" onClick={handleAdd} disabled={isReadOnly || !name.trim()}>
          Add Color
        </Button>
      </div>

      <div className="mt-3">
        <TranslationFields entityType="attribute" entityId={attribute.id} fields={[{ key: "name", label: "Name" }]} />
      </div>
    </Card>
  );
}
