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
import {
  type PaymentMethod,
  type PaymentMethodInput,
  createPaymentMethod,
  deletePaymentMethod,
  listPaymentMethods,
  updatePaymentMethod,
} from "../../lib/api/payment-methods";

export const handle = { title: "Payment Methods" };

const brands = ["Visa", "Mastercard", "American Express", "Discover"];

const currentYear = new Date().getFullYear();
const yearOptions = Array.from({ length: 16 }, (_, i) => currentYear + i);

const emptyForm: PaymentMethodInput = {
  brand: "",
  last4: "",
  exp_month: 1,
  exp_year: currentYear,
  is_default: false,
};

export default function PaymentMethods() {
  const [methods, setMethods] = useState<PaymentMethod[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingMethod, setEditingMethod] = useState<PaymentMethod | null>(null);
  const [form, setForm] = useState<PaymentMethodInput>(emptyForm);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  async function refresh() {
    try {
      setMethods(await listPaymentMethods());
    } catch {
      setError("Could not load payment methods.");
    }
  }

  useEffect(() => {
    refresh();
  }, []);

  function openCreateModal() {
    setEditingMethod(null);
    setForm(emptyForm);
    setSaveError(null);
    setIsModalOpen(true);
  }

  function openEditModal(method: PaymentMethod) {
    setEditingMethod(method);
    setForm({
      brand: method.brand,
      last4: method.last4,
      exp_month: method.exp_month,
      exp_year: method.exp_year,
      is_default: method.is_default,
    });
    setSaveError(null);
    setIsModalOpen(true);
  }

  function update<K extends keyof PaymentMethodInput>(key: K, value: PaymentMethodInput[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  async function handleSave() {
    if (!form.brand.trim()) {
      setSaveError("Card brand is required.");
      return;
    }
    if (!/^\d{4}$/.test(form.last4)) {
      setSaveError("Last 4 digits must be exactly 4 numbers.");
      return;
    }
    setIsSaving(true);
    setSaveError(null);
    try {
      if (editingMethod) {
        await updatePaymentMethod(editingMethod.id, form);
      } else {
        await createPaymentMethod(form);
      }
      setIsModalOpen(false);
      await refresh();
    } catch {
      setSaveError(editingMethod ? "Could not save changes. Try again." : "Could not add card. Try again.");
    } finally {
      setIsSaving(false);
    }
  }

  async function handleDelete(method: PaymentMethod) {
    if (!window.confirm(`Remove the ${method.brand} card ending in ${method.last4}?`)) {
      return;
    }
    try {
      await deletePaymentMethod(method.id);
      await refresh();
    } catch {
      setError("Could not remove payment method.");
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-end">
        <Button variant="primary" onClick={openCreateModal}>
          <Icon name="plus" size={16} />
          Add Card
        </Button>
      </div>

      {error && (
        <Text size="sm" tone="danger">
          {error}
        </Text>
      )}

      {methods === null ? (
        <Text size="sm" tone="muted">
          Loading…
        </Text>
      ) : methods.length === 0 ? (
        <EmptyState icon="payment" title="No payment methods yet" description="Add a card to use at checkout." />
      ) : (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          {methods.map((method) => (
            <Card key={method.id} className="p-5">
              <div className="flex items-start justify-between gap-3">
                <div className="flex items-center gap-2">
                  <Icon name="payment" size={18} className="text-stone-500" />
                  <Text className="font-medium">{method.brand}</Text>
                  {method.is_default && <Badge variant="brand">Default</Badge>}
                </div>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label="Edit payment method"
                    title="Edit payment method"
                    onClick={() => openEditModal(method)}
                  >
                    <Icon name="pencil" size={15} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    aria-label="Remove payment method"
                    title="Remove payment method"
                    onClick={() => handleDelete(method)}
                    className="text-danger-600 hover:bg-danger-50"
                  >
                    <Icon name="trash" size={15} />
                  </Button>
                </div>
              </div>
              <div className="mt-3 flex flex-col gap-0.5 text-sm text-stone-600">
                <span>•••• •••• •••• {method.last4}</span>
                <span>
                  Expires {String(method.exp_month).padStart(2, "0")}/{method.exp_year}
                </span>
              </div>
            </Card>
          ))}
        </div>
      )}

      <Modal open={isModalOpen} onClose={() => setIsModalOpen(false)} title={editingMethod ? "Edit Card" : "Add Card"}>
        <div className="flex flex-col gap-4">
          {saveError && (
            <Text size="sm" tone="danger">
              {saveError}
            </Text>
          )}
          <Text size="sm" tone="muted">
            For your security, we only store the card brand, last 4 digits, and expiry — never the full card number.
          </Text>
          <FormField label="Card brand" htmlFor="payment-brand">
            <Select id="payment-brand" value={form.brand} onChange={(e) => update("brand", e.target.value)}>
              <option value="">Choose a brand…</option>
              {brands.map((brand) => (
                <option key={brand} value={brand}>
                  {brand}
                </option>
              ))}
            </Select>
          </FormField>
          <FormField label="Last 4 digits" htmlFor="payment-last4">
            <Input
              id="payment-last4"
              value={form.last4}
              onChange={(e) => update("last4", e.target.value.replace(/\D/g, "").slice(0, 4))}
              placeholder="4242"
              inputMode="numeric"
              maxLength={4}
            />
          </FormField>
          <div className="grid grid-cols-2 gap-4">
            <FormField label="Expiry month" htmlFor="payment-exp-month">
              <Select
                id="payment-exp-month"
                value={form.exp_month}
                onChange={(e) => update("exp_month", Number(e.target.value))}
              >
                {Array.from({ length: 12 }, (_, i) => i + 1).map((month) => (
                  <option key={month} value={month}>
                    {String(month).padStart(2, "0")}
                  </option>
                ))}
              </Select>
            </FormField>
            <FormField label="Expiry year" htmlFor="payment-exp-year">
              <Select
                id="payment-exp-year"
                value={form.exp_year}
                onChange={(e) => update("exp_year", Number(e.target.value))}
              >
                {yearOptions.map((year) => (
                  <option key={year} value={year}>
                    {year}
                  </option>
                ))}
              </Select>
            </FormField>
          </div>
          <Checkbox
            id="payment-is-default"
            label="Set as default payment method"
            checked={form.is_default}
            onChange={(e) => update("is_default", e.target.checked)}
          />
        </div>

        <div className="mt-6 flex justify-end gap-3">
          <Button variant="outline" onClick={() => setIsModalOpen(false)} disabled={isSaving}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={isSaving}>
            {isSaving ? "Saving…" : "Save"}
          </Button>
        </div>
      </Modal>
    </div>
  );
}
