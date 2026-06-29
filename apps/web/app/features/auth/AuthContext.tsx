import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { mergeGuestCartIntoUser } from "../../lib/api/cart";
import { apiFetch } from "../../lib/api/client";
import { clearToken, getToken, setToken } from "../../lib/auth/session";
import { clearCartToken } from "../../lib/cart/session";

export type Profile = {
  id: string;
  email: string;
  full_name: string;
  phone: string;
  roles: string[];
};

type AuthContextValue = {
  profile: Profile | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  loginWithGoogleIdToken: (idToken: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshProfile: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

type SessionResponse = {
  token: string;
  expires_at: string;
};

export function AuthProvider({ children }: { children: ReactNode }) {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  async function loadProfile() {
    if (!getToken()) {
      setProfile(null);
      setIsLoading(false);
      return;
    }
    try {
      const me = await apiFetch<Profile>("/api/v1/me");
      setProfile(me);
    } catch {
      clearToken();
      setProfile(null);
    } finally {
      setIsLoading(false);
    }
  }

  useEffect(() => {
    loadProfile();
  }, []);

  async function loginWithGoogleIdToken(idToken: string) {
    const session = await apiFetch<SessionResponse>("/api/v1/auth/google", {
      method: "POST",
      body: { id_token: idToken },
      auth: false,
    });
    setToken(session.token);
    try {
      await mergeGuestCartIntoUser();
    } catch {
      // a failed merge shouldn't block login — the guest cart, if any,
      // just stays unmerged and can be retried on next login
    }
    await loadProfile();
  }

  async function logout() {
    try {
      await apiFetch("/api/v1/auth/logout", { method: "POST" });
    } finally {
      clearToken();
      clearCartToken();
      setProfile(null);
    }
  }

  return (
    <AuthContext.Provider
      value={{
        profile,
        isLoading,
        isAuthenticated: profile !== null,
        loginWithGoogleIdToken,
        logout,
        refreshProfile: loadProfile,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return ctx;
}
