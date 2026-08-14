import { useEffect, useState } from "react";

import { AddressForm, type StructuredAddress, emptyStructuredAddress, isStructuredAddressComplete } from "../../components/address/AddressForm";
import { EmptyState } from "../../components/admin/EmptyState";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { Checkbox } from "../../components/ui/Checkbox";
import { FormField } from "../../components/ui/FormField";
import { Icon } from "../../components/ui/Icon";
import { Input } from "../../components/ui/Input";
import { Modal } from "../../components/ui/Modal";
import { Text } from "../../components/ui/Text";
import { useAuth } from "../../features/auth/AuthContext";
import { useLanguage } from "../../features/i18n/LanguageContext";
import {
  type Address,
  type AddressInput,
  createAddress,
  deleteAddress,
  listAddresses,
  updateAddress,
} from "../../lib/api/users";

export const handle = { title: "Addresses" };

// structuredFromAddress lifts the Speedy-structured fields out of a saved
// Address into the shape the form edits.
function structuredFromAddress(a: Address): StructuredAddress {
  return {
    recipient_name: a.recipient_name,
    phone: a.phone,
    country_code: a.country_code || "BG",
    country_id: a.country_id || 100,
    site_id: a.site_id,
    city: a.city,
    post_code: a.post_code,
    complex_id: a.complex_id,
    complex_name: a.complex_name,
    street_id: a.street_id,
    street_name: a.street_name,
    street_no: a.street_no,
    block_no: a.block_no,
    entrance_no: a.entrance_no,
    floor_no: a.floor_no,
    apartment_no: a.apartment_no,
  };
}

// formatAddressLines renders a saved address for the card, skipping empties.
function formatAddressLines(a: Address): string[] {
  const street = [a.street_name, a.street_no].filter(Boolean).join(" ");
  const detail = [
    a.block_no && `bl. ${a.block_no}`,
    a.floor_no && `fl. ${a.floor_no}`,
    a.apartment_no && `ap. ${a.apartment_no}`,
  ]
    .filter(Boolean)
    .join(", ");
  const cityLine = [a.post_code, a.city].filter(Boolean).join(" ");
  return [street, a.complex_name, detail, cityLine].filter((line): line is string => Boolean(line && line.trim()));
}

export default function Addresses() {
  const { t } = useLanguage();
  const { profile } = useAuth();
  const [addresses, setAddresses] = useState<Address[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingAddress, setEditingAddress] = useState<Address | null>(null);
  const [addr, setAddr] = useState<StructuredAddress>(emptyStructuredAddress);
  const [label, setLabel] = useState("");
  const [isDefault, setIsDefault] = useState(false);
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
    setAddr(emptyStructuredAddress);
    setLabel("");
    setIsDefault(false);
    setSaveError(null);
    setIsModalOpen(true);
  }

  function openEditModal(address: Address) {
    setEditingAddress(address);
    setAddr(structuredFromAddress(address));
    setLabel(address.label);
    setIsDefault(address.is_default);
    setSaveError(null);
    setIsModalOpen(true);
  }

  async function handleSave() {
    if (!isStructuredAddressComplete(addr)) {
      setSaveError(t("account.addresses.required_error", "City, neighbourhood and street are required."));
      return;
    }
    setIsSaving(true);
    setSaveError(null);
    const payload: AddressInput = {
      ...addr,
      recipient_name: profile?.full_name ?? "",
      phone: profile?.phone ?? "",
      label,
      is_default: isDefault,
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
                {formatAddressLines(address).map((line, i) => (
                  <span key={i}>{line}</span>
                ))}
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
            <Input id="address-label" value={label} onChange={(e) => setLabel(e.target.value)} placeholder="Home" />
          </FormField>

          <AddressForm value={addr} onChange={setAddr} idPrefix="account-addr" />

          <Checkbox
            id="address-is-default"
            label={t("account.addresses.set_default", "Set as default address")}
            checked={isDefault}
            onChange={(e) => setIsDefault(e.target.checked)}
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
