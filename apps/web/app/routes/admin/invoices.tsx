import { useEffect, useState } from "react";

import { useAdminPermissions } from "../../features/admin/AdminPermissionsContext";

import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Input } from "../../components/ui/Input";
import { Modal } from "../../components/ui/Modal";
import { Select } from "../../components/ui/Select";
import { Tabs } from "../../components/ui/Tabs";
import { Eyebrow, Text } from "../../components/ui/Text";
import {
  type Courier,
  type InvoiceListItem,
  type InvoiceSettings,
  type ListInvoicesParams,
  createCourier,
  deleteCourier,
  exportInvoicesCSV,
  fetchInvoiceHTML,
  getInvoiceSettings,
  generateStorno,
  listCouriers,
  listInvoices,
  saveInvoiceSettings,
  updateCourier,
} from "../../lib/api/invoices";
import {
  type TaxGroup,
  TAX_GROUP_IDENTIFIERS,
  createTaxGroup,
  deleteTaxGroup,
  listTaxGroups,
  updateTaxGroup,
} from "../../lib/api/tax-groups";

export const handle = { title: "Invoices" };

const TABS = [
  { id: "invoices", label: "Invoices" },
  { id: "settings", label: "Settings" },
  { id: "tax", label: "Tax" },
];

const PAGE_SIZE = 20;

function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString("bg-BG", { dateStyle: "medium" });
}

// Invoice amounts arrive from the API already in major units (e.g. 24.50),
// not minor units — the backend divides by 100 before serializing.
function formatAmount(amount: number, currency: string) {
  return `${amount.toFixed(2)} ${currency}`;
}

function paymentLabel(method: string) {
  switch (method) {
    case "card_online": return "Card online";
    case "cash_on_delivery": return "Cash on delivery";
    case "easy_box": return "EasyBox";
    default: return method;
  }
}

function InvoicesTab() {
  const { isReadOnly } = useAdminPermissions();
  const [invoices, setInvoices] = useState<InvoiceListItem[]>([]);
  const [total, setTotal] = useState(0);
  const [offset, setOffset] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [docType, setDocType] = useState("");
  const [paymentMethod, setPaymentMethod] = useState("");
  const [search, setSearch] = useState("");

  const [stornoModal, setStornoModal] = useState<InvoiceListItem | null>(null);
  const [stornoLoading, setStornoLoading] = useState(false);
  const [exportLoading, setExportLoading] = useState(false);

  async function load(params: ListInvoicesParams) {
    setLoading(true);
    setError(null);
    try {
      const res = await listInvoices({ ...params, limit: PAGE_SIZE });
      setInvoices(res.invoices ?? []);
      setTotal(res.total ?? 0);
    } catch {
      setError("Could not load invoices.");
    } finally {
      setLoading(false);
    }
  }

  // Open the invoice HTML in a new tab. The window is opened synchronously
  // (so the browser doesn't treat it as a blocked popup), then filled with the
  // token-authenticated HTML once it arrives.
  async function handleView(id: string) {
    const win = window.open("", "_blank");
    try {
      const html = await fetchInvoiceHTML(id);
      if (win) {
        win.document.open();
        win.document.write(html);
        win.document.close();
      }
    } catch {
      win?.close();
      setError("Could not open the invoice.");
    }
  }

  useEffect(() => {
    load({ from: from || undefined, to: to || undefined, document_type: docType || undefined, payment_method: paymentMethod || undefined, q: search || undefined, offset });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [from, to, docType, paymentMethod, search, offset]);

  async function handleStorno() {
    if (!stornoModal) return;
    setStornoLoading(true);
    try {
      await generateStorno(stornoModal.id);
      setStornoModal(null);
      load({ from: from || undefined, to: to || undefined, document_type: docType || undefined, payment_method: paymentMethod || undefined, q: search || undefined, offset });
    } catch {
      setError("Could not issue credit note.");
    } finally {
      setStornoLoading(false);
    }
  }

  async function handleExport() {
    if (!from || !to) return;
    setExportLoading(true);
    try {
      await exportInvoicesCSV(from, to);
    } catch {
      setError("Export failed.");
    } finally {
      setExportLoading(false);
    }
  }

  const totalPages = Math.ceil(total / PAGE_SIZE);
  const currentPage = Math.floor(offset / PAGE_SIZE) + 1;

  return (
    <div className="flex flex-col gap-6">
      <Card className="p-4">
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6">
          <FormField label="From" htmlFor="from">
            <Input id="from" type="date" value={from} onChange={(e) => { setFrom(e.target.value); setOffset(0); }} />
          </FormField>
          <FormField label="To" htmlFor="to">
            <Input id="to" type="date" value={to} onChange={(e) => { setTo(e.target.value); setOffset(0); }} />
          </FormField>
          <FormField label="Type" htmlFor="doc-type">
            <Select id="doc-type" value={docType} onChange={(e) => { setDocType(e.target.value); setOffset(0); }}>
              <option value="">All</option>
              <option value="фактура">Invoice</option>
              <option value="сторно">Credit note</option>
            </Select>
          </FormField>
          <FormField label="Payment" htmlFor="payment">
            <Select id="payment" value={paymentMethod} onChange={(e) => { setPaymentMethod(e.target.value); setOffset(0); }}>
              <option value="">All</option>
              <option value="card_online">Card online</option>
              <option value="cash_on_delivery">Cash on delivery</option>
              <option value="easy_box">EasyBox</option>
            </Select>
          </FormField>
          <FormField label="Search" htmlFor="search">
            <Input id="search" placeholder="Invoice or order #" value={search} onChange={(e) => { setSearch(e.target.value); setOffset(0); }} />
          </FormField>
          <div className="flex items-end">
            <Button variant="secondary" disabled={!from || !to || exportLoading} onClick={handleExport}>
              {exportLoading ? "Exporting…" : "Export CSV"}
            </Button>
          </div>
        </div>
      </Card>

      {error && <Text tone="danger">{error}</Text>}

      <Card>
        {loading ? (
          <div className="p-6">
            <Text tone="muted">Loading…</Text>
          </div>
        ) : invoices.length === 0 ? (
          <div className="p-6">
            <Text tone="muted">No invoices found.</Text>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-xs font-medium uppercase tracking-wide text-[var(--color-text-muted)]">
                  <th className="px-4 py-3">Invoice #</th>
                  <th className="px-4 py-3">Date</th>
                  <th className="px-4 py-3">Order #</th>
                  <th className="px-4 py-3">Type</th>
                  <th className="px-4 py-3">Payment</th>
                  <th className="px-4 py-3">Recipient</th>
                  <th className="px-4 py-3 text-right">Amount</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y">
                {invoices.map((inv) => (
                  <tr key={inv.id} className="hover:bg-[var(--color-surface-hover)]">
                    <td className="px-4 py-3 font-mono">{inv.invoice_number}</td>
                    <td className="px-4 py-3">{formatDate(inv.created_at)}</td>
                    <td className="px-4 py-3">{inv.order_number}</td>
                    <td className="px-4 py-3">
                      <Badge variant={inv.document_type === "фактура" ? "success" : "neutral"}>
                        {inv.document_type}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">{paymentLabel(inv.payment_method)}</td>
                    <td className="px-4 py-3">{inv.recipient_name}</td>
                    <td className="px-4 py-3 text-right font-medium">{formatAmount(inv.total_incl_vat, inv.currency)}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        <Button variant="ghost" size="sm" onClick={() => handleView(inv.id)}>
                          View
                        </Button>
                        {inv.document_type === "фактура" && (
                          <Button variant="ghost" size="sm" onClick={() => setStornoModal(inv)} disabled={isReadOnly}>
                            Credit note
                          </Button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-3">
          <Button variant="ghost" size="sm" disabled={currentPage === 1} onClick={() => setOffset(offset - PAGE_SIZE)}>
            ← Previous
          </Button>
          <Text size="sm" tone="muted">Page {currentPage} of {totalPages}</Text>
          <Button variant="ghost" size="sm" disabled={currentPage === totalPages} onClick={() => setOffset(offset + PAGE_SIZE)}>
            Next →
          </Button>
        </div>
      )}

      {stornoModal && (
        <Modal open title="Issue credit note" onClose={() => setStornoModal(null)}>
          <div className="flex flex-col gap-6">
            <Text>
              Are you sure you want to issue a credit note for invoice <strong>{stornoModal.invoice_number}</strong>? This action cannot be undone.
            </Text>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => setStornoModal(null)}>Cancel</Button>
              <Button variant="primary" onClick={handleStorno} disabled={stornoLoading || isReadOnly}>
                {stornoLoading ? "Processing…" : "Issue credit note"}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}

const EMPTY_SETTINGS: InvoiceSettings = {
  company_name: "",
  company_legal_type: "ООД",
  company_eik: "",
  company_address_street: "",
  company_address_city: "",
  company_address_postal_code: "",
  company_address_country: "България",
  company_email: "",
  company_phone: "",
  nra_store_number: "",
  vat_number: "",
  vat_rate: 20,
};

function SettingsTab() {
  const { isReadOnly } = useAdminPermissions();
  const [settings, setSettings] = useState<InvoiceSettings>(EMPTY_SETTINGS);
  const [couriers, setCouriers] = useState<Courier[]>([]);
  const [loading, setLoading] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [courierModal, setCourierModal] = useState<Courier | "new" | null>(null);
  const [courierForm, setCourierForm] = useState({ name: "", identifier: "", is_active: true, sort_order: 0 });

  async function load() {
    setLoading(true);
    try {
      const [s, c] = await Promise.all([getInvoiceSettings(), listCouriers()]);
      setSettings(s);
      setCouriers(c ?? []);
    } catch {
      setError("Could not load settings.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => { load(); }, []);

  async function handleSaveSettings() {
    setIsSaving(true);
    setSaved(false);
    setError(null);
    try {
      const updated = await saveInvoiceSettings(settings);
      setSettings(updated);
      setSaved(true);
    } catch {
      setError("Could not save settings.");
    } finally {
      setIsSaving(false);
    }
  }

  function openNewCourier() {
    setCourierForm({ name: "", identifier: "", is_active: true, sort_order: couriers.length + 1 });
    setCourierModal("new");
  }

  function openEditCourier(c: Courier) {
    setCourierForm({ name: c.name, identifier: c.identifier, is_active: c.is_active, sort_order: c.sort_order });
    setCourierModal(c);
  }

  async function handleSaveCourier() {
    setIsSaving(true);
    try {
      if (courierModal === "new") {
        await createCourier(courierForm);
      } else if (courierModal) {
        await updateCourier(courierModal.id, courierForm);
      }
      setCourierModal(null);
      const updated = await listCouriers();
      setCouriers(updated ?? []);
    } catch {
      setError("Could not save courier.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDeleteCourier(id: string) {
    try {
      await deleteCourier(id);
      setCouriers(couriers.filter((c) => c.id !== id));
    } catch {
      setError("Could not delete courier.");
    }
  }

  if (loading) return <Text tone="muted">Loading…</Text>;

  return (
    <div className="flex flex-col gap-8">
      {error && <Text tone="danger">{error}</Text>}

      <section>
        <Eyebrow>Company details</Eyebrow>
        <Card className="mt-3 p-6">
          <div className="flex flex-col gap-4">
            <div className="grid grid-cols-2 gap-4">
              <FormField label="Company name" htmlFor="company-name">
                <Input id="company-name" value={settings.company_name} onChange={(e) => setSettings({ ...settings, company_name: e.target.value })} />
              </FormField>
              <FormField label="Legal type" htmlFor="legal-type" hint="ООД / АД / ЕТ / ЕООД">
                <Input id="legal-type" value={settings.company_legal_type} onChange={(e) => setSettings({ ...settings, company_legal_type: e.target.value })} />
              </FormField>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <FormField label="EIK" htmlFor="eik">
                <Input id="eik" value={settings.company_eik} onChange={(e) => setSettings({ ...settings, company_eik: e.target.value })} />
              </FormField>
              <FormField label="VAT number" htmlFor="vat-number">
                <Input id="vat-number" value={settings.vat_number} onChange={(e) => setSettings({ ...settings, vat_number: e.target.value })} />
              </FormField>
            </div>
            <FormField label="Street" htmlFor="address-street">
              <Input id="address-street" value={settings.company_address_street} onChange={(e) => setSettings({ ...settings, company_address_street: e.target.value })} placeholder="e.g. ул. Витоша 15" />
            </FormField>
            <div className="grid grid-cols-3 gap-4">
              <FormField label="Postal code" htmlFor="address-postal">
                <Input id="address-postal" value={settings.company_address_postal_code} onChange={(e) => setSettings({ ...settings, company_address_postal_code: e.target.value })} placeholder="e.g. 1000" />
              </FormField>
              <FormField label="City" htmlFor="address-city">
                <Input id="address-city" value={settings.company_address_city} onChange={(e) => setSettings({ ...settings, company_address_city: e.target.value })} placeholder="e.g. София" />
              </FormField>
              <FormField label="Country" htmlFor="address-country">
                <Input id="address-country" value={settings.company_address_country} onChange={(e) => setSettings({ ...settings, company_address_country: e.target.value })} />
              </FormField>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <FormField label="Email" htmlFor="company-email">
                <Input id="company-email" type="email" value={settings.company_email} onChange={(e) => setSettings({ ...settings, company_email: e.target.value })} />
              </FormField>
              <FormField label="Phone" htmlFor="company-phone">
                <Input id="company-phone" value={settings.company_phone} onChange={(e) => setSettings({ ...settings, company_phone: e.target.value })} />
              </FormField>
            </div>
            <FormField label="NRA store number (УНП)" htmlFor="nra-number" hint="Unique store number from НАП">
              <Input id="nra-number" value={settings.nra_store_number} onChange={(e) => setSettings({ ...settings, nra_store_number: e.target.value })} />
            </FormField>
            <div className="flex items-center gap-4">
              <Button variant="primary" onClick={handleSaveSettings} disabled={isSaving || isReadOnly}>
                {isSaving ? "Saving…" : "Save"}
              </Button>
              {saved && <Text size="sm" tone="muted">Settings saved.</Text>}
            </div>
          </div>
        </Card>
      </section>

      <section>
        <div className="flex items-center justify-between">
          <Eyebrow>Couriers</Eyebrow>
          <Button variant="secondary" size="sm" onClick={openNewCourier} disabled={isReadOnly}>+ Add courier</Button>
        </div>
        <Card className="mt-3">
          {couriers.length === 0 ? (
            <div className="p-6">
              <Text tone="muted">No couriers yet.</Text>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-xs font-medium uppercase tracking-wide text-[var(--color-text-muted)]">
                  <th className="px-4 py-3">Name</th>
                  <th className="px-4 py-3">Identifier</th>
                  <th className="px-4 py-3">Active</th>
                  <th className="px-4 py-3">Order</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y">
                {couriers.sort((a, b) => a.sort_order - b.sort_order).map((c) => (
                  <tr key={c.id} className="hover:bg-[var(--color-surface-hover)]">
                    <td className="px-4 py-3 font-medium">{c.name}</td>
                    <td className="px-4 py-3 font-mono text-xs">{c.identifier}</td>
                    <td className="px-4 py-3">{c.is_active ? "Yes" : "No"}</td>
                    <td className="px-4 py-3">{c.sort_order}</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        <Button variant="ghost" size="sm" onClick={() => openEditCourier(c)} disabled={isReadOnly}>Edit</Button>
                        <Button variant="ghost" size="sm" onClick={() => handleDeleteCourier(c.id)} disabled={isReadOnly}>Delete</Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
      </section>

      {courierModal !== null && (
        <Modal
          open
          title={courierModal === "new" ? "New courier" : "Edit courier"}
          onClose={() => setCourierModal(null)}
        >
          <div className="flex flex-col gap-4">
            <FormField label="Name" htmlFor="c-name">
              <Input id="c-name" value={courierForm.name} onChange={(e) => setCourierForm({ ...courierForm, name: e.target.value })} />
            </FormField>
            <FormField label="Identifier" htmlFor="c-identifier" hint="Unique slug, e.g. speedy">
              <Input id="c-identifier" value={courierForm.identifier} onChange={(e) => setCourierForm({ ...courierForm, identifier: e.target.value })} />
            </FormField>
            <FormField label="Sort order" htmlFor="c-sort">
              <Input id="c-sort" type="number" value={String(courierForm.sort_order)} onChange={(e) => setCourierForm({ ...courierForm, sort_order: Number(e.target.value) })} />
            </FormField>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => setCourierModal(null)}>Cancel</Button>
              <Button variant="primary" onClick={handleSaveCourier} disabled={isSaving || isReadOnly}>
                {isSaving ? "Saving…" : "Save"}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}

function TaxGroupsTab() {
  const { isReadOnly } = useAdminPermissions();
  const [groups, setGroups] = useState<TaxGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [modal, setModal] = useState<null | "new" | TaxGroup>(null);
  const [form, setForm] = useState<{ identifier: string; vat_rate: string }>({ identifier: "Б", vat_rate: "20" });
  const [saveError, setSaveError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  async function refresh() {
    try {
      setGroups(await listTaxGroups());
    } catch {
      setError("Could not load tax groups.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  function openNew() {
    setForm({ identifier: "Б", vat_rate: "20" });
    setSaveError(null);
    setModal("new");
  }

  function openEdit(g: TaxGroup) {
    setForm({ identifier: g.identifier, vat_rate: String(g.vat_rate) });
    setSaveError(null);
    setModal(g);
  }

  async function handleSave() {
    const rate = Number(form.vat_rate);
    if (Number.isNaN(rate) || rate < 0 || rate > 100) {
      setSaveError("VAT rate must be between 0 and 100.");
      return;
    }
    setIsSaving(true);
    setSaveError(null);
    try {
      if (modal === "new") {
        await createTaxGroup(form.identifier, rate);
      } else if (modal) {
        await updateTaxGroup(modal.id, form.identifier, rate);
      }
      setModal(null);
      await refresh();
    } catch {
      setSaveError("Could not save. The identifier may already be in use.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete(id: string) {
    if (!window.confirm("Delete this tax group? Products using it will fall back to the default 20% rate.")) return;
    try {
      await deleteTaxGroup(id);
      await refresh();
    } catch {
      setError("Could not delete tax group.");
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <section>
        <div className="flex items-center justify-between">
          <Eyebrow>Tax groups</Eyebrow>
          <Button variant="secondary" size="sm" onClick={openNew} disabled={isReadOnly}>+ Add tax group</Button>
        </div>
        <Text tone="muted" size="sm" className="mt-1">
          VAT groups (А–Ж) assigned to products. Each product's group sets the VAT rate applied per line on its invoices.
        </Text>

        {error && (
          <Text tone="danger" size="sm" className="mt-2">
            {error}
          </Text>
        )}

        <Card className="mt-3">
          {loading ? (
            <div className="p-6">
              <Text tone="muted">Loading…</Text>
            </div>
          ) : groups.length === 0 ? (
            <div className="p-6">
              <Text tone="muted">No tax groups yet.</Text>
            </div>
          ) : (
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-xs font-medium uppercase tracking-wide text-[var(--color-text-muted)]">
                  <th className="px-4 py-3">Identifier</th>
                  <th className="px-4 py-3">VAT rate</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody className="divide-y">
                {groups.map((g) => (
                  <tr key={g.id} className="hover:bg-[var(--color-surface-hover)]">
                    <td className="px-4 py-3 font-medium">{g.identifier}</td>
                    <td className="px-4 py-3">{g.vat_rate}%</td>
                    <td className="px-4 py-3">
                      <div className="flex items-center justify-end gap-2">
                        <Button variant="ghost" size="sm" onClick={() => openEdit(g)} disabled={isReadOnly}>Edit</Button>
                        <Button variant="ghost" size="sm" onClick={() => handleDelete(g.id)} disabled={isReadOnly}>Delete</Button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </Card>
      </section>

      {modal !== null && (
        <Modal open title={modal === "new" ? "New tax group" : "Edit tax group"} onClose={() => setModal(null)}>
          <div className="flex flex-col gap-4">
            <FormField label="Identifier" htmlFor="tg-identifier" hint="Bulgarian VAT group letter" error={saveError ?? undefined}>
              <Select
                id="tg-identifier"
                value={form.identifier}
                onChange={(e) => setForm({ ...form, identifier: e.target.value })}
              >
                {TAX_GROUP_IDENTIFIERS.map((id) => (
                  <option key={id} value={id}>
                    {id}
                  </option>
                ))}
              </Select>
            </FormField>
            <FormField label="VAT rate (%)" htmlFor="tg-rate">
              <Input
                id="tg-rate"
                type="number"
                value={form.vat_rate}
                onChange={(e) => setForm({ ...form, vat_rate: e.target.value })}
              />
            </FormField>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={() => setModal(null)}>Cancel</Button>
              <Button variant="primary" onClick={handleSave} disabled={isSaving || isReadOnly}>
                {isSaving ? "Saving…" : "Save"}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
}

export default function AdminInvoices() {
  const [activeTab, setActiveTab] = useState("invoices");
  return (
    <div className="flex flex-col gap-6">
      <Tabs tabs={TABS} activeTab={activeTab} onChange={setActiveTab}>
        {activeTab === "invoices" && <InvoicesTab />}
        {activeTab === "settings" && <SettingsTab />}
        {activeTab === "tax" && <TaxGroupsTab />}
      </Tabs>
    </div>
  );
}
