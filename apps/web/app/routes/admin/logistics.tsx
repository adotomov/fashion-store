import { useEffect, useState } from "react";

import { EmptyState } from "../../components/admin/EmptyState";
import { Accordion } from "../../components/ui/Accordion";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Input } from "../../components/ui/Input";
import { Select } from "../../components/ui/Select";
import { Toggle } from "../../components/ui/Toggle";
import { Heading, Text } from "../../components/ui/Text";
import { type LogisticsProvider, listProviders, saveProvider } from "../../lib/api/admin-logistics";

export const handle = { title: "Logistics" };

export default function AdminLogistics() {
  const [providers, setProviders] = useState<LogisticsProvider[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listProviders()
      .then(setProviders)
      .catch(() => setError("Could not load logistics providers."));
  }, []);

  if (error) {
    return (
      <Text size="sm" tone="danger">
        {error}
      </Text>
    );
  }

  if (providers === null) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  if (providers.length === 0) {
    return (
      <EmptyState
        icon="shipping"
        title="No logistics providers available"
        description="Provider integrations will appear here once they're added to the backend."
      />
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {providers.map((provider) => (
        <ProviderCard
          key={provider.provider}
          provider={provider}
          onSaved={(updated) =>
            setProviders((prev) => prev?.map((p) => (p.provider === updated.provider ? updated : p)) ?? prev)
          }
        />
      ))}
    </div>
  );
}

function ProviderCard({
  provider,
  onSaved,
}: {
  provider: LogisticsProvider;
  onSaved: (updated: LogisticsProvider) => void;
}) {
  const [enabled, setEnabled] = useState(provider.enabled);
  const [config, setConfig] = useState(provider.config);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  function update(key: string, value: string) {
    setConfig((c) => ({ ...c, [key]: value }));
    setSaved(false);
  }

  async function handleSave() {
    setIsSaving(true);
    setError(null);
    try {
      const updated = await saveProvider(provider.provider, { enabled, config });
      onSaved(updated);
      setConfig(updated.config);
      setSaved(true);
    } catch {
      setError("Could not save logistics settings.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <Card className="overflow-hidden">
      <div className="flex items-center justify-between px-6 py-5">
        <div>
          <Heading as="h3" size="sm">
            {provider.name}
          </Heading>
          <Text size="sm" tone="muted" className="mt-1">
            {enabled ? "Offered at checkout." : "Disabled — hidden from checkout."}
          </Text>
        </div>
        <Toggle checked={enabled} onChange={setEnabled} aria-label={`Enable ${provider.name}`} />
      </div>

      <Accordion open={enabled}>
        {provider.provider === "speedy" ? (
          <SpeedyConfigForm config={config} onChange={update} />
        ) : (
          <Text size="sm" tone="muted">
            No configuration needed for this provider.
          </Text>
        )}

        {error && (
          <Text size="sm" tone="danger" className="mt-4">
            {error}
          </Text>
        )}
        {saved && !error && (
          <Text size="sm" tone="accent" className="mt-4">
            Settings saved.
          </Text>
        )}

        <div className="mt-6 flex justify-end">
          <Button variant="primary" onClick={handleSave} disabled={isSaving}>
            {isSaving ? "Saving…" : "Save settings"}
          </Button>
        </div>
      </Accordion>
    </Card>
  );
}

function SpeedyConfigForm({
  config,
  onChange,
}: {
  config: Record<string, string>;
  onChange: (key: string, value: string) => void;
}) {
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
      <FormField label="Username" htmlFor="speedy-username">
        <Input
          id="speedy-username"
          value={config.username ?? ""}
          onChange={(e) => onChange("username", e.target.value)}
        />
      </FormField>
      <FormField label="Password" htmlFor="speedy-password" hint="Leave unchanged to keep the current password.">
        <Input
          id="speedy-password"
          type="password"
          value={config.password ?? ""}
          onChange={(e) => onChange("password", e.target.value)}
        />
      </FormField>
      <FormField label="Language" htmlFor="speedy-language">
        <Select id="speedy-language" value={config.language ?? "EN"} onChange={(e) => onChange("language", e.target.value)}>
          <option value="EN">English</option>
          <option value="BG">Bulgarian</option>
        </Select>
      </FormField>
      <FormField label="Client system ID" htmlFor="speedy-client-system-id" hint="Optional">
        <Input
          id="speedy-client-system-id"
          value={config.client_system_id ?? ""}
          onChange={(e) => onChange("client_system_id", e.target.value)}
        />
      </FormField>
      <FormField
        label="Courier service ID"
        htmlFor="speedy-courier-service-id"
        hint="Used for door-delivery (Speedy Courier) shipments"
      >
        <Input
          id="speedy-courier-service-id"
          value={config.default_courier_service_id ?? ""}
          onChange={(e) => onChange("default_courier_service_id", e.target.value)}
        />
      </FormField>
      <FormField
        label="Locker service ID"
        htmlFor="speedy-locker-service-id"
        hint="Used for EasyBox locker shipments"
      >
        <Input
          id="speedy-locker-service-id"
          value={config.default_locker_service_id ?? ""}
          onChange={(e) => onChange("default_locker_service_id", e.target.value)}
        />
      </FormField>
      <FormField
        label="Default parcel weight (kg)"
        htmlFor="speedy-default-weight"
        hint="Used since the catalog doesn't track product weight"
        className="sm:col-span-2"
      >
        <Input
          id="speedy-default-weight"
          type="number"
          min="0"
          step="0.1"
          value={config.default_parcel_weight_kg ?? ""}
          onChange={(e) => onChange("default_parcel_weight_kg", e.target.value)}
        />
      </FormField>
    </div>
  );
}
