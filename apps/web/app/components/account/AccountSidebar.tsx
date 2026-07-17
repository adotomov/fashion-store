import { NavLink } from "react-router";

import { cn } from "../../lib/utils/cn";
import { Icon, type IconName } from "../ui/Icon";

type AccountNavItem = {
  label: string;
  href: string;
  icon: IconName;
};

const navItems: AccountNavItem[] = [
  { label: "Personal Info", href: "/account", icon: "profile" },
  { label: "Addresses", href: "/account/addresses", icon: "mapPin" },
  { label: "My Orders", href: "/account/orders", icon: "inventory" },
];

export function AccountSidebar() {
  return (
    <nav>
      <ul className="flex flex-col gap-1">
        {navItems.map((item) => (
          <li key={item.href}>
            <NavLink
              to={item.href}
              end
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-sm px-3 py-2.5 text-sm font-medium transition-colors",
                  isActive ? "bg-stone-900 text-white" : "text-stone-600 hover:bg-stone-50 hover:text-stone-900",
                )
              }
            >
              <Icon name={item.icon} size={18} />
              {item.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </nav>
  );
}
