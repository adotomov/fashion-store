import { LegalDocumentPage } from "../../components/ecommerce/LegalDocumentPage";
import { buildMeta } from "../../lib/seo/meta";

export function meta() {
  return buildMeta({
    title: "Terms of Service",
    description: "The terms and conditions governing your use of the Verani online store.",
    path: "/legal/terms",
  });
}

export default function Terms() {
  return <LegalDocumentPage title="Terms of Service" documentType="terms" />;
}
