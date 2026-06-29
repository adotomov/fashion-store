import { cn } from "../../lib/utils/cn";

export type Tab = {
  id: string;
  label: string;
};

type TabsProps = {
  tabs: Tab[];
  activeTab: string;
  onChange: (id: string) => void;
};

export function Tabs({ tabs, activeTab, onChange }: TabsProps) {
  return (
    <div className="flex gap-6 border-b border-stone-200">
      {tabs.map((tab) => {
        const isActive = tab.id === activeTab;
        return (
          <button
            key={tab.id}
            type="button"
            onClick={() => onChange(tab.id)}
            aria-current={isActive}
            className={cn(
              "border-b-2 px-1 pb-3 text-sm font-medium transition-colors",
              isActive ? "border-stone-900 text-stone-900" : "border-transparent text-stone-500 hover:text-stone-900",
            )}
          >
            {tab.label}
          </button>
        );
      })}
    </div>
  );
}
