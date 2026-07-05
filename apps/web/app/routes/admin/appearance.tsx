import { useEffect, useState } from "react";

import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Input } from "../../components/ui/Input";
import { Heading, Text } from "../../components/ui/Text";
import {
  type HeroSettings,
  type SaveHeroSettingsInput,
  getHeroSettings,
  saveHeroSettings,
} from "../../lib/api/admin-appearance";

export const handle = { title: "Appearance" };

type FormState = {
  eyebrow: string;
  heading: string;
  subtext: string;
  cta_primary_label: string;
  cta_primary_url: string;
  cta_secondary_label: string;
  cta_secondary_url: string;
};

function settingsToForm(s: HeroSettings): FormState {
  return {
    eyebrow: s.eyebrow,
    heading: s.heading,
    subtext: s.subtext,
    cta_primary_label: s.cta_primary_label,
    cta_primary_url: s.cta_primary_url,
    cta_secondary_label: s.cta_secondary_label ?? "",
    cta_secondary_url: s.cta_secondary_url ?? "",
  };
}

const emptyForm: FormState = {
  eyebrow: "",
  heading: "",
  subtext: "",
  cta_primary_label: "",
  cta_primary_url: "",
  cta_secondary_label: "",
  cta_secondary_url: "",
};

export default function AdminAppearance() {
  const [form, setForm] = useState<FormState>(emptyForm);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getHeroSettings()
      .then((s) => setForm(settingsToForm(s)))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, []);

  function set(field: keyof FormState, value: string) {
    setForm((prev) => ({ ...prev, [field]: value }));
    setSuccess(false);
    setError(null);
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setSuccess(false);
    setError(null);
    try {
      const input: SaveHeroSettingsInput = {
        eyebrow: form.eyebrow,
        heading: form.heading,
        subtext: form.subtext,
        cta_primary_label: form.cta_primary_label,
        cta_primary_url: form.cta_primary_url,
        cta_secondary_label: form.cta_secondary_label || undefined,
        cta_secondary_url: form.cta_secondary_url || undefined,
      };
      const saved = await saveHeroSettings(input);
      setForm(settingsToForm(saved));
      setSuccess(true);
    } catch {
      setError("Failed to save appearance settings.");
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <Heading as="h1" size="xl">
        Appearance
      </Heading>

      <form onSubmit={(e) => void handleSubmit(e)} className="flex flex-col gap-6">
        <Card className="flex flex-col gap-5 p-6">
          <Heading as="h2" size="sm">
            Hero Banner
          </Heading>

          <FormField label="Eyebrow Text">
            <Input
              value={form.eyebrow}
              onChange={(e) => set("eyebrow", e.target.value)}
              placeholder="e.g. New Season"
            />
          </FormField>

          <FormField label="Heading">
            <Input
              value={form.heading}
              onChange={(e) => set("heading", e.target.value)}
              placeholder="e.g. Quietly considered style, for every day."
            />
          </FormField>

          <FormField label="Subtext">
            <textarea
              value={form.subtext}
              onChange={(e) => set("subtext", e.target.value)}
              rows={3}
              placeholder="Supporting text shown below the heading."
              className="w-full rounded-sm border border-stone-300 bg-white px-3.5 py-2.5 text-sm text-stone-900 placeholder:text-stone-400 transition-colors focus:border-stone-900 focus:outline-none disabled:cursor-not-allowed disabled:bg-stone-50 disabled:text-stone-400 resize-none"
            />
          </FormField>

          <div className="flex flex-col gap-2">
            <Text size="sm" className="font-medium text-stone-700">
              Primary CTA Button
            </Text>
            <div className="grid grid-cols-2 gap-3">
              <FormField label="Button Label">
                <Input
                  value={form.cta_primary_label}
                  onChange={(e) => set("cta_primary_label", e.target.value)}
                  placeholder="e.g. Shop All Items"
                />
              </FormField>
              <FormField label="Button URL">
                <Input
                  value={form.cta_primary_url}
                  onChange={(e) => set("cta_primary_url", e.target.value)}
                  placeholder="e.g. /shop"
                />
              </FormField>
            </div>
          </div>

          <div className="flex flex-col gap-2">
            <Text size="sm" className="font-medium text-stone-700">
              Secondary CTA Button
            </Text>
            <div className="grid grid-cols-2 gap-3">
              <FormField label="Button Label" hint="Leave blank to hide">
                <Input
                  value={form.cta_secondary_label}
                  onChange={(e) => set("cta_secondary_label", e.target.value)}
                  placeholder="e.g. View the Sale"
                />
              </FormField>
              <FormField label="Button URL" hint="Leave blank to hide">
                <Input
                  value={form.cta_secondary_url}
                  onChange={(e) => set("cta_secondary_url", e.target.value)}
                  placeholder="e.g. /shop?sale=true"
                />
              </FormField>
            </div>
          </div>
        </Card>

        {error && (
          <Text size="sm" tone="danger">
            {error}
          </Text>
        )}
        {success && (
          <Text size="sm" className="text-sage-600">
            Appearance settings saved.
          </Text>
        )}

        <div className="flex justify-end">
          <Button type="submit" disabled={saving}>
            {saving ? "Saving…" : "Save Changes"}
          </Button>
        </div>
      </form>
    </div>
  );
}
