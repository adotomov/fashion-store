import { useEffect, useRef } from "react";

import { useAuth } from "./AuthContext";

declare global {
  interface Window {
    google?: {
      accounts: {
        id: {
          initialize: (config: {
            client_id: string;
            callback: (response: { credential: string }) => void;
          }) => void;
          renderButton: (parent: HTMLElement, options: Record<string, unknown>) => void;
        };
      };
    };
  }
}

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID ?? "";
const GSI_SCRIPT_SRC = "https://accounts.google.com/gsi/client";

export function GoogleSignInButton() {
  const containerRef = useRef<HTMLDivElement>(null);
  const { loginWithGoogleIdToken } = useAuth();

  useEffect(() => {
    if (!GOOGLE_CLIENT_ID) return;

    function render() {
      if (!window.google || !containerRef.current) return;
      window.google.accounts.id.initialize({
        client_id: GOOGLE_CLIENT_ID,
        callback: (response) => {
          void loginWithGoogleIdToken(response.credential);
        },
      });
      window.google.accounts.id.renderButton(containerRef.current, {
        theme: "outline",
        size: "large",
        shape: "rectangular",
        width: 360,
      });
    }

    if (window.google) {
      render();
      return;
    }

    const existing = document.querySelector(`script[src="${GSI_SCRIPT_SRC}"]`);
    if (existing) {
      existing.addEventListener("load", render);
      return;
    }

    const script = document.createElement("script");
    script.src = GSI_SCRIPT_SRC;
    script.async = true;
    script.defer = true;
    script.onload = render;
    document.head.appendChild(script);
  }, [loginWithGoogleIdToken]);

  if (!GOOGLE_CLIENT_ID) {
    return (
      <p className="rounded-sm border border-danger-100 bg-danger-50 px-3.5 py-2.5 text-sm text-danger-600">
        Set VITE_GOOGLE_CLIENT_ID to enable Google sign-in.
      </p>
    );
  }

  return <div className="flex justify-center" ref={containerRef} />;
}
