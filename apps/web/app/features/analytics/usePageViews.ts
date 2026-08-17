import { useEffect, useRef } from "react";
import { useLocation } from "react-router";

import { trackPageView } from "../../lib/analytics/gtag";

// usePageViews fires a GA4 page_view on every client-side navigation. Because
// the gtag config was set with send_page_view:false, this is the sole source of
// page views — including the first one on initial load (the ref starts null, so
// the mount effect sends it). Consecutive effects with an unchanged path are
// skipped so a re-render doesn't double-count.
//
// Safe when analytics is OFF: trackPageView is a no-op without a Measurement ID.
export function usePageViews(): void {
  const location = useLocation();
  const lastPath = useRef<string | null>(null);

  useEffect(() => {
    const path = location.pathname + location.search;
    if (lastPath.current === path) return;
    lastPath.current = path;
    trackPageView(path);
  }, [location.pathname, location.search]);
}
