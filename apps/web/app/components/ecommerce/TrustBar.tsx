import { useLanguage } from "../../features/i18n/LanguageContext";
import { useStoreBranding } from "../../features/store-settings/StoreSettingsContext";
import { withThreshold } from "../../lib/shipping/free-shipping";
import { Icon, type IconName } from "../ui/Icon";

type TrustItem = {
  icon: IconName;
  titleKey: string;
  titleFallback: string;
  subtitleKey: string;
  subtitleFallback: string;
};

// Static reassurance strip shown directly under the Hero. Its job is to answer
// the "can I trust this store?" question before the first product shelf — free
// shipping, returns, secure checkout, authenticity. Content is bilingual via
// the i18n `t()` lookup, matching how the Footer handles its static copy.
const ITEMS: TrustItem[] = [
  {
    icon: "shipping",
    titleKey: "trust.shipping_title",
    titleFallback: "Free shipping",
    subtitleKey: "trust.shipping_subtitle",
    subtitleFallback: "On orders over {amount}",
  },
  {
    icon: "returns",
    titleKey: "trust.returns_title",
    titleFallback: "Easy returns",
    subtitleKey: "trust.returns_subtitle",
    subtitleFallback: "30 days to change your mind",
  },
  {
    icon: "lock",
    titleKey: "trust.secure_title",
    titleFallback: "Secure checkout",
    subtitleKey: "trust.secure_subtitle",
    subtitleFallback: "Encrypted payments",
  },
  {
    icon: "shieldCheck",
    titleKey: "trust.authentic_title",
    titleFallback: "Authentic products",
    subtitleKey: "trust.authentic_subtitle",
    subtitleFallback: "Quality guaranteed",
  },
];

export function TrustBar() {
  const { t } = useLanguage();
  const { freeShippingThresholdMinor, storeLocale } = useStoreBranding();

  return (
    <section className="border-b border-stone-200 bg-stone-50">
      <div className="mx-auto grid max-w-7xl grid-cols-2 gap-x-6 gap-y-6 px-4 py-6 sm:px-6 lg:grid-cols-4 lg:px-8">
        {ITEMS.map((item) => (
          <div key={item.titleKey} className="flex items-center gap-3">
            <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-white text-clay-600 shadow-sm ring-1 ring-stone-200">
              <Icon name={item.icon} size={20} />
            </span>
            <span className="flex flex-col leading-tight">
              <span className="text-sm font-medium text-stone-900">
                {t(item.titleKey, item.titleFallback)}
              </span>
              <span className="text-xs text-stone-500">
                {withThreshold(t(item.subtitleKey, item.subtitleFallback), freeShippingThresholdMinor, storeLocale)}
              </span>
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}
