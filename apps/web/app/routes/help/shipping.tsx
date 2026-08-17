import { LegalDocumentPage } from "../../components/ecommerce/LegalDocumentPage";
import { buildMeta } from "../../lib/seo/meta";

export function meta() {
  return buildMeta({
    title: "Shipping & Returns",
    description: "Verani delivery options, shipping times, and our returns and exchange policy for Bulgaria.",
    path: "/help/shipping",
  });
}

export default function Shipping() {
  return <LegalDocumentPage title="Shipping & Returns" documentType="shipping" />;
}
