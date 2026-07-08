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
};

// Color filter options may carry an explicit id so the panel can toggle by a
// stable value id (e.g. an attribute_value_id) rather than the color's name.
export type FilterColorOption = ColorOption & { id?: string };

export type FilterGroup =
  | { id: string; label: string; type: "checkbox"; options: FilterOption[] }
  | { id: string; label: string; type: "color"; options: FilterColorOption[] };

type FilterPanelProps = {
  groups: FilterGroup[];
  /** group id -> selected option ids (or color names) */
  selected: Record<string, string[]>;
  onToggle: (groupId: string, optionId: string) => void;
  onClear?: () => void;
  className?: string;
};

export function FilterPanel({ groups, selected, onToggle, onClear, className }: FilterPanelProps) {
  const { t } = useLanguage();
  const hasActiveFilters = Object.values(selected).some((ids) => ids.length > 0);

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
        <details key={group.id} open className="border-b border-stone-200 py-4 first:pt-0">
          <summary className="cursor-pointer list-none text-sm font-medium text-stone-900">
            {group.label}
          </summary>

          <div className="mt-3">
            {group.type === "checkbox" && (
              <ul className="flex flex-col gap-2.5">
                {group.options.map((option) => (
                  <li key={option.id}>
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
