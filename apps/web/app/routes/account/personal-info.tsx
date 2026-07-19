import { useEffect, useState } from "react";

import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Input } from "../../components/ui/Input";
import { Eyebrow, Text } from "../../components/ui/Text";
import { useAuth } from "../../features/auth/AuthContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { type Profile, getProfile, updateProfile } from "../../lib/api/users";

export const handle = { title: "Personal Info" };

export default function PersonalInfo() {
  const { t } = useLanguage();
  const { refreshProfile } = useAuth();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [fullName, setFullName] = useState("");
  const [phone, setPhone] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    getProfile()
      .then((loaded) => {
        setProfile(loaded);
        setFullName(loaded.full_name);
        setPhone(loaded.phone);
      })
      .catch(() => setError(t("account.profile.load_error", "Could not load your profile.")));
  }, []);

  async function handleSave() {
    if (!phone.trim()) {
      setError(t("account.profile.phone_required", "A phone number is required so we can reach you about your orders."));
      setSaved(false);
      return;
    }
    setIsSaving(true);
    setError(null);
    setSaved(false);
    try {
      const updated = await updateProfile({ full_name: fullName, phone });
      setProfile(updated);
      await refreshProfile();
      setSaved(true);
    } catch {
      setError(t("account.profile.save_error", "Could not save changes."));
    } finally {
      setIsSaving(false);
    }
  }

  if (!profile) {
    return (
      <Text size="sm" tone="muted">
        {error ?? t("common.loading", "Loading…")}
      </Text>
    );
  }

  return (
    <div className="flex max-w-2xl flex-col gap-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          {saved && (
            <Text size="sm" tone="muted">
              {t("account.profile.saved", "Saved")}
            </Text>
          )}
          <Button variant="primary" onClick={handleSave} disabled={isSaving}>
            {isSaving ? t("common.saving", "Saving…") : t("account.profile.save_changes", "Save Changes")}
          </Button>
        </div>
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      <section>
        <Eyebrow>{t("account.profile.your_details", "Your Details")}</Eyebrow>
        <Card className="mt-3 p-6">
          <div className="flex flex-col gap-4">
            <FormField label={t("common.email", "Email")} htmlFor="email" hint={t("account.profile.email_hint", "Managed via your Google sign-in")}>
              <Input id="email" value={profile.email} disabled />
            </FormField>
            <FormField label={t("common.full_name", "Full name")} htmlFor="full-name">
              <Input id="full-name" value={fullName} onChange={(e) => setFullName(e.target.value)} />
            </FormField>
            <FormField label={t("common.phone", "Phone")} htmlFor="phone">
              <Input id="phone" type="tel" required value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="+1 555 123 4567" />
            </FormField>
          </div>
        </Card>
      </section>
    </div>
  );
}
