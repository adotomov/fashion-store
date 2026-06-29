import type { ReactNode } from "react";

import { cn } from "../../lib/utils/cn";

type TabsProps = {
  tabs: { id: string; label: string }[];
  activeTab: string;
  onChange: (id: string) => void;
  children: ReactNode;
};

export function Tabs({ tabs, activeTab, onChange, children }: TabsProps) {
  return (
    <div>
      <div role="tablist" className="flex gap-6 border-b border-stone-200">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            type="button"
            role="tab"
            aria-selected={activeTab === tab.id}
            onClick={() => onChange(tab.id)}
            className={cn(
              "border-b-2 pb-3 text-sm font-medium transition-colors",
              activeTab === tab.id
                ? "border-stone-900 text-stone-900"
                : "border-transparent text-stone-500 hover:text-stone-900",
            )}
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div className="mt-6">{children}</div>
    </div>
  );
}
