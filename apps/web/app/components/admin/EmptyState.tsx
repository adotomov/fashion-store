import { Icon, type IconName } from "../ui/Icon";
import { Text } from "../ui/Text";

type EmptyStateProps = {
  icon: IconName;
  title: string;
  description: string;
};

export function EmptyState({ icon, title, description }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center rounded-sm border border-dashed border-stone-300 bg-white px-6 py-20 text-center">
      <span className="flex h-12 w-12 items-center justify-center rounded-full bg-stone-100 text-stone-500">
        <Icon name={icon} size={22} />
      </span>
      <Text className="mt-4 font-medium">{title}</Text>
      <Text size="sm" tone="muted" className="mt-1.5 max-w-sm">
        {description}
      </Text>
    </div>
  );
}
