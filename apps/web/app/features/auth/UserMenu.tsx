import { useState } from "react";
import { Link, useNavigate } from "react-router";

import { Icon } from "../../components/ui/Icon";
import { Text } from "../../components/ui/Text";
import { useAuth } from "./AuthContext";

function getInitials(fullName: string, email: string): string {
  const parts = fullName.trim().split(/\s+/).filter(Boolean);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return email.slice(0, 2).toUpperCase();
}

export function UserMenu() {
  const { profile, logout } = useAuth();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  if (!profile) return null;

  async function handleLogout() {
    setOpen(false);
    await logout();
    navigate("/");
  }

  return (
    <div className="relative" onMouseEnter={() => setOpen(true)} onMouseLeave={() => setOpen(false)}>
      <Link
        to="/account"
        aria-label="Account"
        className="flex items-center gap-2 rounded-sm p-1.5 pr-2 hover:bg-stone-50"
      >
        <span className="flex h-8 w-8 items-center justify-center rounded-full bg-stone-200 text-xs font-medium text-stone-900">
          {getInitials(profile.full_name, profile.email)}
        </span>
      </Link>

      {open && (
        <div className="absolute right-0 top-full z-40 w-56 rounded-sm border border-stone-200 bg-white py-2 shadow-lg">
          <div className="flex items-center gap-3 px-4 py-2">
            <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-stone-200 text-sm font-medium text-stone-900">
              {getInitials(profile.full_name, profile.email)}
            </span>
            <div className="min-w-0">
              <Text size="sm" className="truncate font-medium">
                {profile.full_name || profile.email}
              </Text>
              <Text size="xs" tone="muted" className="truncate">
                {profile.email}
              </Text>
            </div>
          </div>

          <div className="my-2 border-t border-stone-200" />

          <Link
            to="/account"
            onClick={() => setOpen(false)}
            className="flex items-center gap-2.5 px-4 py-2 text-sm text-stone-700 hover:bg-stone-50 hover:text-stone-900"
          >
            <Icon name="profile" size={16} />
            Profile
          </Link>
          <button
            type="button"
            onClick={handleLogout}
            className="flex w-full items-center gap-2.5 px-4 py-2 text-left text-sm text-stone-700 hover:bg-stone-50 hover:text-stone-900"
          >
            <Icon name="logout" size={16} />
            Logout
          </button>
        </div>
      )}
    </div>
  );
}
