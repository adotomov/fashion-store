import { useState } from "react";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { cn } from "../../lib/utils/cn";
import { Checkbox } from "../ui/Checkbox";
import { type ColorOption, ColorSwatch } from "../ui/ColorSwatch";
import { Icon } from "../ui/Icon";
import { Text } from "../ui/Text";

export type FilterOption = {
  id: string;
  label: string;
  count?: number;
  /** Nesting level (0 = top-level). Used to indent subcategories. */
  depth?: number;
};

// Color filter options may carry an explicit id so the panel can toggle by a
// stable value id (e.g. an attribute_value_id) rather than the color's name.
export type FilterColorOption = ColorOption & { id?: string };

// A node in a hierarchical (tree) filter group — e.g. Type → Category →
// Subcategory. Each level is independently selectable and, when it has
// children, collapsible.
export type FilterTreeNode = {
  id: string;
  label: string;
  children?: FilterTreeNode[];
};

export type FilterGroup =
  | { id: string; label: string; type: "checkbox"; options: FilterOption[] }
  | { id: string; label: string; type: "color"; options: FilterColorOption[] }
  | { id: string; label: string; type: "tree"; nodes: FilterTreeNode[] };

type FilterPanelProps = {
  groups: FilterGroup[];
  /** group id -> selected option ids (or color names) */
  selected: Record<string, string[]>;
  onToggle: (groupId: string, optionId: string) => void;
  onClear?: () => void;
  className?: string;
};

// Recursive renderer for a tree filter group. Nodes with children get a
// chevron toggle that expands/collapses their subtree; every node carries a
// checkbox. Collapsed state is tracked by the parent panel keyed on node id.
function TreeNodes({
  nodes,
  groupId,
  selected,
  onToggle,
  collapsed,
  onToggleCollapse,
  depth = 0,
}: {
  nodes: FilterTreeNode[];
  groupId: string;
  selected: string[];
  onToggle: (groupId: string, optionId: string) => void;
  collapsed: Set<string>;
  onToggleCollapse: (nodeId: string) => void;
  depth?: number;
}) {
  return (
    <ul className="flex flex-col gap-2.5">
      {nodes.map((node) => {
        const hasChildren = (node.children?.length ?? 0) > 0;
        const isOpen = hasChildren && !collapsed.has(node.id);
        return (
          <li key={node.id}>
            <div
              className={cn("flex items-center gap-1", depth === 0 && "font-medium")}
              style={depth ? { paddingLeft: `${depth * 1}rem` } : undefined}
            >
              {hasChildren ? (
                <button
                  type="button"
                  onClick={() => onToggleCollapse(node.id)}
                  aria-label={isOpen ? "Collapse" : "Expand"}
                  aria-expanded={isOpen}
                  className="-ml-1 flex h-5 w-5 shrink-0 items-center justify-center rounded-sm text-stone-400 hover:text-stone-700"
                >
                  <Icon name="chevronRight" size={14} className={cn("transition-transform", isOpen && "rotate-90")} />
                </button>
              ) : (
                <span className="w-4 shrink-0" />
              )}
              <Checkbox
                id={`${groupId}-${node.id}`}
                checked={selected.includes(node.id)}
                onChange={() => onToggle(groupId, node.id)}
                label={node.label}
              />
            </div>
            {isOpen && node.children && (
              <div className="mt-2.5">
                <TreeNodes
                  nodes={node.children}
                  groupId={groupId}
                  selected={selected}
                  onToggle={onToggle}
                  collapsed={collapsed}
                  onToggleCollapse={onToggleCollapse}
                  depth={depth + 1}
                />
              </div>
            )}
          </li>
        );
      })}
    </ul>
  );
}

export function FilterPanel({ groups, selected, onToggle, onClear, className }: FilterPanelProps) {
  const { t } = useLanguage();
  const hasActiveFilters = Object.values(selected).some((ids) => ids.length > 0);
  // Collapsed tree-node ids, shared across every tree group in this panel.
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const toggleCollapse = (nodeId: string) =>
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(nodeId)) next.delete(nodeId);
      else next.add(nodeId);
      return next;
    });

  return (
    <div className={cn("flex flex-col gap-1", className)}>
      <div className="mb-2 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Icon name="filters" size={16} className="text-stone-500" />
          <Text size="sm" className="font-medium">
            {t("shop.filters", "Filters")}
          </Text>
        </div>
        {hasActiveFilters && (
          <button type="button" onClick={onClear} className="text-xs text-stone-500 underline hover:text-stone-900">
            {t("shop.clear_filters", "Clear all")}
          </button>
        )}
      </div>

      {groups.map((group) => (
        <details key={group.id} open className="group border-b border-stone-200 py-4 first:pt-0">
          <summary className="flex cursor-pointer list-none items-center justify-between text-sm font-medium text-stone-900">
            {group.label}
            <Icon
              name="chevronDown"
              size={16}
              className="text-stone-400 transition-transform group-open:rotate-180"
            />
          </summary>

          <div className="mt-3">
            {group.type === "checkbox" && (
              <ul className="flex flex-col gap-2.5">
                {group.options.map((option) => (
                  <li key={option.id} style={option.depth ? { paddingLeft: `${option.depth * 1.25}rem` } : undefined}>
                    <Checkbox
                      id={`${group.id}-${option.id}`}
                      checked={selected[group.id]?.includes(option.id) ?? false}
                      onChange={() => onToggle(group.id, option.id)}
                      label={option.count !== undefined ? `${option.label} (${option.count})` : option.label}
                    />
                  </li>
                ))}
              </ul>
            )}

            {group.type === "tree" && (
              <TreeNodes
                nodes={group.nodes}
                groupId={group.id}
                selected={selected[group.id] ?? []}
                onToggle={onToggle}
                collapsed={collapsed}
                onToggleCollapse={toggleCollapse}
              />
            )}

            {group.type === "color" && (
              <div className="flex flex-wrap gap-3">
                {group.options.map((color) => {
                  const key = color.id ?? color.name;
                  return (
                    <ColorSwatch
                      key={key}
                      color={color}
                      selected={selected[group.id]?.includes(key) ?? false}
                      onSelect={() => onToggle(group.id, key)}
                    />
                  );
                })}
              </div>
            )}
          </div>
        </details>
      ))}
    </div>
  );
}
