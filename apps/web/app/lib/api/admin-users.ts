import { apiFetch } from "./client";

export type AdminUser = {
  id: string;
  email: string;
  full_name: string;
  phone: string;
  roles: string[];
  order_count: number;
  created_at: string;
};

export type AdminUserList = {
  users: AdminUser[];
  total: number;
  page: number;
  page_size: number;
};

export async function listAdminUsers(filter?: { search?: string; page?: number; pageSize?: number }): Promise<AdminUserList> {
  const params = new URLSearchParams();
  if (filter?.search) params.set("search", filter.search);
  if (filter?.page) params.set("page", String(filter.page));
  if (filter?.pageSize) params.set("page_size", String(filter.pageSize));
  const query = params.toString();
  return apiFetch<AdminUserList>(`/api/v1/admin/users${query ? `?${query}` : ""}`);
}

export async function getAdminUser(id: string): Promise<AdminUser> {
  return apiFetch<AdminUser>(`/api/v1/admin/users/${id}`);
}

export async function updateUserRoles(id: string, roles: string[]): Promise<AdminUser> {
  return apiFetch<AdminUser>(`/api/v1/admin/users/${id}/roles`, { method: "PATCH", body: { roles } });
}

export type CountBreakdown = { label: string; count: number };

export type DailyUserCount = { date: string; count: number };

export type UserStats = {
  total_users: number;
  new_24h: number;
  new_7d: number;
  new_30d: number;
  role_breakdown: CountBreakdown[];
  by_country: CountBreakdown[];
  daily_registrations: DailyUserCount[];
};

export async function getUserStats(): Promise<UserStats> {
  return apiFetch<UserStats>("/api/v1/admin/users/stats");
}
