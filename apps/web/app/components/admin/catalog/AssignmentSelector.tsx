import { useState } from "react";

import { Badge } from "../../ui/Badge";
import { Button } from "../../ui/Button";
import { Icon } from "../../ui/Icon";
import { Select } from "../../ui/Select";
import { Text } from "../../ui/Text";

type Option = {
  id: string;
  label: string;
};

type AssignmentSelectorProps = {
  options: Option[];
  selectedIds: string[];
  onAdd: (id: string) => void;
  onRemove: (id: string) => void;
  placeholder: string;
  emptyMessage: string;
};

// Generic "pick from a dropdown of not-yet-assigned options, show assigned
// ones as removable badges" pattern, used for both category and catalog
// assignment on the product editor.
export function AssignmentSelector({
  options,
  selectedIds,
  onAdd,
  onRemove,
  placeholder,
  emptyMessage,
}: AssignmentSelectorProps) {
  const [pendingId, setPendingId] = useState("");

  const selected = options.filter((o) => selectedIds.includes(o.id));
  const available = options.filter((o) => !selectedIds.includes(o.id));

  function handleAdd() {
    if (!pendingId) return;
    onAdd(pendingId);
    setPendingId("");
  }

  return (
    <div className="flex flex-col gap-3">
      {selected.length === 0 ? (
        <Text size="sm" tone="muted">
          {emptyMessage}
        </Text>
      ) : (
        <div className="flex flex-wrap gap-2">
          {selected.map((option) => (
            <Badge key={option.id} variant="brand" className="gap-1.5">
              {option.label}
              <button
                type="button"
                aria-label={`Remove ${option.label}`}
                onClick={() => onRemove(option.id)}
                className="text-stone-600 hover:text-danger-600"
              >
                <Icon name="close" size={12} />
              </button>
            </Badge>
          ))}
        </div>
      )}

      {available.length > 0 && (
        <div className="flex gap-2">
          <Select value={pendingId} onChange={(e) => setPendingId(e.target.value)} className="h-9 max-w-xs text-sm">
            <option value="">{placeholder}</option>
            {available.map((option) => (
              <option key={option.id} value={option.id}>
                {option.label}
              </option>
            ))}
          </Select>
          <Button variant="outline" size="sm" onClick={handleAdd} disabled={!pendingId}>
            Add
          </Button>
        </div>
      )}
    </div>
  );
}
