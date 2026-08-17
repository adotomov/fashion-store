import { LegalDocumentPage } from "../../components/ecommerce/LegalDocumentPage";
import { buildMeta } from "../../lib/seo/meta";

export function meta() {
  return buildMeta({
    title: "Privacy Policy",
    description: "How Verani collects, uses, and protects your personal data, including cookies and analytics.",
    path: "/legal/privacy",
  });
}

export default function Privacy() {
  return <LegalDocumentPage title="Privacy Policy" documentType="privacy" />;
}
