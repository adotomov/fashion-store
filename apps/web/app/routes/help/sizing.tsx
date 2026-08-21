import { LegalDocumentPage } from "../../components/ecommerce/LegalDocumentPage";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { buildMeta } from "../../lib/seo/meta";

export function meta() {
  return buildMeta({
    title: "Size Guide",
    description: "Verani size charts for clothing, plus how to measure yourself for the right fit.",
    path: "/help/sizing",
  });
}

export default function Sizing() {
  const { t } = useLanguage();
  return <LegalDocumentPage title={t("sizeguide.title", "Size Guide")} documentType="size_guide" />;
}
