import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useLanguage } from "../../features/i18n/LanguageContext";
import { useStoreBranding } from "../../features/store-settings/StoreSettingsContext";
import { type NavType, getNav } from "../../lib/api/storefront";
import { Icon } from "../ui/Icon";
import { Eyebrow, Text } from "../ui/Text";

type FooterColumn = {
  title: string;
  links: { label: string; href: string }[];
};

export function Footer() {
  const { storeName, logoUrl } = useStoreBranding();
  const { locale, t } = useLanguage();
  const [navTypes, setNavTypes] = useState<NavType[]>([]);

  useEffect(() => {
    getNav(locale)
      .then(setNavTypes)
      .catch(() => {});
  }, [locale]);

  const shopColumn: FooterColumn = {
    title: t("nav.shop", "Shop"),
    links: [
      { label: t("nav.new_arrivals", "New Arrivals"), href: "/shop" },
      ...navTypes.map((type) => ({ label: type.name, href: `/shop?type=${type.slug}` })),
    ],
  };

  const staticColumns: FooterColumn[] = [
    {
      title: t("footer.help", "Help"),
      links: [
        { label: t("footer.shipping_returns", "Shipping & Returns"), href: "/help/shipping" },
        { label: t("footer.size_guide", "Size Guide"), href: "/help/sizing" },
        { label: t("footer.contact_us", "Contact Us"), href: "/help/contact" },
        { label: t("footer.faq", "FAQ"), href: "/help/faq" },
      ],
    },
    {
      title: t("footer.company", "Company"),
      links: [
        { label: t("footer.about", "About"), href: "/about" },
        { label: t("footer.sustainability", "Sustainability"), href: "/sustainability" },
        { label: t("footer.privacy_policy", "Privacy Policy"), href: "/legal/privacy" },
        { label: t("footer.terms_of_service", "Terms of Service"), href: "/legal/terms" },
      ],
    },
  ];

  const columns = [shopColumn, ...staticColumns];

  return (
    <footer className="border-t border-stone-200 bg-stone-50">
      <div className="mx-auto max-w-7xl px-4 py-12 sm:px-6 lg:px-8">
        <div className="grid grid-cols-2 gap-8 md:grid-cols-5">
          <div className="col-span-2">
            <span className="flex items-center gap-2">
              {logoUrl && <img src={logoUrl} alt={storeName} className="h-8 w-auto object-contain" />}
              <span className="font-display text-2xl font-medium tracking-wide text-stone-900">{storeName}</span>
            </span>
            <Text size="sm" tone="muted" className="mt-3 max-w-xs">
              Clothing, jewelry, bags, and accessories, thoughtfully made and delivered with care.
            </Text>
            <div className="mt-4 flex items-center gap-3 text-stone-500">
              <a href="#" aria-label="Instagram" className="hover:text-stone-900">
                <Icon name="instagram" size={20} />
              </a>
              <a href="#" aria-label="Facebook" className="hover:text-stone-900">
                <Icon name="facebook" size={20} />
              </a>
            </div>
          </div>

          {columns.map((column) => (
            <div key={column.title}>
              <Eyebrow>{column.title}</Eyebrow>
              <ul className="mt-3 flex flex-col gap-2.5">
                {column.links.map((link) => (
                  <li key={link.href}>
                    <Link to={link.href} className="text-sm text-stone-600 hover:text-stone-900">
                      {link.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-10 flex flex-col gap-4 border-t border-stone-200 pt-6 sm:flex-row sm:items-center sm:justify-between">
          <Text size="xs" tone="muted">
            &copy; {new Date().getFullYear()} {storeName}. All rights reserved.
          </Text>
          <div className="flex items-center gap-4 text-stone-500">
            <span className="flex items-center gap-1.5 text-xs">
              <Icon name="shipping" size={14} />
              {t("footer.free_shipping", "Free shipping over $100")}
            </span>
            <span className="flex items-center gap-1.5 text-xs">
              <Icon name="mail" size={14} />
              hello@maison.example
            </span>
          </div>
        </div>
      </div>
    </footer>
  );
}
