import { useEffect, useRef } from "react";
import { useLocation } from "react-router";

import { trackPageView } from "../../lib/analytics/gtag";
import { pixelPageView } from "../../lib/analytics/meta-pixel";

// usePageViews fires a page view on every client-side navigation, for both GA4
// (gtag config has send_page_view:false) and the Meta Pixel (its bootstrap omits
// the load-time PageView). This is the sole source of page views for both tags —
// including the first one on initial load (the ref starts null, so the mount
// effect sends it). Consecutive effects with an unchanged path are skipped so a
// re-render doesn't double-count.
//
// Safe when either tag is OFF: both helpers are no-ops without their ID.
export function usePageViews(): void {
  const location = useLocation();
  const lastPath = useRef<string | null>(null);

  useEffect(() => {
    const path = location.pathname + location.search;
    if (lastPath.current === path) return;
    lastPath.current = path;
    trackPageView(path);
    pixelPageView();
  }, [location.pathname, location.search]);
}
