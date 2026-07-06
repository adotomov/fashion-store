import { useEffect, useState } from "react";

import { useAdminPermissions } from "../../features/admin/AdminPermissionsContext";

import { Badge } from "../../components/ui/Badge";
import { Button } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { Checkbox } from "../../components/ui/Checkbox";
import { EmptyState } from "../../components/admin/EmptyState";
import { FormField } from "../../components/ui/FormField";
import { Icon } from "../../components/ui/Icon";
import { Input } from "../../components/ui/Input";
import { Modal } from "../../components/ui/Modal";
import { Select } from "../../components/ui/Select";
import { Tabs } from "../../components/ui/Tabs";
import { Text } from "../../components/ui/Text";
import { type Category, listCategories } from "../../lib/api/categories";
import { type Product, listProducts } from "../../lib/api/products";
import { type ProductType, listProductTypes } from "../../lib/api/product-types";
import {
  type CreateDiscountCodeInput,
  type CreatePromotionInput,
  type DiscountCode,
  type Promotion,
  type PromotionType,
  type TargetType,
  createDiscountCode,
  createPromotion,
  deleteDiscountCode,
  deletePromotion,
  listDiscountCodes,
  listPromotions,
  updateDiscountCode,
  updatePromotion,
} from "../../lib/api/admin-promotions";

export const handle = { title: "Promotions" };

const TABS = [
  { id: "summary", label: "Summary" },
  { id: "promotions", label: "Promotions" },
  { id: "codes", label: "Discount Codes" },
];

function formatDate(val?: string): string {
  if (!val) return "—";
  return new Date(val).toLocaleDateString(undefined, { dateStyle: "medium" });
}

function promotionLabel(p: Promotion): string {
  if (p.type === "percentage" && p.value_percent != null) return `-${p.value_percent}%`;
  if (p.type === "fixed" && p.value_fixed_minor != null)
    return `-${(p.value_fixed_minor / 100).toFixed(2)} ${p.value_fixed_currency ?? ""}`;
  if (p.type === "bxgy" && p.buy_qty != null && p.get_qty != null)
    return `Buy ${p.buy_qty} Get ${p.get_qty}${p.get_discount_pct === 100 ? " Free" : ` at ${p.get_discount_pct}% off`}`;
  return p.type;
}

function targetLabel(p: Promotion): string {
  switch (p.target_type) {
    case "all":
      return "All products";
    case "category":
      return `${p.category_ids.length} categor${p.category_ids.length === 1 ? "y" : "ies"}`;
    case "product_type":
      return `${p.type_ids.length} product type${p.type_ids.length === 1 ? "" : "s"}`;
    case "product":
      return `${p.product_ids.length} product${p.product_ids.length === 1 ? "" : "s"}`;
  }
}

// ─── Summary tab ────────────────────────────────────────────────────────────

function SummaryTab({ promotions }: { promotions: Promotion[] }) {
  const active = promotions.filter((p) => p.is_active);
  if (active.length === 0) {
    return (
      <EmptyState
        icon="tag"
        title="No active promotions"
        description="Activate a promotion in the Promotions tab to see it here."
      />
    );
  }
  return (
    <div className="flex flex-col gap-3">
      {active.map((p) => (
        <Card key={p.id} className="flex items-center gap-4 p-4">
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <Text size="sm" className="font-medium text-stone-900">
                {p.name}
              </Text>
              <Badge variant="accent">{promotionLabel(p)}</Badge>
            </div>
            <Text size="xs" tone="muted" className="mt-0.5">
              {targetLabel(p)} · Priority {p.priority}
              {p.starts_at || p.ends_at ? ` · ${formatDate(p.starts_at)} – ${formatDate(p.ends_at)}` : ""}
            </Text>
          </div>
        </Card>
      ))}
    </div>
  );
}

// ─── Promotion form ──────────────────────────────────────────────────────────

type PromotionFormData = {
  name: string;
  description: string;
  type: PromotionType;
  value_percent: string;
  value_fixed_minor: string;
  value_fixed_currency: string;
  buy_qty: string;
  get_qty: string;
  get_discount_pct: string;
  min_quantity: string;
  target_type: TargetType;
  category_ids: string[];
  type_ids: string[];
  product_ids: string[];
  starts_at: string;
  ends_at: string;
  is_active: boolean;
  priority: string;
};

const emptyForm: PromotionFormData = {
  name: "",
  description: "",
  type: "percentage",
  value_percent: "10",
  value_fixed_minor: "",
  value_fixed_currency: "BGN",
  buy_qty: "2",
  get_qty: "1",
  get_discount_pct: "100",
  min_quantity: "1",
  target_type: "all",
  category_ids: [],
  type_ids: [],
  product_ids: [],
  starts_at: "",
  ends_at: "",
  is_active: true,
  priority: "0",
};

function promotionToForm(p: Promotion): PromotionFormData {
  return {
    name: p.name,
    description: p.description,
    type: p.type,
    value_percent: p.value_percent != null ? String(p.value_percent) : "",
    value_fixed_minor: p.value_fixed_minor != null ? String(p.value_fixed_minor / 100) : "",
    value_fixed_currency: p.value_fixed_currency ?? "BGN",
    buy_qty: p.buy_qty != null ? String(p.buy_qty) : "",
    get_qty: p.get_qty != null ? String(p.get_qty) : "",
    get_discount_pct: p.get_discount_pct != null ? String(p.get_discount_pct) : "100",
    min_quantity: String(p.min_quantity),
    target_type: p.target_type,
    category_ids: p.category_ids ?? [],
    type_ids: p.type_ids ?? [],
    product_ids: p.product_ids ?? [],
    starts_at: p.starts_at ? p.starts_at.slice(0, 16) : "",
    ends_at: p.ends_at ? p.ends_at.slice(0, 16) : "",
    is_active: p.is_active,
    priority: String(p.priority),
  };
}

function formToInput(f: PromotionFormData): CreatePromotionInput {
  return {
    name: f.name,
    description: f.description || undefined,
    type: f.type,
    value_percent: f.type === "percentage" && f.value_percent ? Number(f.value_percent) : undefined,
    value_fixed_minor:
      f.type === "fixed" && f.value_fixed_minor ? Math.round(Number(f.value_fixed_minor) * 100) : undefined,
    value_fixed_currency: f.type === "fixed" ? f.value_fixed_currency : undefined,
    buy_qty: f.type === "bxgy" && f.buy_qty ? Number(f.buy_qty) : undefined,
    get_qty: f.type === "bxgy" && f.get_qty ? Number(f.get_qty) : undefined,
    get_discount_pct: f.type === "bxgy" && f.get_discount_pct ? Number(f.get_discount_pct) : undefined,
    min_quantity: f.min_quantity ? Number(f.min_quantity) : 1,
    target_type: f.target_type,
    category_ids: f.target_type === "category" ? f.category_ids : undefined,
    type_ids: f.target_type === "product_type" ? f.type_ids : undefined,
    product_ids: f.target_type === "product" ? f.product_ids : undefined,
    starts_at: f.starts_at ? new Date(f.starts_at).toISOString() : undefined,
    ends_at: f.ends_at ? new Date(f.ends_at).toISOString() : undefined,
    is_active: f.is_active,
    priority: f.priority ? Number(f.priority) : 0,
  };
}

function MultiCheckList({
  items,
  selected,
  onChange,
}: {
  items: { id: string; label: string }[];
  selected: string[];
  onChange: (ids: string[]) => void;
}) {
  function toggle(id: string) {
    onChange(selected.includes(id) ? selected.filter((x) => x !== id) : [...selected, id]);
  }
  if (items.length === 0) {
    return <Text size="sm" tone="muted">No items found.</Text>;
  }
  return (
    <div className="flex max-h-40 flex-col gap-1 overflow-y-auto rounded-sm border border-stone-200 p-2">
      {items.map((item) => (
        <label key={item.id} className="flex cursor-pointer items-center gap-2 rounded px-1 py-0.5 text-sm hover:bg-stone-50">
          <input
            type="checkbox"
            checked={selected.includes(item.id)}
            onChange={() => toggle(item.id)}
            className="h-4 w-4 rounded-sm border-stone-400 accent-stone-900"
          />
          {item.label}
        </label>
      ))}
    </div>
  );
}

function PromotionModal({
  editing,
  onClose,
  onSaved,
}: {
  editing: Promotion | null;
  onClose: () => void;
  onSaved: (p: Promotion) => void;
}) {
  const [form, setForm] = useState<PromotionFormData>(editing ? promotionToForm(editing) : emptyForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [categories, setCategories] = useState<Category[]>([]);
  const [productTypes, setProductTypes] = useState<ProductType[]>([]);
  const [products, setProducts] = useState<Product[]>([]);

  useEffect(() => {
    listCategories().then(setCategories).catch(() => {});
    listProductTypes().then(setProductTypes).catch(() => {});
    listProducts().then(setProducts).catch(() => {});
  }, []);

  function set(field: keyof PromotionFormData, value: string | boolean) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  function setIds(field: "category_ids" | "type_ids" | "product_ids", ids: string[]) {
    setForm((prev) => ({ ...prev, [field]: ids }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    try {
      const input = formToInput(form);
      const result = editing ? await updatePromotion(editing.id, input) : await createPromotion(input);
      onSaved(result);
    } catch {
      setError("Failed to save promotion.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal open title={editing ? "Edit Promotion" : "New Promotion"} onClose={onClose}>
      <form onSubmit={(e) => void handleSubmit(e)} className="flex flex-col gap-4">
        <FormField label="Name">
          <Input value={form.name} onChange={(e) => set("name", e.target.value)} required />
        </FormField>
        <FormField label="Description">
          <Input value={form.description} onChange={(e) => set("description", e.target.value)} />
        </FormField>

        <div className="grid grid-cols-2 gap-3">
          <FormField label="Type">
            <Select value={form.type} onChange={(e) => set("type", e.target.value as PromotionType)}>
              <option value="percentage">Percentage off</option>
              <option value="fixed">Fixed amount off</option>
              <option value="bxgy">Buy X Get Y</option>
            </Select>
          </FormField>
          <FormField label="Target">
            <Select value={form.target_type} onChange={(e) => set("target_type", e.target.value as TargetType)}>
              <option value="all">All products</option>
              <option value="category">By category</option>
              <option value="product_type">By product type</option>
              <option value="product">Specific products</option>
            </Select>
          </FormField>
        </div>

        {form.target_type === "category" && (
          <FormField label="Select categories">
            <MultiCheckList
              items={categories.map((c) => ({ id: c.id, label: c.name }))}
              selected={form.category_ids}
              onChange={(ids) => setIds("category_ids", ids)}
            />
          </FormField>
        )}
        {form.target_type === "product_type" && (
          <FormField label="Select product types">
            <MultiCheckList
              items={productTypes.map((t) => ({ id: t.id, label: t.name }))}
              selected={form.type_ids}
              onChange={(ids) => setIds("type_ids", ids)}
            />
          </FormField>
        )}
        {form.target_type === "product" && (
          <FormField label="Select products">
            <MultiCheckList
              items={products.map((p) => ({ id: p.id, label: p.name }))}
              selected={form.product_ids}
              onChange={(ids) => setIds("product_ids", ids)}
            />
          </FormField>
        )}

        {form.type === "percentage" && (
          <FormField label="Discount %">
            <Input
              type="number"
              min={1}
              max={100}
              value={form.value_percent}
              onChange={(e) => set("value_percent", e.target.value)}
              required
            />
          </FormField>
        )}
        {form.type === "fixed" && (
          <div className="grid grid-cols-2 gap-3">
            <FormField label="Amount">
              <Input
                type="number"
                min={0}
                step="0.01"
                value={form.value_fixed_minor}
                onChange={(e) => set("value_fixed_minor", e.target.value)}
                required
              />
            </FormField>
            <FormField label="Currency">
              <Input value={form.value_fixed_currency} onChange={(e) => set("value_fixed_currency", e.target.value)} />
            </FormField>
          </div>
        )}
        {form.type === "bxgy" && (
          <div className="grid grid-cols-3 gap-3">
            <FormField label="Buy qty">
              <Input
                type="number"
                min={1}
                value={form.buy_qty}
                onChange={(e) => set("buy_qty", e.target.value)}
                required
              />
            </FormField>
            <FormField label="Get qty">
              <Input
                type="number"
                min={1}
                value={form.get_qty}
                onChange={(e) => set("get_qty", e.target.value)}
                required
              />
            </FormField>
            <FormField label="Discount on Get %">
              <Input
                type="number"
                min={1}
                max={100}
                value={form.get_discount_pct}
                onChange={(e) => set("get_discount_pct", e.target.value)}
              />
            </FormField>
          </div>
        )}

        <div className="grid grid-cols-2 gap-3">
          <FormField label="Min quantity">
            <Input
              type="number"
              min={1}
              value={form.min_quantity}
              onChange={(e) => set("min_quantity", e.target.value)}
            />
          </FormField>
          <FormField label="Priority">
            <Input type="number" value={form.priority} onChange={(e) => set("priority", e.target.value)} />
          </FormField>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <FormField label="Starts at">
            <Input
              type="datetime-local"
              value={form.starts_at}
              onChange={(e) => set("starts_at", e.target.value)}
            />
          </FormField>
          <FormField label="Ends at">
            <Input
              type="datetime-local"
              value={form.ends_at}
              onChange={(e) => set("ends_at", e.target.value)}
            />
          </FormField>
        </div>

        <Checkbox checked={form.is_active} onChange={(e) => set("is_active", e.target.checked)} label="Active" />

        {error && (
          <Text size="sm" tone="danger">
            {error}
          </Text>
        )}

        <div className="flex gap-2 justify-end">
          <Button type="button" variant="secondary" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" size="sm" disabled={saving}>
            {saving ? "Saving…" : "Save"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

// ─── Promotions tab ──────────────────────────────────────────────────────────

function PromotionsTab({
  promotions,
  onUpdated,
  onDeleted,
  onCreated,
}: {
  promotions: Promotion[];
  onUpdated: (p: Promotion) => void;
  onDeleted: (id: string) => void;
  onCreated: (p: Promotion) => void;
}) {
  const { isReadOnly } = useAdminPermissions();
  const [modalPromotion, setModalPromotion] = useState<Promotion | null | "new">(null);

  async function handleToggle(p: Promotion) {
    try {
      const updated = await updatePromotion(p.id, { is_active: !p.is_active });
      onUpdated(updated);
    } catch {
      /* ignore */
    }
  }

  async function handleDelete(p: Promotion) {
    if (!confirm(`Delete promotion "${p.name}"?`)) return;
    try {
      await deletePromotion(p.id);
      onDeleted(p.id);
    } catch {
      /* ignore */
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Button size="sm" onClick={() => setModalPromotion("new")} disabled={isReadOnly}>
          <Icon name="plus" size={16} />
          Add Promotion
        </Button>
      </div>

      {promotions.length === 0 ? (
        <EmptyState icon="tag" title="No promotions yet" description="Create your first promotion to get started." />
      ) : (
        <div className="flex flex-col gap-2">
          {promotions.map((p) => (
            <Card key={p.id} className="flex items-center gap-4 p-4">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <Text size="sm" className="font-medium text-stone-900">
                    {p.name}
                  </Text>
                  <Badge variant={p.is_active ? "accent" : "neutral"}>{promotionLabel(p)}</Badge>
                  {!p.is_active && (
                    <Badge variant="neutral">
                      <Text size="xs" tone="muted">
                        inactive
                      </Text>
                    </Badge>
                  )}
                </div>
                <Text size="xs" tone="muted" className="mt-0.5">
                  {targetLabel(p)} · min qty: {p.min_quantity} · priority: {p.priority}
                  {(p.starts_at || p.ends_at) && ` · ${formatDate(p.starts_at)} – ${formatDate(p.ends_at)}`}
                </Text>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <button
                  type="button"
                  onClick={() => void handleToggle(p)}
                  disabled={isReadOnly}
                  className="text-xs text-stone-500 hover:text-stone-900 px-2 py-1 rounded border border-stone-200 hover:border-stone-400 transition-colors disabled:pointer-events-none disabled:opacity-40"
                >
                  {p.is_active ? "Deactivate" : "Activate"}
                </button>
                <button
                  type="button"
                  onClick={() => setModalPromotion(p)}
                  disabled={isReadOnly}
                  className="rounded p-1.5 text-stone-500 hover:bg-stone-100 hover:text-stone-900 transition-colors disabled:pointer-events-none disabled:opacity-40"
                >
                  <Icon name="pencil" size={15} />
                </button>
                <button
                  type="button"
                  onClick={() => void handleDelete(p)}
                  disabled={isReadOnly}
                  className="rounded p-1.5 text-stone-500 hover:bg-danger-50 hover:text-danger-600 transition-colors disabled:pointer-events-none disabled:opacity-40"
                >
                  <Icon name="trash" size={15} />
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {modalPromotion != null && (
        <PromotionModal
          editing={modalPromotion === "new" ? null : modalPromotion}
          onClose={() => setModalPromotion(null)}
          onSaved={(saved) => {
            if (modalPromotion === "new") {
              onCreated(saved);
            } else {
              onUpdated(saved);
            }
            setModalPromotion(null);
          }}
        />
      )}
    </div>
  );
}

// ─── Discount Codes tab ──────────────────────────────────────────────────────

type CodeFormData = {
  code: string;
  value_percent: string;
  starts_at: string;
  expires_at: string;
  max_uses: string;
  is_active: boolean;
};

const emptyCodeForm: CodeFormData = {
  code: "",
  value_percent: "10",
  starts_at: "",
  expires_at: "",
  max_uses: "",
  is_active: true,
};

function codeToForm(c: DiscountCode): CodeFormData {
  return {
    code: c.code,
    value_percent: String(c.value_percent),
    starts_at: c.starts_at ? c.starts_at.slice(0, 16) : "",
    expires_at: c.expires_at ? c.expires_at.slice(0, 16) : "",
    max_uses: c.max_uses != null ? String(c.max_uses) : "",
    is_active: c.is_active,
  };
}

function codeFormToInput(f: CodeFormData): CreateDiscountCodeInput {
  return {
    code: f.code.toUpperCase(),
    value_percent: Number(f.value_percent),
    starts_at: f.starts_at ? new Date(f.starts_at).toISOString() : undefined,
    expires_at: f.expires_at ? new Date(f.expires_at).toISOString() : undefined,
    max_uses: f.max_uses ? Number(f.max_uses) : undefined,
    is_active: f.is_active,
  };
}

function CodeModal({
  editing,
  onClose,
  onSaved,
}: {
  editing: DiscountCode | null;
  onClose: () => void;
  onSaved: (c: DiscountCode) => void;
}) {
  const [form, setForm] = useState<CodeFormData>(editing ? codeToForm(editing) : emptyCodeForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  function set(field: keyof CodeFormData, value: string | boolean) {
    setForm((prev) => ({ ...prev, [field]: value }));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    try {
      const input = codeFormToInput(form);
      const result = editing ? await updateDiscountCode(editing.id, input) : await createDiscountCode(input);
      onSaved(result);
    } catch {
      setError("Failed to save code. It may already exist.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal open title={editing ? "Edit Discount Code" : "New Discount Code"} onClose={onClose}>
      <form onSubmit={(e) => void handleSubmit(e)} className="flex flex-col gap-4">
        <FormField label="Code">
          <Input
            value={form.code}
            onChange={(e) => set("code", e.target.value.toUpperCase())}
            placeholder="e.g. SUMMER20"
            required
            disabled={!!editing}
          />
        </FormField>
        <FormField label="Discount %">
          <Input
            type="number"
            min={1}
            max={100}
            value={form.value_percent}
            onChange={(e) => set("value_percent", e.target.value)}
            required
          />
        </FormField>
        <div className="grid grid-cols-2 gap-3">
          <FormField label="Starts at">
            <Input
              type="datetime-local"
              value={form.starts_at}
              onChange={(e) => set("starts_at", e.target.value)}
            />
          </FormField>
          <FormField label="Expires at">
            <Input
              type="datetime-local"
              value={form.expires_at}
              onChange={(e) => set("expires_at", e.target.value)}
            />
          </FormField>
        </div>
        <FormField label="Max uses (blank = unlimited)">
          <Input
            type="number"
            min={1}
            value={form.max_uses}
            onChange={(e) => set("max_uses", e.target.value)}
            placeholder="Unlimited"
          />
        </FormField>
        <Checkbox checked={form.is_active} onChange={(e) => set("is_active", e.target.checked)} label="Active" />

        {error && (
          <Text size="sm" tone="danger">
            {error}
          </Text>
        )}

        <div className="flex gap-2 justify-end">
          <Button type="button" variant="secondary" size="sm" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" size="sm" disabled={saving}>
            {saving ? "Saving…" : "Save"}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function CodesTab({
  codes,
  onUpdated,
  onDeleted,
  onCreated,
}: {
  codes: DiscountCode[];
  onUpdated: (c: DiscountCode) => void;
  onDeleted: (id: string) => void;
  onCreated: (c: DiscountCode) => void;
}) {
  const { isReadOnly } = useAdminPermissions();
  const [modal, setModal] = useState<DiscountCode | null | "new">(null);

  async function handleToggle(c: DiscountCode) {
    try {
      const updated = await updateDiscountCode(c.id, { is_active: !c.is_active });
      onUpdated(updated);
    } catch {
      /* ignore */
    }
  }

  async function handleDelete(c: DiscountCode) {
    if (!confirm(`Delete code "${c.code}"?`)) return;
    try {
      await deleteDiscountCode(c.id);
      onDeleted(c.id);
    } catch {
      /* ignore */
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-end">
        <Button size="sm" onClick={() => setModal("new")} disabled={isReadOnly}>
          <Icon name="plus" size={16} />
          Add Code
        </Button>
      </div>

      {codes.length === 0 ? (
        <EmptyState
          icon="tag"
          title="No discount codes yet"
          description="Create codes to share with customers for checkout discounts."
        />
      ) : (
        <div className="flex flex-col gap-2">
          {codes.map((c) => (
            <Card key={c.id} className="flex items-center gap-4 p-4">
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 flex-wrap">
                  <Text size="sm" className="font-mono font-medium text-stone-900">
                    {c.code}
                  </Text>
                  <Badge variant={c.is_active ? "accent" : "neutral"}>-{c.value_percent}%</Badge>
                </div>
                <Text size="xs" tone="muted" className="mt-0.5">
                  Used {c.use_count}{c.max_uses != null ? `/${c.max_uses}` : ""} times
                  {(c.starts_at || c.expires_at) && ` · ${formatDate(c.starts_at)} – ${formatDate(c.expires_at)}`}
                </Text>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <button
                  type="button"
                  onClick={() => void handleToggle(c)}
                  disabled={isReadOnly}
                  className="text-xs text-stone-500 hover:text-stone-900 px-2 py-1 rounded border border-stone-200 hover:border-stone-400 transition-colors disabled:pointer-events-none disabled:opacity-40"
                >
                  {c.is_active ? "Deactivate" : "Activate"}
                </button>
                <button
                  type="button"
                  onClick={() => setModal(c)}
                  disabled={isReadOnly}
                  className="rounded p-1.5 text-stone-500 hover:bg-stone-100 hover:text-stone-900 transition-colors disabled:pointer-events-none disabled:opacity-40"
                >
                  <Icon name="pencil" size={15} />
                </button>
                <button
                  type="button"
                  onClick={() => void handleDelete(c)}
                  disabled={isReadOnly}
                  className="rounded p-1.5 text-stone-500 hover:bg-danger-50 hover:text-danger-600 transition-colors disabled:pointer-events-none disabled:opacity-40"
                >
                  <Icon name="trash" size={15} />
                </button>
              </div>
            </Card>
          ))}
        </div>
      )}

      {modal != null && (
        <CodeModal
          editing={modal === "new" ? null : modal}
          onClose={() => setModal(null)}
          onSaved={(saved) => {
            if (modal === "new") {
              onCreated(saved);
            } else {
              onUpdated(saved);
            }
            setModal(null);
          }}
        />
      )}
    </div>
  );
}

// ─── Page ────────────────────────────────────────────────────────────────────

export default function AdminPromotions() {
  const [activeTab, setActiveTab] = useState("summary");
  const [promotions, setPromotions] = useState<Promotion[] | null>(null);
  const [codes, setCodes] = useState<DiscountCode[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    Promise.all([listPromotions(), listDiscountCodes()])
      .then(([p, c]) => {
        setPromotions(p);
        setCodes(c);
      })
      .catch(() => setError("Could not load promotions data."));
  }, []);

  if (error) {
    return (
      <Text size="sm" tone="danger">
        {error}
      </Text>
    );
  }

  if (promotions === null || codes === null) {
    return (
      <Text size="sm" tone="muted">
        Loading…
      </Text>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <Tabs tabs={TABS} activeTab={activeTab} onChange={setActiveTab}>
        {activeTab === "summary" && <SummaryTab promotions={promotions} />}
        {activeTab === "promotions" && (
          <PromotionsTab
            promotions={promotions}
            onUpdated={(p) => setPromotions((prev) => prev?.map((x) => (x.id === p.id ? p : x)) ?? prev)}
            onDeleted={(id) => setPromotions((prev) => prev?.filter((x) => x.id !== id) ?? prev)}
            onCreated={(p) => setPromotions((prev) => [p, ...(prev ?? [])])}
          />
        )}
        {activeTab === "codes" && (
          <CodesTab
            codes={codes}
            onUpdated={(c) => setCodes((prev) => prev?.map((x) => (x.id === c.id ? c : x)) ?? prev)}
            onDeleted={(id) => setCodes((prev) => prev?.filter((x) => x.id !== id) ?? prev)}
            onCreated={(c) => setCodes((prev) => [c, ...(prev ?? [])])}
          />
        )}
      </Tabs>
    </div>
  );
}
