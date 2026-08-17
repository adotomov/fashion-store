import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

import { setAnalyticsConsent } from "../../lib/analytics/gtag";

// Cookie-consent state for Google Consent Mode v2. The gtag bootstrap defaults
// every storage type to "denied"; this context is the single place that flips
// analytics_storage to "granted" once the visitor accepts, and it remembers the
// choice so returning visitors aren't asked again.
//
// Only "analytics" is modelled here — the store runs no advertising tags, so
// ad_* stays denied. Add an "ads" category here if that ever changes.

const STORAGE_KEY = "store_cookie_consent";

export type ConsentDecision = "accepted" | "rejected";

type ConsentState = {
  // null until the visitor has decided (or on the server before hydration).
  decision: ConsentDecision | null;
  // False during SSR and until the stored choice has been read on the client.
  // The banner waits for this so it never flashes for a returning visitor who
  // already decided (and never renders server-side).
  ready: boolean;
  accept: () => void;
  reject: () => void;
  // Re-open the choice (e.g. from a footer "Cookie settings" link).
  reopen: () => void;
};

const ConsentContext = createContext<ConsentState>({
  decision: null,
  ready: false,
  accept: () => {},
  reject: () => {},
  reopen: () => {},
});

function readStoredDecision(): ConsentDecision | null {
  if (typeof window === "undefined") return null;
  const v = window.localStorage.getItem(STORAGE_KEY);
  return v === "accepted" || v === "rejected" ? v : null;
}

export function ConsentProvider({ children }: { children: ReactNode }) {
  const [decision, setDecision] = useState<ConsentDecision | null>(null);
  const [ready, setReady] = useState(false);

  // Restore a prior choice on load and re-apply it to Consent Mode, so a
  // returning visitor who accepted is tracked from the first hit of the session
  // (gtag's default is denied until we say otherwise).
  useEffect(() => {
    const stored = readStoredDecision();
    if (stored) {
      setDecision(stored);
      setAnalyticsConsent(stored === "accepted");
    }
    setReady(true);
  }, []);

  function persist(next: ConsentDecision) {
    if (typeof window !== "undefined") window.localStorage.setItem(STORAGE_KEY, next);
    setDecision(next);
    setAnalyticsConsent(next === "accepted");
  }

  return (
    <ConsentContext.Provider
      value={{
        decision,
        ready,
        accept: () => persist("accepted"),
        reject: () => persist("rejected"),
        reopen: () => setDecision(null),
      }}
    >
      {children}
    </ConsentContext.Provider>
  );
}

export function useConsent(): ConsentState {
  return useContext(ConsentContext);
}
