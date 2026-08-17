import { Link } from "react-router";

import { Button } from "../../components/ui/Button";
import { isAnalyticsEnabled } from "../../lib/analytics/config";
import { useLanguage } from "../i18n/LanguageContext";

import { useConsent } from "./ConsentContext";

// Cookie-consent banner for Google Consent Mode v2. Shown only when analytics is
// actually configured (no Measurement ID → no analytics cookies → nothing to
// consent to) and the visitor hasn't decided yet. Accept flips analytics
// tracking on; Decline keeps it off. The choice is remembered by ConsentContext.
export function ConsentBanner() {
  const { t } = useLanguage();
  const { decision, ready, accept, reject } = useConsent();

  if (!isAnalyticsEnabled || !ready || decision !== null) return null;

  return (
    <div
      role="dialog"
      aria-live="polite"
      aria-label={t("consent.aria_label", "Cookie consent")}
      className="fixed inset-x-0 bottom-0 z-50 border-t border-stone-200 bg-white/95 backdrop-blur supports-[backdrop-filter]:bg-white/80"
    >
      <div className="container mx-auto flex flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-stone-600">
          {t(
            "consent.message",
            "We use cookies to understand how our store is used and improve your experience. You can accept analytics cookies or continue with only the essentials.",
          )}{" "}
          <Link to="/legal/privacy" className="underline hover:text-stone-900">
            {t("consent.learn_more", "Learn more")}
          </Link>
        </p>
        <div className="flex shrink-0 gap-2">
          <Button variant="outline" size="sm" onClick={reject}>
            {t("consent.decline", "Decline")}
          </Button>
          <Button variant="primary" size="sm" onClick={accept}>
            {t("consent.accept", "Accept")}
          </Button>
        </div>
      </div>
    </div>
  );
}
