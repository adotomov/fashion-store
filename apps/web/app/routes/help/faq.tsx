import { LegalDocumentPage } from "../../components/ecommerce/LegalDocumentPage";
import { buildMeta } from "../../lib/seo/meta";

export function meta() {
  return buildMeta({
    title: "Frequently Asked Questions",
    description: "Answers to common questions about ordering, payment, delivery, and returns at Verani.",
    path: "/help/faq",
  });
}

export default function FAQ() {
  return <LegalDocumentPage title="Frequently Asked Questions" documentType="faq" />;
}
