import type { ReactNode } from "react";
import { useMemo } from "react";
import { Navigate, useLocation } from "react-router";

import { useAuth } from "../features/auth/AuthContext";

// UX-only route guard. The backend is the source of truth for
// authorization; this just avoids flashing protected UI to signed-out users.
export function RequireAuth({ children }: { children: ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();
  const location = useLocation();
  // Navigate re-fires its redirect effect whenever `state` changes identity,
  // so this must stay referentially stable across renders of the same
  // location — an inline `{ from: location }` literal caused an infinite
  // redirect loop here.
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

  return <>{children}</>;
}
