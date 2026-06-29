import { useEffect, useState } from "react";

import { Footer } from "./Footer";
import { Header } from "./Header";
import { Heading, Text } from "../ui/Text";
import { buttonStyles } from "../ui/Button";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { type DocumentType, storefrontDocumentUrl } from "../../lib/api/storefront";

type LegalDocumentPageProps = {
  title: string;
  documentType: DocumentType;
};

export function LegalDocumentPage({ title, documentType }: LegalDocumentPageProps) {
  const { locale } = useLanguage();
  const [available, setAvailable] = useState<boolean | null>(null);
  const fileUrl = storefrontDocumentUrl(documentType, locale);

  useEffect(() => {
    let cancelled = false;
    fetch(fileUrl, { method: "HEAD" })
      .then((res) => {
        if (!cancelled) setAvailable(res.ok);
      })
      .catch(() => {
        if (!cancelled) setAvailable(false);
      });
    return () => {
      cancelled = true;
    };
  }, [fileUrl]);

  return (
    <div className="flex min-h-screen flex-col">
      <Header />
      <main className="flex-1">
        <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6 lg:px-8">
          <div className="flex flex-wrap items-center justify-between gap-4">
            <Heading as="h1" size="lg">
              {title}
            </Heading>
            {available && (
              <a href={fileUrl} download className={buttonStyles({ variant: "outline", size: "sm" })}>
                Download
              </a>
            )}
          </div>

          {available ? (
            <iframe title={title} src={fileUrl} className="mt-8 h-[75vh] w-full rounded-sm border border-stone-200" />
          ) : (
            <Text className="mt-6" tone="muted">
              This document is not available yet.
            </Text>
          )}
        </div>
      </main>
      <Footer />
    </div>
  );
}
