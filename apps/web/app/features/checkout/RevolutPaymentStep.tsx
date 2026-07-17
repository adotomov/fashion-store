import type { RevolutCheckoutCardField, RevolutCheckoutInstance } from "@revolut/checkout";
import { useEffect, useRef, useState } from "react";

import { Button } from "../../components/ui/Button";
import { Text } from "../../components/ui/Text";
import { getOrderPaymentStatus } from "../../lib/api/checkout";
import { useLanguage } from "../i18n/LanguageContext";

type PaymentRequestInstance = ReturnType<RevolutCheckoutInstance["paymentRequest"]>;

// The checkout widget mode is derived from a single build-time env var; no
// publishable key ships in the bundle — the per-order token carries the auth.
const REVOLUT_ENV = (import.meta.env.VITE_REVOLUT_ENV as "sandbox" | "prod" | undefined) ?? "sandbox";

// Poll cadence for the post-payment confirmation: the widget's success is a
// client-side signal; the order only becomes "paid" when our webhook settles
// it, usually within a second or two.
const POLL_INTERVAL_MS = 1500;
const POLL_MAX_ATTEMPTS = 20;

type Props = {
  /** Revolut order public token from the checkout initiation response. */
  token: string;
  orderNumber: string;
  cardHolderName: string;
  email: string;
  payLabel: string;
  onPaid: () => void;
  onFailed: (message: string) => void;
};

export function RevolutPaymentStep({ token, orderNumber, cardHolderName, email, payLabel, onPaid, onFailed }: Props) {
  const { t } = useLanguage();
  const cardTargetRef = useRef<HTMLDivElement>(null);
  const payRequestRef = useRef<HTMLDivElement>(null);
  const cardFieldRef = useRef<RevolutCheckoutCardField | null>(null);
  const paymentRequestRef = useRef<PaymentRequestInstance | null>(null);
  const instanceRef = useRef<RevolutCheckoutInstance | null>(null);

  const [ready, setReady] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [walletAvailable, setWalletAvailable] = useState(false);

  // Wait for the webhook to settle the order before showing success.
  async function confirmSettlement() {
    setConfirming(true);
    for (let attempt = 0; attempt < POLL_MAX_ATTEMPTS; attempt++) {
      try {
        const { status } = await getOrderPaymentStatus(orderNumber);
        if (status === "payment_failed") {
          setConfirming(false);
          onFailed(t("checkout.payment_failed_error", "Payment could not be processed. Please try again."));
          return;
        }
        if (status !== "pending_payment") {
          setConfirming(false);
          onPaid();
          return;
        }
      } catch {
        // transient — keep polling
      }
      await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
    }
    // The card cleared but our confirmation is lagging; treat as success — the
    // webhook/sweeper will settle it and the order shows on the account page.
    setConfirming(false);
    onPaid();
  }

  useEffect(() => {
    let cancelled = false;

    async function mount() {
      const cardTarget = cardTargetRef.current;
      const payTarget = payRequestRef.current;
      if (!cardTarget || !payTarget) return;
      try {
        const RevolutCheckout = (await import("@revolut/checkout")).default;
        const instance = await RevolutCheckout(token, REVOLUT_ENV);
        if (cancelled) {
          instance.destroy();
          return;
        }
        instanceRef.current = instance;

        cardFieldRef.current = instance.createCardField({
          target: cardTarget,
          onSuccess: () => void confirmSettlement(),
          onError: (err) => {
            setSubmitting(false);
            setError(err?.message ?? t("checkout.payment_failed_error", "Payment could not be processed. Please try again."));
          },
        });

        // Apple Pay / Google Pay — rendered only for the wallet the current
        // device/browser supports, and nothing if none is available.
        const pr = instance.paymentRequest({
          target: payTarget,
          onSuccess: () => void confirmSettlement(),
          onError: (err) => setError(err?.message ?? "wallet error"),
        });
        paymentRequestRef.current = pr;
        try {
          if (await pr.canMakePayment()) {
            await pr.render();
            setWalletAvailable(true);
          }
        } catch {
          setWalletAvailable(false);
        }

        setReady(true);
      } catch {
        if (!cancelled) setError(t("checkout.payment_widget_error", "Could not load the payment form. Please try again."));
      }
    }

    void mount();
    return () => {
      cancelled = true;
      paymentRequestRef.current?.destroy();
      instanceRef.current?.destroy();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token]);

  function handlePay() {
    if (!cardFieldRef.current) return;
    setError(null);
    setSubmitting(true);
    cardFieldRef.current.submit({ name: cardHolderName, email });
  }

  if (confirming) {
    return (
      <div className="flex flex-col items-center gap-3 py-8 text-center">
        <Text className="font-medium">{t("checkout.confirming_payment", "Confirming your payment…")}</Text>
        <Text size="sm" tone="muted">
          {t("checkout.confirming_payment_hint", "This only takes a moment.")}
        </Text>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {/* Always mounted so the ref exists when the SDK renders the wallet
          button; the divider only shows once a wallet is confirmed available. */}
      <div ref={payRequestRef} />
      {walletAvailable && (
        <div className="flex items-center gap-3 text-xs uppercase tracking-wide text-stone-400">
          <span className="h-px flex-1 bg-stone-200" />
          {t("checkout.or_pay_by_card", "or pay by card")}
          <span className="h-px flex-1 bg-stone-200" />
        </div>
      )}

      <div ref={cardTargetRef} className="rounded-sm border border-stone-200 p-3" />

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      <Button variant="primary" onClick={handlePay} disabled={!ready || submitting}>
        {submitting ? t("checkout.processing_payment", "Processing payment…") : payLabel}
      </Button>

      <Text size="xs" tone="muted" className="text-center">
        {t("checkout.card_secured_by_revolut", "Payments are securely processed by Revolut.")}
      </Text>
    </div>
  );
}
