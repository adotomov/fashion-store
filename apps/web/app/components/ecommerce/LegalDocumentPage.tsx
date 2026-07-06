import { useEffect, useState } from "react";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { Footer } from "./Footer";
import { Header } from "./Header";
import { Heading, Text } from "../ui/Text";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { type DocumentType, getStorefrontLegalContent } from "../../lib/api/store-documents";

type LegalDocumentPageProps = {
  title: string;
  documentType: DocumentType;
};

export function LegalDocumentPage({ title, documentType }: LegalDocumentPageProps) {
  const { locale } = useLanguage();
  const [content, setContent] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setContent(null);
    setNotFound(false);
    getStorefrontLegalContent(documentType, locale)
      .then((r) => {
        if (!cancelled) setContent(r.content_md);
      })
      .catch(() => {
        if (!cancelled) setNotFound(true);
      });
    return () => {
      cancelled = true;
    };
  }, [documentType, locale]);

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
          <Heading as="h1" size="lg">
            {title}
          </Heading>

          {notFound ? (
            <Text className="mt-6" tone="muted">
              This document is not available yet.
            </Text>
          ) : content === null ? (
            <Text className="mt-6" tone="muted">
              Loading…
            </Text>
          ) : (
            <div className="prose prose-stone mt-8 max-w-none">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
            </div>
          )}
        </div>
      </main>
      <Footer />
    </div>
  );
}
