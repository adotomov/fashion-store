import { useState, type FormEvent } from "react";

import { Button } from "../../components/ui/Button";
import { FormField } from "../../components/ui/FormField";
import { Input } from "../../components/ui/Input";
import { Heading, Text } from "../../components/ui/Text";
import { useLanguage } from "../i18n/LanguageContext";
import { useAuth } from "./AuthContext";

/**
 * One-time, non-dismissible gate shown right after a first-time Google
 * registration to capture a phone number before the shopper enters the app.
 * Google sign-in never carries a phone, so a fresh account has none to enter;
 * an account previously created via guest checkout may already have one to
 * confirm (and edit). Either way we require a number before continuing.
 */
export function PhoneSetupGate() {
  const { t } = useLanguage();
  const { profile, completePhoneSetup, logout } = useAuth();
  const hadPhone = Boolean(profile?.phone?.trim());
  const [phone, setPhone] = useState(profile?.phone ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const title = hadPhone
    ? t("auth.phone_setup.confirm_title", "Confirm your phone number")
    : t("auth.phone_setup.add_title", "Add your phone number");

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    if (!phone.trim()) {
      setError(t("auth.phone_setup.required", "Please enter a phone number to continue."));
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await completePhoneSetup(phone.trim());
    } catch {
      setError(t("auth.phone_setup.save_error", "Could not save your phone number. Please try again."));
      setSaving(false);
    }
  }

  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-stone-900/50" aria-hidden="true" />

      <div
        role="dialog"
        aria-modal="true"
        aria-label={title}
        className="relative w-full max-w-md rounded-sm bg-white p-6 shadow-xl sm:p-8"
      >
        <Heading as="h2" size="md">
          {title}
        </Heading>
        <Text tone="muted" className="mt-2">
          {hadPhone
            ? t(
                "auth.phone_setup.confirm_desc",
                "We use this to reach you about your orders and deliveries. Is this number correct? Edit it below if not.",
              )
            : t(
                "auth.phone_setup.add_desc",
                "We need a phone number to reach you about your orders and deliveries.",
              )}
        </Text>

        <form onSubmit={handleSubmit} className="mt-6 flex flex-col gap-4">
          <FormField label={t("common.phone", "Phone")} htmlFor="setup-phone">
            <Input
              id="setup-phone"
              type="tel"
              required
              autoFocus
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="+1 555 123 4567"
            />
          </FormField>

          {error && (
            <Text size="sm" tone="danger">
              {error}
            </Text>
          )}

          <Button type="submit" variant="primary" disabled={saving} className="h-11">
            {saving
              ? t("common.saving", "Saving…")
              : hadPhone
                ? t("auth.phone_setup.confirm_cta", "Confirm & Continue")
                : t("auth.phone_setup.add_cta", "Save & Continue")}
          </Button>
        </form>

        <button
          type="button"
          onClick={() => void logout()}
          className="mt-4 w-full text-center text-sm text-stone-400 underline hover:text-stone-700"
        >
          {t("auth.phone_setup.sign_out", "Not now — sign out")}
        </button>
      </div>
    </div>
  );
}
