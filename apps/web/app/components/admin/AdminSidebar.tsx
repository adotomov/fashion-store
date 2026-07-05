import { useEffect, useState } from "react";
import { Link, NavLink } from "react-router";

import { useAuth } from "../../features/auth/AuthContext";
import { useStoreBranding } from "../../features/store-settings/StoreSettingsContext";
import { getUnviewedOrderCount } from "../../lib/api/admin-orders";
import { cn } from "../../lib/utils/cn";
import { Icon, type IconName } from "../ui/Icon";

type AdminNavItem = {
  label: string;
  href: string;
  icon: IconName;
};

const navItems: AdminNavItem[] = [
  { label: "Dashboard", href: "/admin", icon: "dashboard" },
  { label: "Home Page", href: "/admin/home", icon: "catalog" },
  { label: "Store Settings", href: "/admin/settings", icon: "settings" },
  { label: "Translations", href: "/admin/translations", icon: "globe" },
  { label: "User Management", href: "/admin/users", icon: "users" },
  { label: "Catalog", href: "/admin/catalog", icon: "catalog" },
  { label: "Orders", href: "/admin/orders", icon: "invoices" },
  { label: "Inventory", href: "/admin/inventory", icon: "inventory" },
  { label: "Logistics", href: "/admin/logistics", icon: "shipping" },
  { label: "Invoices & Tax", href: "/admin/invoices", icon: "invoices" },
  { label: "Promotions", href: "/admin/promotions", icon: "tag" },
];

// Polled rather than pushed — no websocket/SSE infrastructure exists yet,
// so a periodic refresh is the cheapest way to surface "new orders" without
// a manual page reload.
const UNVIEWED_POLL_INTERVAL_MS = 30_000;

export function AdminSidebar() {
  const { profile, logout } = useAuth();
  const { storeName, logoUrl } = useStoreBranding();
  const initials = getInitials(profile?.full_name || profile?.email || "?");
  const [unviewedCount, setUnviewedCount] = useState(0);

  useEffect(() => {
    function refresh() {
      getUnviewedOrderCount()
        .then(setUnviewedCount)
        .catch(() => {});
    }
    refresh();
    const interval = setInterval(refresh, UNVIEWED_POLL_INTERVAL_MS);
    return () => clearInterval(interval);
  }, []);

  return (
    <aside className="flex h-full w-64 shrink-0 flex-col border-r border-stone-200 bg-white">
      <div className="flex h-20 items-center justify-between gap-2 border-b border-stone-200 px-6">
        <div className="flex min-w-0 items-center gap-2">
          {logoUrl && <img src={logoUrl} alt={storeName} className="h-7 w-auto shrink-0 object-contain" />}
          <span className="truncate font-display text-xl font-medium tracking-wide text-stone-900">{storeName}</span>
          <span className="shrink-0 text-xs font-medium uppercase tracking-wide text-stone-400">Admin</span>
        </div>
        <Link
          to="/"
          aria-label="Back to store"
          title="Back to store"
          className="shrink-0 rounded-sm p-2 text-stone-500 transition-colors hover:bg-stone-50 hover:text-stone-900"
        >
          <Icon name="chevronLeft" size={18} />
        </Link>
      </div>

      <nav className="flex-1 overflow-y-auto px-3 py-4">
        <ul className="flex flex-col gap-1">
          {navItems.map((item) => (
            <li key={item.href}>
              <NavLink
                to={item.href}
                end={item.href === "/admin"}
                className={({ isActive }) =>
                  cn(
                    "flex items-center gap-3 rounded-sm px-3 py-2.5 text-sm font-medium transition-colors",
                    isActive ? "bg-stone-900 text-white" : "text-stone-600 hover:bg-stone-50 hover:text-stone-900",
                  )
                }
              >
                <Icon name={item.icon} size={18} />
                {item.label}
                {item.href === "/admin/orders" && unviewedCount > 0 && (
                  <span className="ml-auto flex h-5 min-w-5 items-center justify-center rounded-full bg-clay-500 px-1.5 text-xs font-medium text-white">
                    {unviewedCount}
                  </span>
                )}
              </NavLink>
            </li>
          ))}
        </ul>
      </nav>

      <div className="border-t border-stone-200 p-3">
        <div className="flex items-center gap-3 rounded-sm px-2 py-2">
          <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-stone-200 text-sm font-medium text-stone-700">
            {initials}
          </span>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-stone-900">{profile?.full_name || "Admin"}</p>
            <p className="truncate text-xs text-stone-500">{profile?.email}</p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => void logout()}
          className="mt-1 flex w-full items-center gap-3 rounded-sm px-3 py-2.5 text-sm font-medium text-stone-600 transition-colors hover:bg-stone-50 hover:text-danger-600"
        >
          <Icon name="logout" size={18} />
          Log Out
        </button>
      </div>
    </aside>
  );
}

function getInitials(value: string): string {
  const parts = value.trim().split(/\s+/);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return value.slice(0, 2).toUpperCase();
}
