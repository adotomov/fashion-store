import { useEffect, useState } from "react";

import { EmptyState } from "../../components/admin/EmptyState";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { Checkbox } from "../../components/ui/Checkbox";
import { FormField } from "../../components/ui/FormField";
import { Icon } from "../../components/ui/Icon";
import { Input } from "../../components/ui/Input";
import { Modal } from "../../components/ui/Modal";
import { Select } from "../../components/ui/Select";
import { Text } from "../../components/ui/Text";
import { useAuth } from "../../features/auth/AuthContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import { COUNTRIES } from "../../lib/data/countries";
import {
  type Address,
  type AddressInput,
  createAddress,
  deleteAddress,
  listAddresses,
  updateAddress,
} from "../../lib/api/users";

export const handle = { title: "Addresses" };

const emptyForm: AddressInput = {
  label: "",
  recipient_name: "",
  phone: "",
  line1: "",
  line2: "",
  city: "",
  region: "",
  postal_code: "",
  country_code: "",
  is_default: false,
};

export default function Addresses() {
  const { t } = useLanguage();
  const { profile } = useAuth();
  const [addresses, setAddresses] = useState<Address[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingAddress, setEditingAddress] = useState<Address | null>(null);
  const [form, setForm] = useState<AddressInput>(emptyForm);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  async function refresh() {
    try {
      setAddresses(await listAddresses());
    } catch {
      setError(t("account.addresses.load_error", "Could not load addresses."));
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  function openCreateModal() {
    setEditingAddress(null);
    setForm(emptyForm);
    setSaveError(null);
    setIsModalOpen(true);
  }

  function openEditModal(address: Address) {
    setEditingAddress(address);
    setForm({
      label: address.label,
      recipient_name: address.recipient_name,
      phone: address.phone,
      line1: address.line1,
      line2: address.line2,
      city: address.city,
      region: address.region,
      postal_code: address.postal_code,
      country_code: address.country_code,
      is_default: address.is_default,
    });
    setSaveError(null);
    setIsModalOpen(true);
  }

  function update<K extends keyof AddressInput>(key: K, value: AddressInput[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSave() {
    if (!form.line1.trim() || !form.city.trim() || !form.postal_code.trim()) {
      setSaveError(t("account.addresses.required_error", "Address line 1, city, and postal code are required."));
      return;
    }
    setIsSaving(true);
    setSaveError(null);
    const payload: AddressInput = {
      ...form,
      recipient_name: profile?.full_name ?? "",
      phone: profile?.phone ?? "",
    };
    try {
      if (editingAddress) {
        await updateAddress(editingAddress.id, payload);
      } else {
        await createAddress(payload);
      }
      setIsModalOpen(false);
      await refresh();
    } catch {
      setSaveError(editingAddress ? t("account.addresses.save_error", "Could not save changes. Try again.") : t("account.addresses.create_error", "Could not create address. Try again."));
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete(address: Address) {
    if (!window.confirm(`Delete the "${address.label || address.recipient_name}" address? This cannot be undone.`)) {
      return;
    }
    try {
      await deleteAddress(address.id);
      await refresh();
    } catch {
      setError(t("account.addresses.delete_error", "Could not delete address."));
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-end">
        <Button variant="primary" onClick={openCreateModal}>
          <Icon name="plus" size={16} />
          {t("account.addresses.add_button", "Add Address")}
        </Button>
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {addresses === null ? (
        <Text size="sm" tone="muted">
          {t("common.loading", "Loading…")}
        </Text>
      ) : addresses.length === 0 ? (
        <EmptyState icon="mapPin" title={t("account.addresses.empty_title", "No addresses yet")} description={t("account.addresses.empty_desc", "Add a shipping address to get started.")} />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {addresses.map((address) => (
            <Card key={address.id} className="p-5">
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-2">
                  <Text className="font-medium">{address.label || t("account.addresses.address_label", "Address")}</Text>
                  {address.is_default && <Badge variant="brand">{t("common.default_badge", "Default")}</Badge>}
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label={t("account.addresses.edit", "Edit address")}
                    title={t("account.addresses.edit", "Edit address")}
                    onClick={() => openEditModal(address)}
                  >
                    <Icon name="pencil" size={15} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label={t("account.addresses.delete", "Delete address")}
                    title={t("account.addresses.delete", "Delete address")}
                    onClick={() => handleDelete(address)}
                    className="text-danger-600 hover:bg-danger-50"
                  >
                    <Icon name="trash" size={15} />
                  </Button>
                </div>
              </div>
              <div className="mt-3 flex flex-col gap-0.5 text-sm text-stone-600">
                <span>{address.recipient_name}</span>
                <span>{address.line1}</span>
                {address.line2 && <span>{address.line2}</span>}
                <span>
                  {address.city}
                  {address.region ? `, ${address.region}` : ""} {address.postal_code}
                </span>
                <span>{address.country_code}</span>
                {address.phone && <span className="mt-1">{address.phone}</span>}
              </div>
            </Card>
          ))}
        </div>
      )}

      <Modal open={isModalOpen} onClose={() => setIsModalOpen(false)} title={editingAddress ? t("account.addresses.modal_edit", "Edit Address") : t("account.addresses.modal_add", "Add Address")}>
        <div className="flex flex-col gap-4">
          {saveError && (
            <Text size="sm" tone="danger">
              {saveError}
            </Text>
          )}
          <FormField label={t("account.addresses.label", "Label")} htmlFor="address-label" hint={t("account.addresses.label_hint", "Optional, e.g. Home or Office")}>
            <Input id="address-label" value={form.label} onChange={(e) => update("label", e.target.value)} placeholder="Home" />
          </FormField>
          <FormField label={t("common.address_line1", "Address line 1")} htmlFor="address-line1">
            <Input id="address-line1" value={form.line1} onChange={(e) => update("line1", e.target.value)} autoFocus />
          </FormField>
          <FormField label={t("common.address_line2", "Address line 2")} htmlFor="address-line2" hint={t("common.optional", "Optional")}>
            <Input id="address-line2" value={form.line2} onChange={(e) => update("line2", e.target.value)} />
          </FormField>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={t("common.city", "City")} htmlFor="address-city">
              <Input id="address-city" value={form.city} onChange={(e) => update("city", e.target.value)} />
            </FormField>
            <FormField label={t("common.region", "Region / State")} htmlFor="address-region" hint={t("common.optional", "Optional")}>
              <Input id="address-region" value={form.region} onChange={(e) => update("region", e.target.value)} />
            </FormField>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <FormField label={t("common.postal_code", "Postal code")} htmlFor="address-postal-code">
              <Input id="address-postal-code" value={form.postal_code} onChange={(e) => update("postal_code", e.target.value)} />
            </FormField>
            <FormField label={t("common.country", "Country")} htmlFor="address-country-code">
              <Select
                id="address-country-code"
                value={form.country_code}
                onChange={(e) => update("country_code", e.target.value)}
              >
                <option value="">{t("common.select_country", "Select a country")}</option>
                {COUNTRIES.map((country) => (
                  <option key={country.code} value={country.code}>
                    {country.name}
                  </option>
                ))}
              </Select>
            </FormField>
          </div>
          <Checkbox
            id="address-is-default"
            label={t("account.addresses.set_default", "Set as default address")}
            checked={form.is_default}
            onChange={(e) => update("is_default", e.target.checked)}
          />
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setIsModalOpen(false)} disabled={isSaving}>
            {t("common.cancel", "Cancel")}
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={isSaving}>
            {isSaving ? t("common.saving", "Saving…") : t("common.save", "Save")}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
