import type { ReactNode } from "react";
import { useMemo } from "react";
import { Navigate, useLocation } from "react-router";

import { useAuth } from "../features/auth/AuthContext";

// UX-only route guard, same caveat as RequireAuth: the backend must enforce
// this independently once admin endpoints exist. Today no backend admin
// module exists yet, so this only gates the UI shell, not real data access.
export function RequireAdmin({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading, profile } = useAuth();
  const location = useLocation();
  // See RequireAuth: this must stay referentially stable or Navigate's
  // redirect effect loops forever.
  const loginState = useMemo(
    () => ({ from: { pathname: location.pathname, search: location.search } }),
    [location.pathname, location.search],
  );

  if (isLoading) {
    return <p>Loading…</p>;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={loginState} replace />;
  }

  if (!profile?.roles.includes("admin")) {
    return <Navigate to="/" replace />;
  }

  return <>{children}</>;
}
