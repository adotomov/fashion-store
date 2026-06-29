import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router";

import { Button, buttonStyles } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Input } from "../../components/ui/Input";
import { Select } from "../../components/ui/Select";
import { Heading, Text } from "../../components/ui/Text";
import {
  type Card as CardInput,
  type CheckoutAddress,
  type Contact,
  type DeliveryMethod,
  type DeliveryMethodCode,
  type PaymentMethodCode,
  type PlacedOrder,
  listDeliveryMethods,
  placeOrder,
} from "../../lib/api/checkout";
import { type Office, listOffices } from "../../lib/api/admin-logistics";
import { type Address, listAddresses } from "../../lib/api/users";
import { COUNTRIES } from "../../lib/data/countries";
import { formatMoney } from "../../lib/money/money";
import { useAuth } from "../auth/AuthContext";
import { useCart } from "../cart/CartContext";

type Step = "details" | "delivery" | "payment" | "confirmation";

const emptyAddress: CheckoutAddress = {
  recipient_name: "",
  phone: "",
  line1: "",
  line2: "",
  city: "",
  region: "",
  postal_code: "",
  country_code: "",
};

function addressFromSaved(a: Address): CheckoutAddress {
  return {
    recipient_name: a.recipient_name,
    phone: a.phone,
    line1: a.line1,
    line2: a.line2,
    city: a.city,
    region: a.region,
    postal_code: a.postal_code,
    country_code: a.country_code,
  };
}

const paymentMethodLabels: Record<PaymentMethodCode, string> = {
  cash_on_delivery: "Cash on Delivery",
  card_on_easybox: "Card on EasyBox Pickup",
  card_online: "Pay by Card Online",
};

const paymentMethodDescriptions: Record<PaymentMethodCode, string> = {
  cash_on_delivery: "Pay in cash when your courier delivers the order.",
  card_on_easybox: "Pay by card at the locker when you collect your order.",
  card_online: "Pay securely now with your card.",
};

export function CheckoutFlow() {
  const { isAuthenticated, profile } = useAuth();
  const { cart, refresh: refreshCart } = useCart();
  const location = useLocation();

  const [step, setStep] = useState<Step>("details");

  const [contact, setContact] = useState<Contact>({ full_name: "", email: "", phone: "" });
  const [shippingAddress, setShippingAddress] = useState<CheckoutAddress>(emptyAddress);
  const [billingSameAsShipping, setBillingSameAsShipping] = useState(true);
  const [billingAddress, setBillingAddress] = useState<CheckoutAddress>(emptyAddress);

  const [savedAddresses, setSavedAddresses] = useState<Address[]>([]);
  const [selectedSavedAddressId, setSelectedSavedAddressId] = useState<string>("");

  const [deliveryMethods, setDeliveryMethods] = useState<DeliveryMethod[] | null>(null);
  const [deliveryMethod, setDeliveryMethod] = useState<DeliveryMethodCode | null>(null);
  const [offices, setOffices] = useState<Office[] | null>(null);
  const [officeId, setOfficeId] = useState<string>("");
  const [officesError, setOfficesError] = useState<string | null>(null);

  const [paymentMethod, setPaymentMethod] = useState<PaymentMethodCode | null>(null);
  const [card, setCard] = useState<CardInput>({ number: "", exp_month: 1, exp_year: new Date().getFullYear(), cvv: "" });

  const [isPlacing, setIsPlacing] = useState(false);
  const [placeError, setPlaceError] = useState<string | null>(null);
  const [placedOrder, setPlacedOrder] = useState<PlacedOrder | null>(null);
  const [detailsError, setDetailsError] = useState<string | null>(null);

  useEffect(() => {
    listDeliveryMethods()
      .then(setDeliveryMethods)
      .catch(() => setDeliveryMethods([]));
  }, []);

  useEffect(() => {
    if (!isAuthenticated) return;
    setContact({ full_name: profile?.full_name ?? "", email: profile?.email ?? "", phone: profile?.phone ?? "" });
    listAddresses()
      .then((addresses) => {
        setSavedAddresses(addresses);
        const fallback = addresses.find((a) => a.is_default) ?? addresses[0];
        if (fallback) {
          setSelectedSavedAddressId(fallback.id);
          setShippingAddress(addressFromSaved(fallback));
        }
      })
      .catch(() => setSavedAddresses([]));
  }, [isAuthenticated, profile]);

  useEffect(() => {
    if (deliveryMethod !== "easybox" || !shippingAddress.city.trim()) {
      setOffices(null);
      setOfficeId("");
      return;
    }
    setOffices(null);
    setOfficesError(null);
    listOffices("speedy", shippingAddress.city, "APT")
      .then((result) => {
        setOffices(result);
        setOfficeId((current) => (result.some((o) => o.id === current) ? current : ""));
      })
      .catch(() => setOfficesError("Could not load lockers for this city."));
  }, [deliveryMethod, shippingAddress.city]);

  const items = cart?.items ?? [];
  const subtotal = cart?.subtotal ?? { amount: 0, currency: "EUR" };
  const selectedDeliveryMethod = deliveryMethods?.find((m) => m.code === deliveryMethod) ?? null;
  const grandTotal = {
    amount: subtotal.amount + (selectedDeliveryMethod?.fee.amount ?? 0),
    currency: subtotal.currency,
  };

  function selectSavedAddress(id: string) {
    setSelectedSavedAddressId(id);
    const found = savedAddresses.find((a) => a.id === id);
    if (found) setShippingAddress(addressFromSaved(found));
  }

  function validateDetails(): boolean {
    if (!contact.full_name.trim() || !contact.email.trim()) {
      setDetailsError("Full name and email are required.");
      return false;
    }
    if (!shippingAddress.line1.trim() || !shippingAddress.city.trim() ||
      !shippingAddress.postal_code.trim() || shippingAddress.country_code.trim().length !== 2) {
      setDetailsError("A complete shipping address with a country is required.");
      return false;
    }
    if (!billingSameAsShipping) {
      if (!billingAddress.line1.trim() || !billingAddress.city.trim() ||
        !billingAddress.postal_code.trim() || billingAddress.country_code.trim().length !== 2) {
        setDetailsError("A complete billing address with a country is required.");
        return false;
      }
    }
    setDetailsError(null);
    return true;
  }

  // The address itself no longer collects a separate recipient name/phone —
  // the contact's full name and phone double as the delivery recipient.
  function withRecipient(address: CheckoutAddress): CheckoutAddress {
    return { ...address, recipient_name: contact.full_name, phone: contact.phone };
  }

  async function submitOrder() {
    if (!deliveryMethod || !paymentMethod) return;
    setIsPlacing(true);
    setPlaceError(null);
    try {
      const order = await placeOrder({
        contact,
        shipping_address: withRecipient(shippingAddress),
        billing_address: withRecipient(billingSameAsShipping ? shippingAddress : billingAddress),
        delivery_method: deliveryMethod,
        delivery_office_id: deliveryMethod === "easybox" ? officeId : undefined,
        payment_method: paymentMethod,
        card: paymentMethod === "card_online" ? card : undefined,
      });
      setPlacedOrder(order);
      setStep("confirmation");
      await refreshCart();
    } catch {
      setPlaceError(
        paymentMethod === "card_online"
          ? "Payment could not be processed. Please check your card details and try again."
          : "Could not place your order. Please try again.",
      );
    } finally {
      setIsPlacing(false);
    }
  }

  if (items.length === 0 && !placedOrder) {
    return (
      <div className="flex flex-col items-center gap-4 py-16 text-center">
        <Text tone="muted">Your cart is empty.</Text>
        <Link to="/shop" className={buttonStyles({ variant: "primary" })}>
          Continue Shopping
        </Link>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1fr_320px]">
      <div className="flex flex-col gap-6">
        <StepIndicator step={step} />

        {step === "details" && (
          <Card className="p-6">
            <Heading as="h2" size="sm">
              Contact &amp; Shipping
            </Heading>

            {!isAuthenticated && (
              <div className="mt-3 flex items-center justify-between gap-3 rounded-sm bg-stone-50 px-4 py-3">
                <Text size="sm" tone="muted">
                  Have an account? Sign in for a faster checkout and to track this order.
                </Text>
                <Link
                  to="/login"
                  state={{ from: { pathname: location.pathname, search: location.search } }}
                  className={buttonStyles({ variant: "outline", size: "sm" })}
                >
                  Log In / Register
                </Link>
              </div>
            )}

            <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
              <FormField label="Full name" htmlFor="contact-name">
                <Input
                  id="contact-name"
                  value={contact.full_name}
                  onChange={(e) => setContact((c) => ({ ...c, full_name: e.target.value }))}
                />
              </FormField>
              <FormField label="Email" htmlFor="contact-email">
                <Input
                  id="contact-email"
                  type="email"
                  value={contact.email}
                  onChange={(e) => setContact((c) => ({ ...c, email: e.target.value }))}
                />
              </FormField>
              <FormField label="Phone" htmlFor="contact-phone" hint="Optional">
                <Input
                  id="contact-phone"
                  type="tel"
                  value={contact.phone}
                  onChange={(e) => setContact((c) => ({ ...c, phone: e.target.value }))}
                />
              </FormField>
            </div>

            {isAuthenticated && savedAddresses.length > 0 && (
              <FormField label="Shipping address" htmlFor="saved-address" className="mt-6">
                <Select id="saved-address" value={selectedSavedAddressId} onChange={(e) => selectSavedAddress(e.target.value)}>
                  {savedAddresses.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.label || a.recipient_name} — {a.line1}, {a.city}
                    </option>
                  ))}
                  <option value="">Enter a new address</option>
                </Select>
              </FormField>
            )}

            {(!isAuthenticated || savedAddresses.length === 0 || selectedSavedAddressId === "") && (
              <AddressFields className="mt-6" address={shippingAddress} onChange={setShippingAddress} />
            )}

            <label className="mt-6 flex items-center gap-2 text-sm text-stone-700">
              <input
                type="checkbox"
                checked={billingSameAsShipping}
                onChange={(e) => setBillingSameAsShipping(e.target.checked)}
                className="h-4 w-4 rounded-sm border-stone-300"
              />
              Billing address same as shipping
            </label>

            {!billingSameAsShipping && (
              <>
                <Text className="mt-4 font-medium">Billing address</Text>
                <AddressFields className="mt-2" address={billingAddress} onChange={setBillingAddress} />
              </>
            )}

            {detailsError && (
              <Text size="sm" tone="danger" className="mt-4">
                {detailsError}
              </Text>
            )}

            <div className="mt-6 flex justify-end">
              <Button
                variant="primary"
                onClick={() => {
                  if (validateDetails()) setStep("delivery");
                }}
              >
                Continue to Delivery
              </Button>
            </div>
          </Card>
        )}

        {step === "delivery" && (
          <Card className="p-6">
            <Heading as="h2" size="sm">
              Delivery Method
            </Heading>
            <div className="mt-4 flex flex-col gap-3">
              {(deliveryMethods ?? []).map((method) => (
                <SelectableOption
                  key={method.code}
                  selected={deliveryMethod === method.code}
                  onClick={() => setDeliveryMethod(method.code)}
                  title={method.name}
                  description={method.fee.amount === 0 ? "Free" : formatMoney(method.fee)}
                />
              ))}
            </div>

            {deliveryMethod === "easybox" && (
              <FormField label="Choose a locker" htmlFor="easybox-office" className="mt-4">
                {officesError ? (
                  <Text size="sm" tone="danger">
                    {officesError}
                  </Text>
                ) : !shippingAddress.city.trim() ? (
                  <Text size="sm" tone="muted">
                    Enter a shipping city to see nearby lockers.
                  </Text>
                ) : offices === null ? (
                  <Text size="sm" tone="muted">
                    Loading lockers…
                  </Text>
                ) : offices.length === 0 ? (
                  <Text size="sm" tone="muted">
                    No lockers found for this city.
                  </Text>
                ) : (
                  <Select id="easybox-office" value={officeId} onChange={(e) => setOfficeId(e.target.value)}>
                    <option value="">Select a locker</option>
                    {offices.map((o) => (
                      <option key={o.id} value={o.id}>
                        {o.name}
                      </option>
                    ))}
                  </Select>
                )}
              </FormField>
            )}

            <div className="mt-6 flex justify-between">
              <Button variant="outline" onClick={() => setStep("details")}>
                Back
              </Button>
              <Button
                variant="primary"
                disabled={!deliveryMethod || (deliveryMethod === "easybox" && !officeId)}
                onClick={() => setStep("payment")}
              >
                Continue to Payment
              </Button>
            </div>
          </Card>
        )}

        {step === "payment" && (
          <Card className="p-6">
            <Heading as="h2" size="sm">
              Payment Method
            </Heading>
            <div className="mt-4 flex flex-col gap-3">
              {(Object.keys(paymentMethodLabels) as PaymentMethodCode[]).map((method) => (
                <SelectableOption
                  key={method}
                  selected={paymentMethod === method}
                  onClick={() => setPaymentMethod(method)}
                  title={paymentMethodLabels[method]}
                  description={paymentMethodDescriptions[method]}
                />
              ))}
            </div>

            {paymentMethod === "card_online" && (
              <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
                <FormField label="Card number" htmlFor="card-number" className="sm:col-span-2" hint="Mock payment — any number works, except one ending in 0000.">
                  <Input
                    id="card-number"
                    value={card.number}
                    onChange={(e) => setCard((c) => ({ ...c, number: e.target.value }))}
                    placeholder="4242 4242 4242 4242"
                  />
                </FormField>
                <FormField label="Expiry month" htmlFor="card-exp-month">
                  <Input
                    id="card-exp-month"
                    type="number"
                    min="1"
                    max="12"
                    value={card.exp_month}
                    onChange={(e) => setCard((c) => ({ ...c, exp_month: Number(e.target.value) }))}
                  />
                </FormField>
                <FormField label="Expiry year" htmlFor="card-exp-year">
                  <Input
                    id="card-exp-year"
                    type="number"
                    value={card.exp_year}
                    onChange={(e) => setCard((c) => ({ ...c, exp_year: Number(e.target.value) }))}
                  />
                </FormField>
                <FormField label="CVV" htmlFor="card-cvv">
                  <Input id="card-cvv" value={card.cvv} onChange={(e) => setCard((c) => ({ ...c, cvv: e.target.value }))} />
                </FormField>
              </div>
            )}

            {placeError && (
              <Text size="sm" tone="danger" className="mt-4">
                {placeError}
              </Text>
            )}

            <div className="mt-6 flex justify-between">
              <Button variant="outline" onClick={() => setStep("delivery")} disabled={isPlacing}>
                Back
              </Button>
              {paymentMethod === "card_online" ? (
                <Button variant="primary" onClick={submitOrder} disabled={isPlacing || !card.number.trim()}>
                  {isPlacing ? "Processing payment…" : `Pay ${formatMoney(grandTotal)}`}
                </Button>
              ) : (
                <Button variant="primary" disabled={!paymentMethod} onClick={() => setStep("confirmation")}>
                  Review Order
                </Button>
              )}
            </div>
          </Card>
        )}

        {step === "confirmation" && (
          <Card className="p-6">
            {placedOrder ? (
              <>
                <Heading as="h2" size="sm">
                  Order Placed
                </Heading>
                <Text className="mt-2" tone="muted">
                  Thank you! Your order <span className="font-medium text-stone-900">{placedOrder.order_number}</span> has
                  been {placedOrder.status === "paid" ? "paid and placed" : "placed"}.
                </Text>
                <OrderSummary order={placedOrder} />
              </>
            ) : (
              <>
                <Heading as="h2" size="sm">
                  Review &amp; Complete
                </Heading>
                <div className="mt-4 flex flex-col gap-2 text-sm text-stone-700">
                  <SummaryRow label="Delivery" value={selectedDeliveryMethod?.name ?? ""} />
                  <SummaryRow label="Payment" value={paymentMethod ? paymentMethodLabels[paymentMethod] : ""} />
                  <SummaryRow label="Subtotal" value={formatMoney(subtotal)} />
                  <SummaryRow
                    label="Delivery fee"
                    value={selectedDeliveryMethod?.fee.amount ? formatMoney(selectedDeliveryMethod.fee) : "Free"}
                  />
                  <SummaryRow label="Total" value={formatMoney(grandTotal)} emphasize />
                </div>
                {placeError && (
                  <Text size="sm" tone="danger" className="mt-4">
                    {placeError}
                  </Text>
                )}
                <div className="mt-6 flex justify-between">
                  <Button variant="outline" onClick={() => setStep("payment")} disabled={isPlacing}>
                    Back
                  </Button>
                  <Button variant="primary" onClick={submitOrder} disabled={isPlacing}>
                    {isPlacing ? "Placing order…" : "Complete Order"}
                  </Button>
                </div>
              </>
            )}
          </Card>
        )}
      </div>

      <Card className="h-fit p-6">
        <Heading as="h2" size="sm">
          Order Summary
        </Heading>
        <ul className="mt-4 flex flex-col gap-3">
          {items.map((item) => (
            <li key={item.id} className="flex items-center justify-between text-sm">
              <span className="text-stone-700">
                {item.product_name}
                {item.variant_label ? ` — ${item.variant_label}` : ""}
                <span className="ml-2 text-stone-400">× {item.quantity}</span>
              </span>
              <span className="text-stone-600">{formatMoney(item.line_total)}</span>
            </li>
          ))}
        </ul>
        <div className="mt-4 flex items-center justify-between border-t border-stone-200 pt-4">
          <Text size="sm" className="font-medium">
            Subtotal
          </Text>
          <Text size="sm" className="font-medium">
            {formatMoney(subtotal)}
          </Text>
        </div>
        <div className="mt-2 flex items-center justify-between">
          <Text size="sm" tone="muted">
            Delivery
          </Text>
          <Text size="sm" tone="muted">
            {selectedDeliveryMethod ? (selectedDeliveryMethod.fee.amount ? formatMoney(selectedDeliveryMethod.fee) : "Free") : "–"}
          </Text>
        </div>
        <div className="mt-2 flex items-center justify-between">
          <Text className="font-medium">Total</Text>
          <Text className="font-medium">{formatMoney(grandTotal)}</Text>
        </div>
      </Card>
    </div>
  );
}

function StepIndicator({ step }: { step: Step }) {
  const steps: { key: Step; label: string }[] = [
    { key: "details", label: "Details" },
    { key: "delivery", label: "Delivery" },
    { key: "payment", label: "Payment" },
    { key: "confirmation", label: "Review" },
  ];
  const currentIndex = steps.findIndex((s) => s.key === step);
  return (
    <div className="flex items-center gap-2 text-sm">
      {steps.map((s, i) => (
        <div key={s.key} className="flex items-center gap-2">
          <span
            className={
              i <= currentIndex
                ? "font-medium text-stone-900"
                : "text-stone-400"
            }
          >
            {i + 1}. {s.label}
          </span>
          {i < steps.length - 1 && <span className="text-stone-300">›</span>}
        </div>
      ))}
    </div>
  );
}

function SelectableOption({
  selected,
  onClick,
  title,
  description,
}: {
  selected: boolean;
  onClick: () => void;
  title: string;
  description: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex items-center justify-between rounded-sm border px-4 py-3 text-left transition-colors ${
        selected ? "border-stone-900 bg-stone-50" : "border-stone-200 hover:bg-stone-50"
      }`}
    >
      <div>
        <Text className="font-medium">{title}</Text>
        <Text size="sm" tone="muted">
          {description}
        </Text>
      </div>
      <span
        className={`h-4 w-4 shrink-0 rounded-full border ${
          selected ? "border-stone-900 bg-stone-900" : "border-stone-300"
        }`}
      />
    </button>
  );
}

function SummaryRow({ label, value, emphasize }: { label: string; value: string; emphasize?: boolean }) {
  return (
    <div className="flex items-center justify-between">
      <span className={emphasize ? "font-medium text-stone-900" : "text-stone-600"}>{label}</span>
      <span className={emphasize ? "font-medium text-stone-900" : "text-stone-600"}>{value}</span>
    </div>
  );
}

function OrderSummary({ order }: { order: PlacedOrder }) {
  return (
    <div className="mt-6 flex flex-col gap-2 text-sm">
      {order.items.map((item, i) => (
        <div key={i} className="flex items-center justify-between">
          <span className="text-stone-700">
            {item.product_name}
            {item.variant_label ? ` — ${item.variant_label}` : ""}
            <span className="ml-2 text-stone-400">× {item.quantity}</span>
          </span>
          <span className="text-stone-600">{formatMoney(item.unit_price)}</span>
        </div>
      ))}
      <div className="mt-2 flex items-center justify-between border-t border-stone-200 pt-2 font-medium text-stone-900">
        <span>Total</span>
        <span>{formatMoney(order.total)}</span>
      </div>
    </div>
  );
}

function AddressFields({
  address,
  onChange,
  className,
}: {
  address: CheckoutAddress;
  onChange: (a: CheckoutAddress) => void;
  className?: string;
}) {
  function update<K extends keyof CheckoutAddress>(key: K, value: CheckoutAddress[K]) {
    onChange({ ...address, [key]: value });
  }

  return (
    <div className={`grid grid-cols-1 gap-4 sm:grid-cols-2 ${className ?? ""}`}>
      <FormField label="Address line 1" htmlFor="addr-line1" className="sm:col-span-2">
        <Input id="addr-line1" value={address.line1} onChange={(e) => update("line1", e.target.value)} />
      </FormField>
      <FormField label="Address line 2" htmlFor="addr-line2" hint="Optional" className="sm:col-span-2">
        <Input id="addr-line2" value={address.line2} onChange={(e) => update("line2", e.target.value)} />
      </FormField>
      <FormField label="City" htmlFor="addr-city">
        <Input id="addr-city" value={address.city} onChange={(e) => update("city", e.target.value)} />
      </FormField>
      <FormField label="Region / State" htmlFor="addr-region" hint="Optional">
        <Input id="addr-region" value={address.region} onChange={(e) => update("region", e.target.value)} />
      </FormField>
      <FormField label="Postal code" htmlFor="addr-postal-code">
        <Input id="addr-postal-code" value={address.postal_code} onChange={(e) => update("postal_code", e.target.value)} />
      </FormField>
      <FormField label="Country" htmlFor="addr-country-code">
        <Select id="addr-country-code" value={address.country_code} onChange={(e) => update("country_code", e.target.value)}>
          <option value="">Select a country</option>
          {COUNTRIES.map((country) => (
            <option key={country.code} value={country.code}>
              {country.name}
            </option>
          ))}
        </Select>
      </FormField>
    </div>
  );
}
