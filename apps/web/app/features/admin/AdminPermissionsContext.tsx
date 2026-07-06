import { createContext, useContext, type ReactNode } from "react";
import { useAuth } from "../auth/AuthContext";

type AdminRole = "admin" | "audit" | "accountant" | null;

type AdminPermissions = {
  role: AdminRole;
  isReadOnly: boolean;
  isAccountant: boolean;
  canAccessPath: (path: string) => boolean;
};

const AdminPermissionsContext = createContext<AdminPermissions | undefined>(undefined);

export function AdminPermissionsProvider({ children }: { children: ReactNode }) {
  const { profile } = useAuth();

  const roles = profile?.roles ?? [];
  const role: AdminRole = roles.includes("admin")
    ? "admin"
    : roles.includes("audit")
      ? "audit"
      : roles.includes("accountant")
        ? "accountant"
        : null;

  const isAccountant = role === "accountant";
  const isReadOnly = role === "audit" || role === "accountant";

  function canAccessPath(path: string): boolean {
    if (!isAccountant) return true;
    return path === "/admin/invoices" || path.startsWith("/admin/invoices/");
  }

  return (
    <AdminPermissionsContext.Provider value={{ role, isReadOnly, isAccountant, canAccessPath }}>
      {children}
    </AdminPermissionsContext.Provider>
  );
}

export function useAdminPermissions(): AdminPermissions {
  const ctx = useContext(AdminPermissionsContext);
  if (!ctx) throw new Error("useAdminPermissions must be used within AdminPermissionsProvider");
  return ctx;
}
