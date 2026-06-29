import { Icon } from "../ui/Icon";
import { Heading } from "../ui/Text";

type AdminHeaderProps = {
  title: string;
};

// Intentionally minimal: page title and a decorative notification icon.
// No real functionality lives here — each page's own content drives the UI.
export function AdminHeader({ title }: AdminHeaderProps) {
  return (
    <header className="flex h-20 shrink-0 items-center justify-between border-b border-stone-200 bg-white px-8">
      <Heading as="h1" size="sm">
        {title}
      </Heading>
      <button
        type="button"
        aria-label="Notifications"
        className="rounded-full p-2.5 text-stone-500 hover:bg-stone-50"
      >
        <Icon name="bell" size={20} />
      </button>
    </header>
  );
}
