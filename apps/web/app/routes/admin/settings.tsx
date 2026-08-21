import { useEffect, useRef, useState } from "react";

import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

import { useAdminPermissions } from "../../features/admin/AdminPermissionsContext";
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
  setDefaultLanguage,
  setLanguageCountry,
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
  getLegalContent,
  saveLegalContent,
} from "../../lib/api/store-documents";
import {
  type StoreSettings,
  deleteAboutCover,
  deleteStoreImage,
  deleteStoreLogo,
  getStoreSettings,
  loadAboutCoverBlobUrl,
  loadStoreImageBlobUrl,
  loadStoreLogoBlobUrl,
  updateStoreSettings,
  uploadAboutCover,
  uploadStoreImage,
  uploadStoreLogo,
} from "../../lib/api/store-settings";

export const handle = { title: "Store Settings" };

const TABS = [
  { id: "identity", label: "Identity" },
  { id: "contacts", label: "Contacts" },
  { id: "about", label: "About Us" },
  { id: "legal", label: "Legal Documents" },
  { id: "faq", label: "FAQ" },
  { id: "shipping", label: "Shipping & Returns" },
  { id: "sizeguide", label: "Size Guide" },
  { id: "languages", label: "Store Language" },
];

export default function AdminSettings() {
  const [activeTab, setActiveTab] = useState("identity");

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <Tabs tabs={TABS} activeTab={activeTab} onChange={setActiveTab}>
        {activeTab === "identity" && <IdentityTab />}
        {activeTab === "contacts" && <ContactsTab />}
        {activeTab === "about" && <AboutTab />}
        {activeTab === "legal" && <LegalDocumentsTab />}
        {activeTab === "faq" && <DocumentEditor type="faq" title="Frequently Asked Questions" />}
        {activeTab === "shipping" && <DocumentEditor type="shipping" title="Shipping & Returns" />}
        {activeTab === "sizeguide" && <DocumentEditor type="size_guide" title="Size Guide" />}
        {activeTab === "languages" && <LanguagesTab />}
      </Tabs>
    </div>
  );
}

// ---------------------------------------------------------------------------
// ImageManager: reusable admin image slot (upload / replace / remove) backed by
// the store-settings singleton. Mirrors the logo handling in IdentityTab but is
// generic over which image field it manages, so About-cover and Contact
// store-image share one implementation. `version` is bumped after each mutation
// to force the preview blob to reload even when the proxy URL is unchanged.
// ---------------------------------------------------------------------------

function ImageManager({
  label,
  hint,
  hasImage,
  loadUrl,
  upload,
  remove,
  onUpdated,
  isReadOnly,
}: {
  label: string;
  hint?: string;
  hasImage: boolean;
  loadUrl: () => Promise<string>;
  upload: (file: File) => Promise<StoreSettings>;
  remove: () => Promise<StoreSettings>;
  onUpdated: (settings: StoreSettings) => void;
  isReadOnly: boolean;
}) {
  const [preview, setPreview] = useState<string | null>(null);
  const [isBusy, setIsBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [version, setVersion] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!hasImage) {
      setPreview(null);
      return;
    }
    let cancelled = false;
    let url: string | null = null;
    loadUrl()
      .then((loaded) => {
        if (cancelled) return;
        url = loaded;
        setPreview(loaded);
      })
      .catch(() => {});
    return () => {
      cancelled = true;
      if (url) URL.revokeObjectURL(url);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [hasImage, version]);

  async function handleSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    setIsBusy(true);
    setError(null);
    try {
      onUpdated(await upload(file));
      setVersion((v) => v + 1);
    } catch {
      setError("Could not upload image.");
    } finally {
      setIsBusy(false);
      if (inputRef.current) inputRef.current.value = "";
    }
  }

  async function handleRemove() {
    setIsBusy(true);
    setError(null);
    try {
      onUpdated(await remove());
      setVersion((v) => v + 1);
    } catch {
      setError("Could not remove image.");
    } finally {
      setIsBusy(false);
    }
  }

  return (
    <FormField label={label} htmlFor={`img-${label}`} hint={hint} error={error ?? undefined}>
      <div className="flex items-center gap-4">
        <div className="flex h-20 w-32 shrink-0 items-center justify-center overflow-hidden rounded-sm border border-dashed border-stone-300 bg-stone-50">
          {preview ? (
            <img src={preview} alt={`${label} preview`} className="h-full w-full object-cover" />
          ) : (
            <Icon name="catalog" size={20} className="text-stone-400" />
          )}
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            type="button"
            disabled={isBusy || isReadOnly}
            onClick={() => inputRef.current?.click()}
          >
            {isBusy ? "Uploading…" : preview ? "Replace" : "Upload"}
          </Button>
          {preview && (
            <Button
              variant="ghost"
              size="sm"
              type="button"
              disabled={isBusy || isReadOnly}
              onClick={handleRemove}
              className="text-danger-600 hover:bg-danger-50"
            >
              Remove
            </Button>
          )}
        </div>
        <input
          ref={inputRef}
          id={`img-${label}`}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          onChange={handleSelected}
          disabled={isBusy}
          className="hidden"
        />
      </div>
    </FormField>
  );
}

// ---------------------------------------------------------------------------
// About Us: cover photo (image slot) + translatable Markdown body.
// ---------------------------------------------------------------------------

function AboutTab() {
  const { isReadOnly } = useAdminPermissions();
  const [settings, setSettings] = useState<StoreSettings | null>(null);

  useEffect(() => {
    getStoreSettings().then(setSettings).catch(() => {});
  }, []);

  return (
    <div className="flex flex-col gap-8">
      <section>
        <Eyebrow>Cover Photo</Eyebrow>
        <Card className="mt-3 p-6">
          {settings ? (
            <ImageManager
              label="Cover photo"
              hint="Shown across the top of the public About page. JPEG, PNG, or WebP."
              hasImage={Boolean(settings.about_cover_url)}
              loadUrl={loadAboutCoverBlobUrl}
              upload={uploadAboutCover}
              remove={deleteAboutCover}
              onUpdated={setSettings}
              isReadOnly={isReadOnly}
            />
          ) : (
            <Text size="sm" tone="muted">
              Loading…
            </Text>
          )}
        </Card>
      </section>

      <DocumentEditor type="about" title="About Us Text" />
    </div>
  );
}

// ---------------------------------------------------------------------------
// Identity: name, legal entity, logo, locale/currency, company description.
// ---------------------------------------------------------------------------

function IdentityTab() {
  const { isReadOnly } = useAdminPermissions();
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
        facebook_url: settings.facebook_url ?? "",
        instagram_url: settings.instagram_url ?? "",
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
              <Input id="store-name" placeholder="Your store name" {...field("store_name")} />
            </FormField>

            <FormField label="Legal entity name" htmlFor="legal-entity-name" hint="Used on invoices and legal documents">
              <Input id="legal-entity-name" placeholder="Your Company Ltd." {...field("legal_entity_name")} />
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
                    disabled={isLogoBusy || isReadOnly}
                    onClick={() => logoInputRef.current?.click()}
                  >
                    {isLogoBusy ? "Uploading…" : logoPreview ? "Replace" : "Upload"}
                  </Button>
                  {logoPreview && (
                    <Button
                      variant="ghost"
                      size="sm"
                      type="button"
                      disabled={isLogoBusy || isReadOnly}
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

            <FormField label="Company description" htmlFor="company-description" hint="Short internal summary of the store. The public About page is edited under the About Us tab.">
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

      <section>
        <Eyebrow>Social Media</Eyebrow>
        <Card className="mt-3 p-6">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <FormField label="Facebook" htmlFor="facebook-url" hint="Full profile URL — leave blank to hide the icon">
              <Input id="facebook-url" type="url" placeholder="https://facebook.com/yourpage" {...field("facebook_url")} />
            </FormField>
            <FormField label="Instagram" htmlFor="instagram-url" hint="Full profile URL — leave blank to hide the icon">
              <Input id="instagram-url" type="url" placeholder="https://instagram.com/yourhandle" {...field("instagram_url")} />
            </FormField>
          </div>
        </Card>
      </section>

      <div className="flex items-center gap-3">
        <Button type="submit" variant="primary" disabled={isSaving || isReadOnly}>
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
  const { isReadOnly } = useAdminPermissions();
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
          opening_hours: settings.opening_hours ?? "",
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
                <Input id="contactEmail" type="email" placeholder="hello@example.com" {...field("contact_email")} />
              </FormField>
              <FormField label="Contact phone" htmlFor="contactPhone">
                <Input id="contactPhone" type="tel" placeholder="+359 2 123 4567" {...field("contact_phone")} />
              </FormField>
            </div>
            <div className="mt-4">
              <FormField
                label="Opening hours"
                htmlFor="openingHours"
                hint="Shown on the public Contact page — one line per day. Write in your store languages as you'd like it to appear."
              >
                <Textarea
                  id="openingHours"
                  rows={5}
                  placeholder={"Mon–Fri: 10:00–19:00\nSat: 10:00–16:00\nSun: Closed"}
                  value={settings.opening_hours ?? ""}
                  onChange={(e) => setSettings((s) => (s ? { ...s, opening_hours: e.target.value } : s))}
                />
              </FormField>
            </div>
          </Card>
        </section>

        <section>
          <Eyebrow>Store Photo</Eyebrow>
          <Card className="mt-3 p-6">
            <ImageManager
              label="Store photo"
              hint="Represents your physical store across the top of the public Contact page. JPEG, PNG, or WebP."
              hasImage={Boolean(settings.store_image_url)}
              loadUrl={loadStoreImageBlobUrl}
              upload={uploadStoreImage}
              remove={deleteStoreImage}
              onUpdated={setSettings}
              isReadOnly={isReadOnly}
            />
          </Card>
        </section>

        <div className="flex items-center gap-3">
          <Button type="submit" variant="primary" disabled={isSaving || isReadOnly}>
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
            disabled={isReadOnly}
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
                      disabled={isReadOnly}
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
                      disabled={isReadOnly}
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
// Legal Documents: per-language inline Markdown editors.
// ---------------------------------------------------------------------------

function LegalDocumentsTab() {
  return (
    <div className="flex flex-col gap-8">
      <DocumentEditor type="terms" title="Terms of Service" />
      <DocumentEditor type="privacy" title="Privacy Policy" />
    </div>
  );
}

function DocumentEditor({ type, title }: { type: DocumentType; title: string }) {
  const { isReadOnly } = useAdminPermissions();
  const [languages, setLanguages] = useState<Language[]>([]);
  const [activeLocale, setActiveLocale] = useState("en");
  const [content, setContent] = useState("");
  const [preview, setPreview] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [savedAt, setSavedAt] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listLanguages()
      .then(setLanguages)
      .catch(() => {});
  }, []);

  useEffect(() => {
    setIsLoading(true);
    setError(null);
    setSavedAt(null);
    getLegalContent(type, activeLocale)
      .then((r) => setContent(r.content_md))
      .catch(() => setContent(""))
      .finally(() => setIsLoading(false));
  }, [type, activeLocale]);

  async function handleSave() {
    setIsSaving(true);
    setError(null);
    try {
      await saveLegalContent(type, activeLocale, content);
      setSavedAt(Date.now());
    } catch {
      setError("Could not save. Please try again.");
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <section>
      <div className="flex items-center justify-between">
        <Eyebrow>{title}</Eyebrow>
        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            type="button"
            onClick={() => setPreview((p) => !p)}
          >
            {preview ? "Edit" : "Preview"}
          </Button>
          <Button
            variant="primary"
            size="sm"
            type="button"
            disabled={isSaving || isReadOnly}
            onClick={handleSave}
          >
            {isSaving ? "Saving…" : "Save"}
          </Button>
        </div>
      </div>

      {languages.length > 1 && (
        <div className="mt-3 flex gap-1">
          {languages.map((lang) => (
            <button
              key={lang.code}
              type="button"
              onClick={() => setActiveLocale(lang.code)}
              className={`rounded-sm px-3 py-1 text-xs font-medium transition-colors ${
                activeLocale === lang.code
                  ? "bg-stone-900 text-white"
                  : "bg-stone-100 text-stone-600 hover:bg-stone-200"
              }`}
            >
              {lang.name}
            </button>
          ))}
        </div>
      )}

      <Card className="mt-3 overflow-hidden p-0">
        {isLoading ? (
          <div className="p-6">
            <Text size="sm" tone="muted">Loading…</Text>
          </div>
        ) : preview ? (
          <div className="prose prose-stone max-w-none p-6 text-sm">
            <MarkdownPreview content={content} />
          </div>
        ) : (
          <textarea
            value={content}
            onChange={(e) => { setContent(e.target.value); setSavedAt(null); }}
            disabled={isReadOnly}
            rows={24}
            placeholder={`# ${title}\n\nWrite the document content here using Markdown…`}
            className="w-full resize-y bg-transparent p-6 font-mono text-sm leading-relaxed text-stone-800 outline-none placeholder:text-stone-400 disabled:cursor-not-allowed disabled:text-stone-400"
          />
        )}
      </Card>

      <div className="mt-2 flex items-center gap-3">
        {error && <Text size="xs" tone="danger">{error}</Text>}
        {!error && savedAt && <Text size="xs" tone="muted">Saved.</Text>}
        <Text size="xs" tone="muted" className="ml-auto">
          Markdown supported — headings, bold, lists, links.
        </Text>
      </div>
    </section>
  );
}

function MarkdownPreview({ content }: { content: string }) {
  if (!content.trim()) {
    return <Text size="sm" tone="muted">Nothing to preview yet.</Text>;
  }
  return <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>;
}

// ---------------------------------------------------------------------------
// Store Language: add/enable/disable/remove languages.
// ---------------------------------------------------------------------------

function LanguagesTab() {
  const { isReadOnly } = useAdminPermissions();
  const [languages, setLanguages] = useState<Language[]>([]);
  const [countryDrafts, setCountryDrafts] = useState<Record<string, string>>({});
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [isAdding, setIsAdding] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function refresh() {
    const langs = await listLanguages();
    setLanguages(langs);
    setCountryDrafts(Object.fromEntries(langs.map((l) => [l.code, l.country_code ?? ""])));
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

  async function handleSetDefault(lang: Language) {
    await setDefaultLanguage(lang.code);
    await refresh();
  }

  async function handleSetCountry(lang: Language) {
    await setLanguageCountry(lang.code, (countryDrafts[lang.code] ?? "").trim().toUpperCase());
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
                <li key={lang.code} className="flex flex-col gap-2 border-b border-stone-100 pb-3 last:border-0 last:pb-0">
                  <div className="flex items-center justify-between gap-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <Text size="sm" className="font-medium">
                        {lang.name}
                      </Text>
                      <Badge variant="neutral">{lang.code.toUpperCase()}</Badge>
                      {lang.is_default && <Badge variant="brand">Default</Badge>}
                      {lang.country_code && <Badge variant="neutral">Geo: {lang.country_code}</Badge>}
                    </div>
                    <div className="flex items-center gap-2">
                      {lang.enabled && !lang.is_default && (
                        <Button variant="ghost" size="sm" type="button" disabled={isReadOnly} onClick={() => handleSetDefault(lang)}>
                          Set default
                        </Button>
                      )}
                      {!lang.is_default && (
                        <Button variant="ghost" size="sm" type="button" disabled={isReadOnly} onClick={() => handleToggle(lang)}>
                          {lang.enabled ? "Disable" : "Enable"}
                        </Button>
                      )}
                      {!lang.is_default && (
                        <Button
                          variant="ghost"
                          size="sm"
                          type="button"
                          disabled={isReadOnly}
                          onClick={() => handleDelete(lang)}
                          className="text-danger-600 hover:bg-danger-50"
                        >
                          Remove
                        </Button>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Text size="xs" tone="muted">
                      Show to visitors from
                    </Text>
                    <Input
                      value={countryDrafts[lang.code] ?? ""}
                      onChange={(e) => setCountryDrafts((d) => ({ ...d, [lang.code]: e.target.value.toUpperCase() }))}
                      placeholder="BG"
                      className="w-20"
                      disabled={isReadOnly}
                    />
                    <Button
                      variant="ghost"
                      size="sm"
                      type="button"
                      disabled={isReadOnly || (countryDrafts[lang.code] ?? "") === (lang.country_code ?? "")}
                      onClick={() => handleSetCountry(lang)}
                    >
                      Save
                    </Button>
                    <Text size="xs" tone="muted">
                      ISO country code — leave blank to disable geo targeting.
                    </Text>
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
            <Button type="submit" variant="primary" disabled={isAdding || isReadOnly || !code.trim() || !name.trim()}>
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
