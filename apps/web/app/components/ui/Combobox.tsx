import { useEffect, useId, useRef, useState } from "react";

import { cn } from "../../lib/utils/cn";
import { Input } from "./Input";

export type ComboboxOption = {
  id: number | string;
  label: string;
  sublabel?: string;
};

type ComboboxProps = {
  id?: string;
  /** Text currently shown in the field (the committed selection's name, or what the user is typing). */
  value: string;
  /** Fired when the user edits the text — the parent should clear any committed id. */
  onValueChange: (text: string) => void;
  /** Async source of suggestions for a query fragment. */
  onSearch: (query: string) => Promise<ComboboxOption[]>;
  /** Fired when the user picks a suggestion. */
  onSelect: (option: ComboboxOption) => void;
  placeholder?: string;
  disabled?: boolean;
  invalid?: boolean;
  /** Minimum characters before searching (default 1). */
  minChars?: number;
  emptyText?: string;
};

// Combobox is an async typeahead: it debounces the query, lists suggestions,
// and reports the chosen option to the parent. The parent owns the committed
// value (e.g. a Speedy siteId + name); this component owns only the open/typing
// UI state.
export function Combobox({
  id,
  value,
  onValueChange,
  onSearch,
  onSelect,
  placeholder,
  disabled,
  invalid,
  minChars = 1,
  emptyText = "No matches",
}: ComboboxProps) {
  const [open, setOpen] = useState(false);
  const [options, setOptions] = useState<ComboboxOption[]>([]);
  const [loading, setLoading] = useState(false);
  const listId = useId();
  const containerRef = useRef<HTMLDivElement>(null);
  // Suppresses the search that would otherwise fire from the value change we
  // trigger ourselves when a suggestion is selected.
  const skipNextSearch = useRef(false);

  useEffect(() => {
    if (skipNextSearch.current) {
      skipNextSearch.current = false;
      return;
    }
    const query = value.trim();
    if (!open || query.length < minChars) {
      setOptions([]);
      return;
    }
    let cancelled = false;
    setLoading(true);
    const handle = setTimeout(() => {
      onSearch(query)
        .then((result) => {
          if (!cancelled) setOptions(result);
        })
        .catch(() => {
          if (!cancelled) setOptions([]);
        })
        .finally(() => {
          if (!cancelled) setLoading(false);
        });
    }, 250);
    return () => {
      cancelled = true;
      clearTimeout(handle);
    };
  }, [value, open, minChars, onSearch]);

  // Close the dropdown when clicking outside.
  useEffect(() => {
    if (!open) return;
    function onDocClick(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, [open]);

  function choose(option: ComboboxOption) {
    skipNextSearch.current = true;
    onSelect(option);
    setOpen(false);
    setOptions([]);
  }

  const showList = open && value.trim().length >= minChars;

  return (
    <div ref={containerRef} className="relative">
      <Input
        id={id}
        role="combobox"
        aria-expanded={showList}
        aria-controls={listId}
        aria-autocomplete="list"
        autoComplete="off"
        value={value}
        placeholder={placeholder}
        disabled={disabled}
        invalid={invalid}
        onChange={(e) => {
          onValueChange(e.target.value);
          setOpen(true);
        }}
        onFocus={() => setOpen(true)}
      />
      {showList && (
        <ul
          id={listId}
          role="listbox"
          className="absolute z-20 mt-1 max-h-64 w-full overflow-auto rounded-sm border border-stone-200 bg-white py-1 shadow-lg"
        >
          {loading && options.length === 0 ? (
            <li className="px-3.5 py-2 text-sm text-stone-400">Searching…</li>
          ) : options.length === 0 ? (
            <li className="px-3.5 py-2 text-sm text-stone-400">{emptyText}</li>
          ) : (
            options.map((option) => (
              <li key={`${option.id}-${option.label}`} role="option" aria-selected={false}>
                <button
                  type="button"
                  className={cn(
                    "flex w-full flex-col items-start px-3.5 py-2 text-left text-sm text-stone-900",
                    "hover:bg-stone-100 focus:bg-stone-100 focus:outline-none",
                  )}
                  // onMouseDown (not onClick) so the selection registers before
                  // the input's blur closes the list.
                  onMouseDown={(e) => {
                    e.preventDefault();
                    choose(option);
                  }}
                >
                  <span>{option.label}</span>
                  {option.sublabel && <span className="text-xs text-stone-400">{option.sublabel}</span>}
                </button>
              </li>
            ))
          )}
        </ul>
      )}
    </div>
  );
}
