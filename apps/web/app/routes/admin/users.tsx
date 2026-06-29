import { useEffect, useState } from "react";

import { useAuth } from "../../features/auth/AuthContext";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { EmptyState } from "../../components/admin/EmptyState";
import { Icon } from "../../components/ui/Icon";
import { Input } from "../../components/ui/Input";
import { Text } from "../../components/ui/Text";
import { type AdminUser, listAdminUsers, updateUserRoles } from "../../lib/api/admin-users";

export const handle = { title: "User Management" };

const PAGE_SIZE = 20;

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(undefined, { dateStyle: "medium" });
}

export default function AdminUsers() {
  const { profile } = useAuth();
  const [users, setUsers] = useState<AdminUser[] | null>(null);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [updatingId, setUpdatingId] = useState<string | null>(null);

  async function refresh() {
    try {
      const result = await listAdminUsers({ search: search || undefined, page, pageSize: PAGE_SIZE });
      setUsers(result.users);
      setTotal(result.total);
    } catch {
      setError("Could not load users.");
    }
  }

  useEffect(() => {
    const timeout = setTimeout(refresh, 300);
    return () => clearTimeout(timeout);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search, page]);

  useEffect(() => {
    setPage(1);
  }, [search]);

  async function toggleAdmin(user: AdminUser) {
    const isAdmin = user.roles.includes("admin");
    const nextRoles = isAdmin ? user.roles.filter((r) => r !== "admin") : [...user.roles, "admin"];

    if (isAdmin && user.id === profile?.id) {
      window.alert("You can't remove your own admin role.");
      return;
    }
    if (!window.confirm(isAdmin ? `Remove admin access from ${user.full_name || user.email}?` : `Grant admin access to ${user.full_name || user.email}?`)) {
      return;
    }

    setUpdatingId(user.id);
    try {
      const updated = await updateUserRoles(user.id, nextRoles);
      setUsers((prev) => prev?.map((u) => (u.id === updated.id ? updated : u)) ?? prev);
    } catch {
      window.alert("Could not update roles. Try again.");
    } finally {
      setUpdatingId(null);
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="flex flex-col gap-4">
      <div className="relative max-w-sm">
        <Icon name="search" size={16} className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-stone-400" />
        <Input
          placeholder="Search users by name or email"
          className="pl-9"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {users === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : users.length === 0 ? (
        <EmptyState icon="users" title="No users found" description="Try a different search." />
      ) : (
        <>
          <div className="overflow-hidden rounded-sm border border-stone-200 bg-white">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-stone-200 bg-stone-50 text-xs uppercase tracking-wide text-stone-500">
                <tr>
                  <th className="px-4 py-3 font-medium">Name</th>
                  <th className="px-4 py-3 font-medium">Email</th>
                  <th className="px-4 py-3 font-medium">Roles</th>
                  <th className="px-4 py-3 font-medium">Joined</th>
                  <th className="px-4 py-3 font-medium">Orders</th>
                  <th className="px-4 py-3 font-medium" />
                </tr>
              </thead>
              <tbody>
                {users.map((user) => {
                  const isAdmin = user.roles.includes("admin");
                  return (
                    <tr key={user.id} className="border-b border-stone-100 last:border-0">
                      <td className="px-4 py-3 font-medium text-stone-900">{user.full_name || "—"}</td>
                      <td className="px-4 py-3 text-stone-600">{user.email}</td>
                      <td className="px-4 py-3">
                        <div className="flex gap-1.5">
                          {user.roles.map((role) => (
                            <Badge key={role} variant={role === "admin" ? "accent" : "neutral"}>
                              {role}
                            </Badge>
                          ))}
                        </div>
                      </td>
                      <td className="px-4 py-3 text-stone-600">{formatDate(user.created_at)}</td>
                      <td className="px-4 py-3 text-stone-600">{user.order_count}</td>
                      <td className="px-4 py-3 text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={updatingId === user.id}
                          onClick={() => toggleAdmin(user)}
                        >
                          {isAdmin ? "Remove admin" : "Make admin"}
                        </Button>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>

          <div className="flex items-center justify-between">
            <Text size="sm" tone="muted">
              {total} {total === 1 ? "user" : "users"}
            </Text>
            <div className="flex items-center gap-2">
              <Button variant="outline" size="sm" disabled={page <= 1} onClick={() => setPage((p) => p - 1)}>
                Previous
              </Button>
              <Text size="sm" tone="muted">
                Page {page} of {totalPages}
              </Text>
              <Button variant="outline" size="sm" disabled={page >= totalPages} onClick={() => setPage((p) => p + 1)}>
                Next
              </Button>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
