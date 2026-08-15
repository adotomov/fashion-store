import type { RevolutCheckoutCardField, RevolutCheckoutInstance } from "@revolut/checkout";
import { useEffect, useRef, useState } from "react";

import { Button } from "../../components/ui/Button";
import { Icon } from "../../components/ui/Icon";
import { Input } from "../../components/ui/Input";
import { Text } from "../../components/ui/Text";
import { getOrderPaymentStatus, reportCheckoutEvent } from "../../lib/api/checkout";
import { useLanguage } from "../i18n/LanguageContext";
import { PaymentBrandLogos } from "./PaymentBrandLogos";

type PaymentRequestInstance = ReturnType<RevolutCheckoutInstance["paymentRequest"]>;

// The checkout widget mode is derived from a single build-time env var; no
// publishable key ships in the bundle — the per-order token carries the auth.
const REVOLUT_ENV = (import.meta.env.VITE_REVOLUT_ENV as "sandbox" | "prod" | undefined) ?? "sandbox";

// Revolut validates the cardholder name from the submit call and declines a
// non-Latin value with error.invalid-name. Bulgarian shoppers routinely enter
// their delivery-contact name in Cyrillic, so we collect the card name
// separately and require Latin here — failing fast with a clear message instead
// of an opaque post-submit gateway decline. Allows basic Latin plus common
// European accents (José, François); rejects Cyrillic and other scripts. Needs
// at least two characters and must start with a letter.
const LATIN_NAME_RE = /^[A-Za-zÀ-ɏ][A-Za-zÀ-ɏ .'-]*$/;

function isLatinName(value: string): boolean {
  const trimmed = value.trim();
  return trimmed.length >= 2 && LATIN_NAME_RE.test(trimmed);
}

// Log severity for a widget event. Interaction chatter (field focus/validation
// while typing) rides at "debug" so the prod INFO log stays quiet; milestones
// (mounted, submit, success) at "info"; failures at "error". The server's
// LOG_LEVEL is what ultimately decides how much of this is written.
type EventLevel = "info" | "debug" | "error";

// safeDetail stringifies an arbitrary event payload (error, validation-error
// array, field-status object) for the log, tolerating Error instances and
// circular structures.
function safeDetail(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (value instanceof Error) return `${value.name}: ${value.message}`;
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

// emitWidgetEvent forwards a single payment-widget telemetry event to the
// backend. Always fires — the server's LOG_LEVEL, not the client, decides what
// is kept. Best-effort and non-blocking (see reportCheckoutEvent).
function emitWidgetEvent(
  event: string,
  level: EventLevel,
  orderNumber: string,
  extra: { status?: string; code?: string; message?: string; detail?: string } = {},
) {
  reportCheckoutEvent({ event, level, order_number: orderNumber, revolut_env: REVOLUT_ENV, ...extra });
}

// Poll cadence for the post-payment confirmation: the widget's success is a
// client-side signal; the order only becomes "paid" when our webhook settles
// it, usually within a second or two.
const POLL_INTERVAL_MS = 1500;
const POLL_MAX_ATTEMPTS = 20;

// Typographic styling for the Revolut card iframe so the number/expiry/CVV
// entry matches the store's own inputs (Inter, stone palette). The widget owns
// its layout (PAN grouped in 4s on top, expiry + CVV beneath) — these keys only
// style the text within each field-status. Colours are the stone-900 body /
// stone-400 placeholder / danger-600 invalid values used elsewhere in the UI.
const CARD_FONT_FAMILY = '"Inter", ui-sans-serif, system-ui, sans-serif';
const cardFieldStyles = {
  default: { color: "#1c1917", fontFamily: CARD_FONT_FAMILY, fontSize: "16px", fontWeight: "500" },
  empty: { color: "#a8a29e", fontFamily: CARD_FONT_FAMILY, fontSize: "16px" },
  autofilled: { color: "#1c1917", fontFamily: CARD_FONT_FAMILY, fontSize: "16px" },
  invalid: { color: "#dc2626", fontFamily: CARD_FONT_FAMILY, fontSize: "16px" },
} as const;

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
  // Tracks whether a Pay click is in flight. Needed because Revolut reports a
  // rejected card via onValidation (a field event that also fires during normal
  // typing), so we only surface it as a submit failure when a submit is pending.
  const submittingRef = useRef(false);

  const [ready, setReady] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [walletAvailable, setWalletAvailable] = useState(false);
  // The cardholder name is entered here, not reused from the delivery contact:
  // Revolut requires it in Latin. Prefill only when the contact name already is
  // Latin, otherwise leave it blank rather than guess a transliteration.
  const [cardName, setCardName] = useState(() => (isLatinName(cardHolderName) ? cardHolderName.trim() : ""));
  const [cardNameInvalid, setCardNameInvalid] = useState(false);

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
          theme: "light",
          styles: cardFieldStyles,
          onSuccess: () => {
            submittingRef.current = false;
            emitWidgetEvent("payment_success", "info", orderNumber);
            void confirmSettlement();
          },
          onError: (err) => {
            submittingRef.current = false;
            setSubmitting(false);
            emitWidgetEvent("card_error", "error", orderNumber, { code: err?.type, message: err?.message, detail: safeDetail(err) });
            setError(err?.message ?? t("checkout.payment_failed_error", "Payment could not be processed. Please try again."));
          },
          onCancel: () => {
            submittingRef.current = false;
            setSubmitting(false);
            emitWidgetEvent("payment_cancel", "info", orderNumber);
          },
          // Fires on every field status change (focus/blur, completeness) — the
          // interaction chatter, so it rides at "debug".
          onStatusChange: (status) => {
            emitWidgetEvent("field_status_change", "debug", orderNumber, { status: safeDetail(status) });
          },
          // A rejected/invalid card comes back through validation, not onError.
          // Only treat it as a submit failure when a Pay click is pending —
          // otherwise this fires on ordinary typing and would flash an error.
          onValidation: (errors) => {
            const rejectedOnSubmit = errors.length > 0 && submittingRef.current;
            // On submit a validation failure is a real decline ("error"); while
            // typing it's just field feedback ("debug").
            emitWidgetEvent(
              rejectedOnSubmit ? "card_rejected" : "validation_change",
              rejectedOnSubmit ? "error" : "debug",
              orderNumber,
              { detail: safeDetail(errors.map((e) => e.type)) },
            );
            if (rejectedOnSubmit) {
              submittingRef.current = false;
              setSubmitting(false);
              // The widget already highlights the offending field inline; this
              // adds a single translatable summary and, crucially, un-sticks the
              // Pay button so the shopper can correct and retry.
              setError(t("checkout.card_invalid_error", "Please check your card details and try again."));
            }
          },
        });

        // Apple Pay / Google Pay — rendered only for the wallet the current
        // device/browser supports, and nothing if none is available.
        const pr = instance.paymentRequest({
          target: payTarget,
          onSuccess: () => {
            emitWidgetEvent("wallet_success", "info", orderNumber);
            void confirmSettlement();
          },
          onError: (err) => {
            emitWidgetEvent("wallet_error", "error", orderNumber, { code: err?.type, message: err?.message, detail: safeDetail(err) });
            setError(err?.message ?? "wallet error");
          },
          onCancel: () => emitWidgetEvent("wallet_cancel", "info", orderNumber),
        });
        paymentRequestRef.current = pr;
        try {
          const method = await pr.canMakePayment();
          if (method) {
            await pr.render();
            setWalletAvailable(true);
            emitWidgetEvent("wallet_available", "info", orderNumber, { detail: method });
          } else {
            emitWidgetEvent("wallet_unavailable", "debug", orderNumber);
          }
        } catch (err) {
          setWalletAvailable(false);
          emitWidgetEvent("wallet_check_failed", "debug", orderNumber, { detail: safeDetail(err) });
        }

        setReady(true);
        emitWidgetEvent("widget_mounted", "info", orderNumber);
      } catch (err) {
        emitWidgetEvent("widget_load_error", "error", orderNumber, {
          message: err instanceof Error ? err.message : String(err),
          detail: safeDetail(err),
        });
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
    const name = cardName.trim();
    // Fail fast on a non-Latin name instead of letting Revolut decline it with
    // error.invalid-name after the round trip.
    if (!isLatinName(name)) {
      setCardNameInvalid(true);
      setError(
        t(
          "checkout.cardholder_name_invalid",
          "Enter the cardholder name in Latin letters, exactly as printed on your card.",
        ),
      );
      emitWidgetEvent("cardholder_name_invalid", "info", orderNumber);
      return;
    }
    setError(null);
    submittingRef.current = true;
    setSubmitting(true);
    emitWidgetEvent("submit_clicked", "info", orderNumber);
    cardFieldRef.current.submit({ name, email });
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
    <div className="flex w-full max-w-md flex-col gap-5 text-left">
      {/* Secure-payment banner, echoing the reassurance strip at the top of a
          Revolut Hosted Checkout page. */}
      <div className="flex items-center justify-center gap-2 rounded-sm bg-stone-50 py-2.5 text-stone-600">
        <Icon name="lock" size={15} />
        <Text size="xs" className="font-medium uppercase tracking-wide">
          {t("checkout.secure_payment_banner", "Secure payment")}
        </Text>
      </div>

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

      <div className="flex flex-col gap-1.5">
        <Text size="xs" className="font-medium uppercase tracking-wide text-stone-500">
          {t("checkout.cardholder_name_label", "Cardholder name")}
        </Text>
        <Input
          value={cardName}
          onChange={(e) => {
            setCardName(e.target.value);
            if (cardNameInvalid) setCardNameInvalid(false);
          }}
          invalid={cardNameInvalid}
          disabled={submitting}
          autoComplete="cc-name"
          placeholder={t("checkout.cardholder_name_placeholder", "As printed on your card")}
          aria-label={t("checkout.cardholder_name_label", "Cardholder name")}
        />
      </div>

      <div className="flex flex-col gap-1.5">
        <Text size="xs" className="font-medium uppercase tracking-wide text-stone-500">
          {t("checkout.card_details_label", "Card details")}
        </Text>
        <div
          ref={cardTargetRef}
          className="rounded-md border border-stone-300 bg-white px-3.5 py-3 shadow-sm transition-colors focus-within:border-stone-900"
        />
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      <Button variant="primary" onClick={handlePay} disabled={!ready || submitting} className="h-12 text-base">
        <Icon name="lock" size={16} />
        {submitting ? t("checkout.processing_payment", "Processing payment…") : payLabel}
      </Button>

      {/* Accepted methods + no-storage reassurance. Translatable copy, brand
          marks rendered inline (see PaymentBrandLogos). */}
      <div className="flex flex-col items-center gap-3 rounded-md border border-stone-200 bg-stone-50/60 px-4 py-4 text-center">
        <PaymentBrandLogos className="flex items-center justify-center gap-2" />
        <div className="flex items-start gap-2 text-left">
          <span className="mt-0.5 text-emerald-600">
            <Icon name="shieldCheck" size={16} />
          </span>
          <Text size="xs" tone="muted" className="leading-relaxed">
            {t(
              "checkout.accepted_methods_note",
              "We accept Visa, Mastercard, Apple Pay and Google Pay. Your payment is encrypted and processed securely by Revolut — we never see or store your card details.",
            )}
          </Text>
        </div>
      </div>
    </div>
  );
}
