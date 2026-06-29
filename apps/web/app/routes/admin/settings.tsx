import { useEffect, useRef, useState } from "react";

import { useStoreBranding } from "../../features/store-settings/StoreSettingsContext";
import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Icon } from "../../components/ui/Icon";
import { Input } from "../../components/ui/Input";
import { Modal } from "../../components/ui/Modal";
import { Select } from "../../components/ui/Select";
import { Tabs } from "../../components/ui/Tabs";
import { Textarea } from "../../components/ui/Textarea";
import { Eyebrow, Text } from "../../components/ui/Text";
import {
  type Language,
  addLanguage,
  deleteLanguage,
  listLanguages,
  setLanguageEnabled,
} from "../../lib/api/languages";
import {
  type StoreAddress,
  type UpsertStoreAddressInput,
  createStoreAddress,
  deleteStoreAddress,
  listStoreAddresses,
  updateStoreAddress,
} from "../../lib/api/store-addresses";
import {
  type DocumentType,
  type StoreDocument,
  deleteStoreDocument,
  listStoreDocuments,
  uploadStoreDocument,
} from "../../lib/api/store-documents";
import {
  type StoreSettings,
  deleteStoreLogo,
  getStoreSettings,
  loadStoreLogoBlobUrl,
  updateStoreSettings,
  uploadStoreLogo,
} from "../../lib/api/store-settings";

export const handle = { title: "Store Settings" };

const TABS = [
  { id: "identity", label: "Identity" },
  { id: "contacts", label: "Contacts" },
  { id: "legal", label: "Legal Documents" },
  { id: "languages", label: "Store Language" },
];

export default function AdminSettings() {
  const [activeTab, setActiveTab] = useState("identity");

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <Tabs tabs={TABS} activeTab={activeTab} onChange={setActiveTab}>
        {activeTab === "identity" && <IdentityTab />}
        {activeTab === "contacts" && <ContactsTab />}
        {activeTab === "legal" && <LegalDocumentsTab />}
        {activeTab === "languages" && <LanguagesTab />}
      </Tabs>
    </div>
  );
}

// ---------------------------------------------------------------------------
// Identity: name, legal entity, logo, locale/currency, company description.
// ---------------------------------------------------------------------------

function IdentityTab() {
  const { refresh: refreshBranding } = useStoreBranding();
  const [settings, setSettings] = useState<StoreSettings | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [logoPreview, setLogoPreview] = useState<string | null>(null);
  const [isLogoBusy, setIsLogoBusy] = useState(false);
  const [logoError, setLogoError] = useState<string | null>(null);
  const logoInputRef = useRef<HTMLInputElement>(null);

  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  async function refresh() {
    try {
      setSettings(await getStoreSettings());
    } catch {
      setLoadError("Could not load store settings.");
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  useEffect(() => {
    if (!settings?.logo_url) {
      setLogoPreview(null);
      return;
    }
    let cancelled = false;
    let url: string | null = null;
    loadStoreLogoBlobUrl()
      .then((loaded) => {
        if (cancelled) return;
        url = loaded;
        setLogoPreview(loaded);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
      if (url) URL.revokeObjectURL(url);
    };
  }, [settings?.logo_url]);

  async function handleSave(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!settings) return;
    setIsSaving(true);
    setSaveError(null);
    try {
      const updated = await updateStoreSettings({
        store_name: settings.store_name,
        legal_entity_name: settings.legal_entity_name ?? "",
        locale: settings.locale,
        currency: settings.currency,
        company_description: settings.company_description ?? "",
      });
      setSettings(updated);
      setSavedAt(Date.now());
      refreshBranding();
    } catch {
      setSaveError("Could not save changes. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  function field<K extends keyof StoreSettings>(key: K) {
    return {
      value: (settings?.[key] as string) ?? "",
      onChange: (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) =>
        setSettings((s) => (s ? { ...s, [key]: e.target.value } : s)),
    };
  }

  async function handleLogoSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setIsLogoBusy(true);
    setLogoError(null);
    try {
      setSettings(await uploadStoreLogo(file));
      refreshBranding();
    } catch {
      setLogoError("Could not upload logo.");
    } finally {
      setIsLogoBusy(false);
      if (logoInputRef.current) logoInputRef.current.value = "";
    }
  }

  async function handleLogoRemove() {
    setIsLogoBusy(true);
    setLogoError(null);
    try {
      setSettings(await deleteStoreLogo());
      refreshBranding();
    } catch {
      setLogoError("Could not remove logo.");
    } finally {
      setIsLogoBusy(false);
    }
  }

  if (loadError) {
    return (
      <Text size="sm" tone="danger">
        {loadError}
      </Text>
    );
  }

  if (!settings) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <form className="flex flex-col gap-8" onSubmit={handleSave}>
      <section>
        <Eyebrow>Store Identity</Eyebrow>
        <Card className="mt-3 p-6">
          <div className="flex flex-col gap-4">
            <FormField label="Store name" htmlFor="store-name">
              <Input id="store-name" placeholder="Maison" {...field("store_name")} />
            </FormField>

            <FormField label="Legal entity name" htmlFor="legal-entity-name" hint="Used on invoices and legal documents">
              <Input id="legal-entity-name" placeholder="Maison Retail Ltd." {...field("legal_entity_name")} />
            </FormField>

            <FormField label="Logo" htmlFor="store-logo" hint="PNG, SVG, or JPEG" error={logoError ?? undefined}>
              <div className="flex items-center gap-4">
                <div className="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-sm border border-dashed border-stone-300 bg-stone-50">
                  {logoPreview ? (
                    <img src={logoPreview} alt="Logo preview" className="h-full w-full object-contain" />
                  ) : (
                    <Icon name="catalog" size={20} className="text-stone-400" />
                  )}
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    type="button"
                    disabled={isLogoBusy}
                    onClick={() => logoInputRef.current?.click()}
                  >
                    {isLogoBusy ? "Uploading…" : logoPreview ? "Replace" : "Upload"}
                  </Button>
                  {logoPreview && (
                    <Button
                      variant="ghost"
                      size="sm"
                      type="button"
                      disabled={isLogoBusy}
                      onClick={handleLogoRemove}
                      className="text-danger-600 hover:bg-danger-50"
                    >
                      Remove
                    </Button>
                  )}
                </div>
                <input
                  ref={logoInputRef}
                  id="store-logo"
                  type="file"
                  accept="image/png,image/svg+xml,image/jpeg"
                  onChange={handleLogoSelected}
                  disabled={isLogoBusy}
                  className="hidden"
                />
              </div>
            </FormField>

            <FormField label="Company description" htmlFor="company-description" hint="Shown on the public About page">
              <Textarea
                id="company-description"
                placeholder="Clothing, jewelry, bags, and accessories, thoughtfully made and delivered with care."
                {...field("company_description")}
              />
            </FormField>
          </div>
        </Card>
      </section>

      <section>
        <Eyebrow>Localization</Eyebrow>
        <Card className="mt-3 p-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Locale" htmlFor="locale">
              <Select id="locale" {...field("locale")}>
                <option value="en-US">English (United States)</option>
                <option value="en-GB">English (United Kingdom)</option>
                <option value="bg-BG">Bulgarian (Bulgaria)</option>
                <option value="de-DE">German (Germany)</option>
              </Select>
            </FormField>
            <FormField label="Currency" htmlFor="currency">
              <Select id="currency" {...field("currency")}>
                <option value="EUR">EUR — Euro</option>
                <option value="USD">USD — US Dollar</option>
                <option value="BGN">BGN — Bulgarian Lev</option>
                <option value="GBP">GBP — British Pound</option>
              </Select>
            </FormField>
          </div>
        </Card>
      </section>

      <div className="flex items-center gap-3">
        <Button type="submit" variant="primary" disabled={isSaving}>
          {isSaving ? "Saving…" : "Save Changes"}
        </Button>
        {saveError && (
          <Text size="xs" tone="danger">
            {saveError}
          </Text>
        )}
        {!saveError && savedAt && (
          <Text size="xs" tone="muted">
            Saved.
          </Text>
        )}
      </div>
    </form>
  );
}

// ---------------------------------------------------------------------------
// Contacts: company contact details + multi-location addresses.
// ---------------------------------------------------------------------------

function ContactsTab() {
  const [settings, setSettings] = useState<StoreSettings | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<number | null>(null);

  const [addresses, setAddresses] = useState<StoreAddress[]>([]);
  const [addressModalOpen, setAddressModalOpen] = useState(false);
  const [editingAddress, setEditingAddress] = useState<StoreAddress | null>(null);

  useEffect(() => {
    getStoreSettings().then(setSettings).catch(() => {});
    listStoreAddresses().then(setAddresses).catch(() => {});
  }, []);

  function field<K extends keyof StoreSettings>(key: K) {
    return {
      value: (settings?.[key] as string) ?? "",
      onChange: (e: React.ChangeEvent<HTMLInputElement>) =>
        setSettings((s) => (s ? { ...s, [key]: e.target.value } : s)),
    };
  }

  async function handleSave(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!settings) return;
    setIsSaving(true);
    setSaveError(null);
    try {
      setSettings(
        await updateStoreSettings({
          contact_email: settings.contact_email ?? "",
          contact_phone: settings.contact_phone ?? "",
        }),
      );
      setSavedAt(Date.now());
    } catch {
      setSaveError("Could not save changes. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  async function refreshAddresses() {
    setAddresses(await listStoreAddresses());
  }

  async function handleDeleteAddress(id: string) {
    await deleteStoreAddress(id);
    await refreshAddresses();
  }

  if (!settings) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      <form className="flex flex-col gap-8" onSubmit={handleSave}>
        <section>
          <Eyebrow>Contact Details</Eyebrow>
          <Card className="mt-3 p-6">
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <FormField label="Contact email" htmlFor="contactEmail">
                <Input id="contactEmail" type="email" placeholder="hello@maison.example" {...field("contact_email")} />
              </FormField>
              <FormField label="Contact phone" htmlFor="contactPhone">
                <Input id="contactPhone" type="tel" placeholder="+359 2 123 4567" {...field("contact_phone")} />
              </FormField>
            </div>
          </Card>
        </section>

        <div className="flex items-center gap-3">
          <Button type="submit" variant="primary" disabled={isSaving}>
            {isSaving ? "Saving…" : "Save Changes"}
          </Button>
          {saveError && (
            <Text size="xs" tone="danger">
              {saveError}
            </Text>
          )}
          {!saveError && savedAt && (
            <Text size="xs" tone="muted">
              Saved.
            </Text>
          )}
        </div>
      </form>

      <section>
        <div className="flex items-center justify-between">
          <Eyebrow>Store Addresses</Eyebrow>
          <Button
            variant="outline"
            size="sm"
            type="button"
            onClick={() => {
              setEditingAddress(null);
              setAddressModalOpen(true);
            }}
          >
            Add Address
          </Button>
        </div>
        <Card className="mt-3 p-6">
          {addresses.length === 0 ? (
            <Text size="sm" tone="muted">
              No addresses yet.
            </Text>
          ) : (
            <ul className="flex flex-col gap-4">
              {addresses.map((address) => (
                <li key={address.id} className="flex items-start justify-between gap-4 border-b border-stone-100 pb-4 last:border-0 last:pb-0">
                  <div>
                    <div className="flex items-center gap-2">
                      <Text size="sm" className="font-medium">
                        {address.label || "Address"}
                      </Text>
                      {address.is_default && <Badge variant="brand">Default</Badge>}
                    </div>
                    <Text size="sm" tone="muted" className="mt-1">
                      {[address.line1, address.line2, address.city, address.region, address.postal_code, address.country]
                        .filter(Boolean)
                        .join(", ")}
                    </Text>
                  </div>
                  <div className="flex shrink-0 gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      type="button"
                      onClick={() => {
                        setEditingAddress(address);
                        setAddressModalOpen(true);
                      }}
                    >
                      Edit
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      type="button"
                      onClick={() => handleDeleteAddress(address.id)}
                      className="text-danger-600 hover:bg-danger-50"
                    >
                      Delete
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </section>

      <AddressModal
        open={addressModalOpen}
        address={editingAddress}
        onClose={() => setAddressModalOpen(false)}
        onSaved={async () => {
          setAddressModalOpen(false);
          await refreshAddresses();
        }}
      />
    </div>
  );
}

function AddressModal({
  open,
  address,
  onClose,
  onSaved,
}: {
  open: boolean;
  address: StoreAddress | null;
  onClose: () => void;
  onSaved: () => void;
}) {
  const [form, setForm] = useState<UpsertStoreAddressInput>(emptyAddressForm());
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (open) {
      setForm(address ? { ...address } : emptyAddressForm());
      setError(null);
    }
  }, [open, address]);

  function update<K extends keyof UpsertStoreAddressInput>(key: K, value: UpsertStoreAddressInput[K]) {
    setForm((f) => ({ ...f, [key]: value }));
  }

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setIsSaving(true);
    setError(null);
    try {
      if (address) {
        await updateStoreAddress(address.id, form);
      } else {
        await createStoreAddress(form);
      }
      onSaved();
    } catch {
      setError("Could not save address.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={address ? "Edit Address" : "Add Address"}>
      <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
        <FormField label="Label" htmlFor="address-label">
          <Input id="address-label" placeholder="Main Store" value={form.label} onChange={(e) => update("label", e.target.value)} />
        </FormField>
        <FormField label="Address line 1" htmlFor="address-line1">
          <Input id="address-line1" value={form.line1} onChange={(e) => update("line1", e.target.value)} />
        </FormField>
        <FormField label="Address line 2" htmlFor="address-line2">
          <Input id="address-line2" value={form.line2 ?? ""} onChange={(e) => update("line2", e.target.value)} />
        </FormField>
        <div className="grid grid-cols-2 gap-4">
          <FormField label="City" htmlFor="address-city">
            <Input id="address-city" value={form.city ?? ""} onChange={(e) => update("city", e.target.value)} />
          </FormField>
          <FormField label="Region" htmlFor="address-region">
            <Input id="address-region" value={form.region ?? ""} onChange={(e) => update("region", e.target.value)} />
          </FormField>
          <FormField label="Postal code" htmlFor="address-postal">
            <Input id="address-postal" value={form.postal_code ?? ""} onChange={(e) => update("postal_code", e.target.value)} />
          </FormField>
          <FormField label="Country" htmlFor="address-country">
            <Input id="address-country" value={form.country ?? ""} onChange={(e) => update("country", e.target.value)} />
          </FormField>
        </div>
        <label className="flex items-center gap-2 text-sm text-stone-700">
          <input type="checkbox" checked={form.is_default} onChange={(e) => update("is_default", e.target.checked)} />
          Set as default address
        </label>

        {error && (
          <Text size="xs" tone="danger">
            {error}
          </Text>
        )}

        <div className="mt-2 flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={isSaving}>
            {isSaving ? "Saving…" : "Save"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function emptyAddressForm(): UpsertStoreAddressInput {
  return { label: "", line1: "", line2: "", city: "", region: "", postal_code: "", country: "", is_default: false };
}

// ---------------------------------------------------------------------------
// Legal Documents: per-language uploads for Terms of Service / Privacy Policy.
// ---------------------------------------------------------------------------

function LegalDocumentsTab() {
  return (
    <div className="flex flex-col gap-8">
      <DocumentSection type="terms" title="Terms of Service" />
      <DocumentSection type="privacy" title="Privacy Policy" />
    </div>
  );
}

function DocumentSection({ type, title }: { type: DocumentType; title: string }) {
  const [docs, setDocs] = useState<StoreDocument[]>([]);
  const [modalOpen, setModalOpen] = useState(false);

  async function refresh() {
    setDocs(await listStoreDocuments(type));
  }

  useEffect(() => {
    refresh();
  }, []);

  async function handleDelete(locale: string) {
    await deleteStoreDocument(type, locale);
    await refresh();
  }

  return (
    <section>
      <div className="flex items-center justify-between">
        <Eyebrow>{title}</Eyebrow>
        <Button variant="outline" size="sm" type="button" onClick={() => setModalOpen(true)}>
          Upload
        </Button>
      </div>
      <Card className="mt-3 p-6">
        {docs.length === 0 ? (
          <Text size="sm" tone="muted">
            No document uploaded yet.
          </Text>
        ) : (
          <ul className="flex flex-col gap-3">
            {docs.map((doc) => (
              <li key={doc.locale} className="flex items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <Badge variant="neutral">{doc.locale.toUpperCase()}</Badge>
                  <Text size="sm">{doc.filename}</Text>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  type="button"
                  onClick={() => handleDelete(doc.locale)}
                  className="text-danger-600 hover:bg-danger-50"
                >
                  Remove
                </Button>
              </li>
            ))}
          </ul>
        )}
      </Card>

      <UploadDocumentModal
        open={modalOpen}
        type={type}
        onClose={() => setModalOpen(false)}
        onUploaded={async () => {
          setModalOpen(false);
          await refresh();
        }}
      />
    </section>
  );
}

function UploadDocumentModal({
  open,
  type,
  onClose,
  onUploaded,
}: {
  open: boolean;
  type: DocumentType;
  onClose: () => void;
  onUploaded: () => void;
}) {
  const [languages, setLanguages] = useState<Language[]>([]);
  const [locale, setLocale] = useState("en");
  const [file, setFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (open) {
      listLanguages().then(setLanguages).catch(() => {});
      setFile(null);
      setError(null);
    }
  }, [open]);

  async function handleSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!file) {
      setError("Choose a file to upload.");
      return;
    }
    setIsUploading(true);
    setError(null);
    try {
      await uploadStoreDocument(type, locale, file);
      onUploaded();
    } catch {
      setError("Could not upload document.");
    } finally {
      setIsUploading(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Upload Document">
      <form className="flex flex-col gap-4" onSubmit={handleSubmit}>
        <FormField label="Language" htmlFor="document-locale">
          <Select id="document-locale" value={locale} onChange={(e) => setLocale(e.target.value)}>
            {languages.map((lang) => (
              <option key={lang.code} value={lang.code}>
                {lang.name}
              </option>
            ))}
          </Select>
        </FormField>

        <FormField label="File" htmlFor="document-file" hint="PDF or Word document" error={error ?? undefined}>
          <input
            ref={fileInputRef}
            id="document-file"
            type="file"
            accept="application/pdf,.doc,.docx,application/msword,application/vnd.openxmlformats-officedocument.wordprocessingml.document"
            onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          />
        </FormField>

        <div className="mt-2 flex justify-end gap-2">
          <Button type="button" variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={isUploading}>
            {isUploading ? "Uploading…" : "Upload"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

// ---------------------------------------------------------------------------
// Store Language: add/enable/disable/remove languages.
// ---------------------------------------------------------------------------

function LanguagesTab() {
  const [languages, setLanguages] = useState<Language[]>([]);
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [isAdding, setIsAdding] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    setLanguages(await listLanguages());
  }

  useEffect(() => {
    refresh();
  }, []);

  async function handleAdd(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setIsAdding(true);
    setError(null);
    try {
      await addLanguage(code.trim().toLowerCase(), name.trim());
      setCode("");
      setName("");
      await refresh();
    } catch {
      setError("Could not add language — check the code isn't already in use.");
    } finally {
      setIsAdding(false);
    }
  }

  async function handleToggle(lang: Language) {
    await setLanguageEnabled(lang.code, !lang.enabled);
    await refresh();
  }

  async function handleDelete(lang: Language) {
    await deleteLanguage(lang.code);
    await refresh();
  }

  return (
    <div className="flex flex-col gap-8">
      <section>
        <Eyebrow>Languages</Eyebrow>
        <Card className="mt-3 p-6">
          {languages.length === 0 ? (
            <Text size="sm" tone="muted">
              Loading…
            </Text>
          ) : (
            <ul className="flex flex-col gap-3">
              {languages.map((lang) => (
                <li key={lang.code} className="flex items-center justify-between gap-3 border-b border-stone-100 pb-3 last:border-0 last:pb-0">
                  <div className="flex items-center gap-2">
                    <Text size="sm" className="font-medium">
                      {lang.name}
                    </Text>
                    <Badge variant="neutral">{lang.code.toUpperCase()}</Badge>
                    {lang.is_default && <Badge variant="brand">Default</Badge>}
                  </div>
                  <div className="flex items-center gap-2">
                    {!lang.is_default && (
                      <Button variant="ghost" size="sm" type="button" onClick={() => handleToggle(lang)}>
                        {lang.enabled ? "Disable" : "Enable"}
                      </Button>
                    )}
                    {!lang.is_default && (
                      <Button
                        variant="ghost"
                        size="sm"
                        type="button"
                        onClick={() => handleDelete(lang)}
                        className="text-danger-600 hover:bg-danger-50"
                      >
                        Remove
                      </Button>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </section>

      <section>
        <Eyebrow>Add a Language</Eyebrow>
        <Card className="mt-3 p-6">
          <form className="flex flex-col gap-4 sm:flex-row sm:items-end" onSubmit={handleAdd}>
            <FormField label="Code" htmlFor="lang-code" className="sm:w-32">
              <Input id="lang-code" value={code} onChange={(e) => setCode(e.target.value)} placeholder="bg" />
            </FormField>
            <FormField label="Name" htmlFor="lang-name" className="flex-1">
              <Input id="lang-name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Bulgarian" />
            </FormField>
            <Button type="submit" variant="primary" disabled={isAdding || !code.trim() || !name.trim()}>
              {isAdding ? "Adding…" : "Add Language"}
            </Button>
          </form>
          <Text size="xs" tone="muted" className="mt-2">
            Code must be ISO 639-1, e.g. bg, de.
          </Text>
          {error && (
            <Text size="xs" tone="danger" className="mt-3">
              {error}
            </Text>
          )}
        </Card>
      </section>

      <Text size="sm" tone="muted">
        Once a second language is added, translatable fields across the catalog and store settings show extra inputs for
        that language. Manage static page text in the <strong>Translations</strong> page.
      </Text>
    </div>
  );
}
