import { useEffect, useRef, useState } from "react";
import { Link, useLocation } from "react-router";

import { Button, buttonStyles } from "../../components/ui/Button";
import { Card } from "../../components/ui/Card";
import { FormField } from "../../components/ui/FormField";
import { Input } from "../../components/ui/Input";
import { Select } from "../../components/ui/Select";
import { Heading, Text } from "../../components/ui/Text";
import {
  type CheckoutAddress,
  type Contact,
  type DeliveryMethod,
  type DeliveryMethodCode,
  type DiscountCodeValidation,
  type PaymentInitiation,
  type PaymentMethodCode,
  type PlacedOrder,
  cancelPayment,
  listDeliveryMethods,
  placeOrder,
  releaseCheckoutSession,
  reserveCheckoutSession,
  validateDiscountCode,
} from "../../lib/api/checkout";
import { ApiError } from "../../lib/api/client";
import { RevolutPaymentStep } from "./RevolutPaymentStep";
import { AddressForm, type StructuredAddress, emptyStructuredAddress, isStructuredAddressComplete } from "../../components/address/AddressForm";
import { Combobox, type ComboboxOption } from "../../components/ui/Combobox";
import { type Office, searchOffices, searchSites } from "../../lib/api/logistics";
import { type Address, listAddresses } from "../../lib/api/users";
import { formatMoneyDual } from "../../lib/money/money";
import { useAuth } from "../auth/AuthContext";
import { useCart } from "../cart/CartContext";
import { useLanguage } from "../i18n/LanguageContext";
import { useStoreBranding } from "../store-settings/StoreSettingsContext";

type Step = "details" | "delivery" | "payment" | "confirmation";

const emptyAddress: CheckoutAddress = emptyStructuredAddress;

// officeTypeFor maps a delivery method to the Speedy location type its picker
// searches, or null for door delivery (which uses the shipping address). Speedy
// Office pickup uses "OFFICE"; EasyBox lockers use "APT".
function officeTypeFor(method: DeliveryMethodCode | null): "OFFICE" | "APT" | null {
  if (method === "speedy_office") return "OFFICE";
  if (method === "easybox") return "APT";
  return null;
}

function addressFromSaved(a: Address): CheckoutAddress {
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

export function CheckoutFlow() {
  const { t } = useLanguage();
  const { storeLocale } = useStoreBranding();
  const { isAuthenticated, profile } = useAuth();
  const { cart, refresh: refreshCart } = useCart();
  const location = useLocation();

  const paymentMethodLabels: Record<PaymentMethodCode, string> = {
    cash_on_delivery: t("checkout.cash_on_delivery", "Pay on Delivery"),
    card_on_easybox: t("checkout.card_on_easybox", "Pay on Terminal"),
    card_online: t("checkout.card_online", "Pay by Card Online"),
  };

  const paymentMethodDescriptions: Record<PaymentMethodCode, string> = {
    cash_on_delivery: t("checkout.cash_on_delivery_desc", "Pay by card or cash to the courier on delivery, or at a Speedy office."),
    card_on_easybox: t("checkout.card_on_easybox_desc", "Pay by card on the locker's terminal when you collect your order."),
    card_online: t("checkout.card_online_desc", "Pay securely now with your card."),
  };

  const [step, setStep] = useState<Step>("details");
  // Scroll the checkout content back to the top whenever the shopper advances
  // (or steps back) so the next step starts in view rather than wherever the
  // previous step's scroll position left them — matters most on mobile.
  const contentRef = useRef<HTMLDivElement>(null);
  const didMountStepRef = useRef(false);
  useEffect(() => {
    if (!didMountStepRef.current) {
      didMountStepRef.current = true;
      return; // don't yank the page on first render
    }
    contentRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, [step]);

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
  // The city the locker picker searches in — defaults to the shipping address's
  // city but the shopper can change it independently (they may collect from a
  // locker in a different town). Held separately so it doesn't mutate the
  // shipping address.
  const [officeSite, setOfficeSite] = useState<{ id: number; name: string }>({ id: 0, name: "" });

  const [paymentMethod, setPaymentMethod] = useState<PaymentMethodCode | null>(null);
  // Set once an online-card order is created and awaiting widget payment.
  const [paymentInitiation, setPaymentInitiation] = useState<PaymentInitiation | null>(null);
  const [pendingSnapshot, setPendingSnapshot] = useState<PlacedOrder | null>(null);

  const [discountCodeInput, setDiscountCodeInput] = useState("");
  const [appliedDiscount, setAppliedDiscount] = useState<DiscountCodeValidation | null>(null);
  const [discountError, setDiscountError] = useState<string | null>(null);
  const [isValidatingCode, setIsValidatingCode] = useState(false);

  const [isPlacing, setIsPlacing] = useState(false);
  const [placeError, setPlaceError] = useState<string | null>(null);
  const [placedOrder, setPlacedOrder] = useState<PlacedOrder | null>(null);
  const [detailsError, setDetailsError] = useState<string | null>(null);
  // Set when the checkout-session stock hold can't be acquired because the
  // items are out of stock — blocks the whole flow with a clear message.
  const [reserveOutOfStock, setReserveOutOfStock] = useState(false);

  // Tracks whether we hold a checkout-session reservation, so the unmount
  // cleanup can release it. Not released while an order is placed or a card
  // payment is in flight (paymentInitiation) — then the hold is owned by the
  // order/settlement, and the server sweeper is the safety net.
  const holdActiveRef = useRef(false);
  const orderInFlightRef = useRef(false);
  orderInFlightRef.current = paymentInitiation !== null || placedOrder !== null;

  useEffect(() => {
    listDeliveryMethods()
      .then(setDeliveryMethods)
      .catch(() => setDeliveryMethods([]));
  }, []);

  // Reserve stock for the whole checkout the moment the shopper arrives with a
  // non-empty cart — so a slow-filling shopper isn't beaten to the last unit at
  // the payment step, and switching payment methods never re-touches stock. A
  // second shopper for the same last unit is told out-of-stock here, up front.
  useEffect(() => {
    if (holdActiveRef.current || reserveOutOfStock) return;
    if (!cart || (cart.items?.length ?? 0) === 0) return;
    if (placedOrder || paymentInitiation) return;
    holdActiveRef.current = true;
    reserveCheckoutSession().catch((err) => {
      holdActiveRef.current = false;
      if (err instanceof ApiError && (err.status === 409 || err.code === "insufficient_stock")) {
        setReserveOutOfStock(true);
      }
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [cart]);

  // Best-effort release of the hold if the shopper leaves checkout without
  // completing (and no card payment is mid-flight); the server sweeper reclaims
  // it otherwise.
  useEffect(() => {
    return () => {
      if (holdActiveRef.current && !orderInFlightRef.current) {
        void releaseCheckoutSession();
      }
    };
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

  // When an office/locker step opens, default its city to the resolved shipping city.
  useEffect(() => {
    if (officeTypeFor(deliveryMethod) && officeSite.id === 0 && shippingAddress.site_id > 0) {
      setOfficeSite({ id: shippingAddress.site_id, name: shippingAddress.city });
    }
  }, [deliveryMethod, shippingAddress.site_id, shippingAddress.city, officeSite.id]);

  useEffect(() => {
    const officeType = officeTypeFor(deliveryMethod);
    if (!officeType || officeSite.id === 0) {
      setOffices(null);
      setOfficeId("");
      return;
    }
    setOffices(null);
    setOfficesError(null);
    searchOffices(officeSite.id, officeType)
      .then((result) => {
        setOffices(result);
        setOfficeId((current) => (result.some((o) => o.id === current) ? current : ""));
      })
      .catch(() => setOfficesError(t("checkout.load_offices_error", "Could not load pickup points for this city.")));
  }, [deliveryMethod, officeSite.id]);

  // Payment options depend on the delivery method (a locker can't take cash,
  // a courier has no locker terminal). Clear a selection that no longer fits
  // when the delivery method changes, so we never submit an invalid combo.
  useEffect(() => {
    const allowed = deliveryMethods?.find((m) => m.code === deliveryMethod)?.payment_methods ?? [];
    setPaymentMethod((current) => (current && allowed.includes(current) ? current : null));
  }, [deliveryMethod, deliveryMethods]);

  const items = cart?.items ?? [];
  const subtotal = cart?.subtotal ?? { amount: 0, currency: "EUR" };
  const selectedDeliveryMethod = deliveryMethods?.find((m) => m.code === deliveryMethod) ?? null;
  const allowedPaymentMethods = selectedDeliveryMethod?.payment_methods ?? [];

  // Copy for the pickup-point picker, which serves both Speedy offices and
  // EasyBox lockers depending on the chosen delivery method.
  const pickupType = officeTypeFor(deliveryMethod);
  const pickupCopy =
    pickupType === "OFFICE"
      ? {
          choose: t("checkout.choose_office", "Choose an office"),
          pickCity: t("checkout.pick_city_for_offices", "Pick a city to see nearby offices."),
          loading: t("checkout.loading_offices", "Loading offices…"),
          none: t("checkout.no_offices_found", "No offices found for this city."),
          select: t("checkout.select_office", "Select an office"),
        }
      : {
          choose: t("checkout.choose_locker", "Choose a locker"),
          pickCity: t("checkout.pick_city_for_lockers", "Pick a city to see nearby lockers."),
          loading: t("checkout.loading_lockers", "Loading lockers…"),
          none: t("checkout.no_lockers_found", "No lockers found for this city."),
          select: t("checkout.select_locker", "Select a locker"),
        };
  const discountAmount = appliedDiscount
    ? Math.round(subtotal.amount * (appliedDiscount.value_percent / 100))
    : 0;
  const grandTotal = {
    amount: subtotal.amount - discountAmount + (selectedDeliveryMethod?.fee.amount ?? 0),
    currency: subtotal.currency,
  };

  // Once an online-card order is initiated the server clears the cart, so the
  // live `cart` is empty while the Revolut widget is on screen. Drive the order
  // summary (and totals) from the snapshot captured at initiation instead.
  const payingByCard = paymentInitiation !== null && pendingSnapshot !== null;
  const summaryLines = payingByCard
    ? pendingSnapshot!.items.map((it, i) => ({
        key: `snap-${i}`,
        product_name: it.product_name,
        variant_label: it.variant_label,
        quantity: it.quantity,
        line_total: { amount: it.unit_price.amount * it.quantity, currency: it.unit_price.currency },
      }))
    : items.map((it) => ({
        key: it.id,
        product_name: it.product_name,
        variant_label: it.variant_label,
        quantity: it.quantity,
        line_total: it.line_total,
      }));
  const summarySubtotal = payingByCard
    ? { amount: summaryLines.reduce((sum, l) => sum + l.line_total.amount, 0), currency: grandTotal.currency }
    : subtotal;
  const summaryDeliveryFee = payingByCard ? pendingSnapshot!.delivery_fee : selectedDeliveryMethod?.fee ?? null;
  const summaryDiscount = payingByCard
    ? pendingSnapshot!.discount_amount
      ? { code: pendingSnapshot!.discount_code ?? "", amount: pendingSnapshot!.discount_amount }
      : null
    : appliedDiscount
      ? { code: appliedDiscount.code, amount: { amount: discountAmount, currency: subtotal.currency } }
      : null;
  const summaryTotal = payingByCard ? pendingSnapshot!.total : grandTotal;

  async function applyDiscountCode() {
    const code = discountCodeInput.trim();
    if (!code) return;
    setIsValidatingCode(true);
    setDiscountError(null);
    setAppliedDiscount(null);
    try {
      const result = await validateDiscountCode(code);
      if (!result.valid) {
        setDiscountError(t("checkout.invalid_discount", "This discount code is invalid or has expired."));
      } else {
        setAppliedDiscount(result);
      }
    } catch {
      setDiscountError(t("checkout.discount_error", "Could not validate the discount code."));
    } finally {
      setIsValidatingCode(false);
    }
  }

  function selectSavedAddress(id: string) {
    setSelectedSavedAddressId(id);
    const found = savedAddresses.find((a) => a.id === id);
    if (found) setShippingAddress(addressFromSaved(found));
  }

  function validateDetails(): boolean {
    if (!contact.full_name.trim() || !contact.email.trim() || !contact.phone.trim()) {
      setDetailsError(t("checkout.contact_required_error", "Full name, email and phone number are required."));
      return false;
    }
    if (!isStructuredAddressComplete(shippingAddress)) {
      setDetailsError(t("checkout.shipping_required_error", "Select a city, neighbourhood and street for the shipping address."));
      return false;
    }
    if (!billingSameAsShipping && !isStructuredAddressComplete(billingAddress)) {
      setDetailsError(t("checkout.billing_required_error", "Select a city, neighbourhood and street for the billing address."));
      return false;
    }
    setDetailsError(null);
    return true;
  }

  function withRecipient(address: CheckoutAddress): CheckoutAddress {
    return { ...address, recipient_name: contact.full_name, phone: contact.phone };
  }

  // Build the confirmation snapshot for a card order from the current cart, so
  // the confirmation screen can render even though the cart is cleared once the
  // (pending_payment) order is created.
  function buildPendingSnapshot(initiation: PaymentInitiation): PlacedOrder {
    return {
      id: initiation.order_id,
      order_number: initiation.order_number,
      status: "pending_payment",
      total: grandTotal,
      delivery_method: deliveryMethod as DeliveryMethodCode,
      delivery_fee: selectedDeliveryMethod?.fee ?? { amount: 0, currency: grandTotal.currency },
      payment_method: "card_online",
      placed_at: new Date().toISOString(),
      discount_code: appliedDiscount?.code,
      discount_amount: appliedDiscount ? { amount: discountAmount, currency: subtotal.currency } : undefined,
      items: items.map((item) => ({
        product_name: item.product_name,
        variant_label: item.variant_label,
        quantity: item.quantity,
        unit_price: item.unit_price,
      })),
    };
  }

  async function submitOrder() {
    if (!deliveryMethod || !paymentMethod) return;
    setIsPlacing(true);
    setPlaceError(null);
    try {
      const result = await placeOrder({
        contact,
        shipping_address: withRecipient(shippingAddress),
        billing_address: withRecipient(billingSameAsShipping ? shippingAddress : billingAddress),
        delivery_method: deliveryMethod,
        delivery_office_id: officeTypeFor(deliveryMethod) ? officeId : undefined,
        payment_method: paymentMethod,
        discount_code: appliedDiscount?.code,
      });
      if (result.kind === "payment_required") {
        // Card order created; render the Revolut widget to collect payment.
        setPendingSnapshot(buildPendingSnapshot(result.initiation));
        setPaymentInitiation(result.initiation);
        await refreshCart();
      } else {
        setPlacedOrder(result.order);
        setStep("confirmation");
        await refreshCart();
      }
    } catch {
      setPlaceError(
        paymentMethod === "card_online"
          ? t("checkout.start_payment_error", "Could not start the payment. Please try again.")
          : t("checkout.place_order_error", "Could not place your order. Please try again."),
      );
    } finally {
      setIsPlacing(false);
    }
  }

  function handleCardPaid() {
    setPlacedOrder(pendingSnapshot ? { ...pendingSnapshot, status: "paid" } : null);
    setPaymentInitiation(null);
    setStep("confirmation");
    // The webhook clears the cart server-side on settlement; refresh so the
    // header/cart badge reflect the now-empty cart.
    void refreshCart();
  }

  function handleCardFailed(message: string) {
    setPaymentInitiation(null);
    setPlaceError(message);
    // Payment failed after the order was created: the reservation is released
    // and the cart was never cleared, so the customer can retry. Reflect that.
    void refreshCart();
  }

  // Back out of an initiated card payment to pick another method: cancel the
  // pending order server-side (releases stock, fails the order) and return to
  // the payment step. The cart was never cleared, so it's still there.
  const [isCancelling, setIsCancelling] = useState(false);
  async function cancelCardPayment() {
    if (!paymentInitiation) return;
    setIsCancelling(true);
    try {
      await cancelPayment(paymentInitiation.order_number, paymentInitiation.revolut_order_id);
    } catch {
      // Even if the cancel call fails, the sweeper will reclaim the pending
      // order; let the customer move on rather than trapping them here.
    } finally {
      setIsCancelling(false);
      setPaymentInitiation(null);
      setPendingSnapshot(null);
      setPaymentMethod(null);
      setPlaceError(null);
      await refreshCart();
    }
  }

  if (items.length === 0 && !placedOrder && !paymentInitiation) {
    return (
      <div className="flex flex-col items-center gap-4 py-16 text-center">
        <Text tone="muted">{t("cart.empty", "Your cart is empty")}</Text>
        <Link to="/shop" className={buttonStyles({ variant: "primary" })}>
          {t("common.continue_shopping", "Continue Shopping")}
        </Link>
      </div>
    );
  }

  // Couldn't hold the stock for checkout — one or more items are out of stock.
  // Send the shopper back to the cart to remove them before trying again.
  if (reserveOutOfStock && !placedOrder && !paymentInitiation) {
    return (
      <div className="flex flex-col items-center gap-4 py-16 text-center">
        <Text tone="danger" className="font-medium">
          {t("checkout.out_of_stock_title", "Some items are no longer in stock")}
        </Text>
        <Text size="sm" tone="muted">
          {t("checkout.out_of_stock_desc", "We couldn't reserve everything in your cart. Please review and remove the out-of-stock items to continue.")}
        </Text>
        <Link to="/cart" className={buttonStyles({ variant: "primary" })}>
          {t("checkout.back_to_cart", "Back to Cart")}
        </Link>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 gap-10 lg:grid-cols-[1fr_320px]">
      <div ref={contentRef} className="flex flex-col gap-6 scroll-mt-24">
        <StepIndicator step={step} />

        {step === "details" && (
          <Card className="p-6">
            <Heading as="h2" size="sm">
              {t("checkout.contact_shipping", "Contact & Shipping")}
            </Heading>

            {!isAuthenticated && (
              <div className="mt-3 flex items-center justify-between gap-3 rounded-sm bg-stone-50 px-4 py-3">
                <Text size="sm" tone="muted">
                  {t("checkout.signin_prompt", "Have an account? Sign in for a faster checkout and to track this order.")}
                </Text>
                <Link
                  to="/login"
                  state={{ from: { pathname: location.pathname, search: location.search } }}
                  className={buttonStyles({ variant: "outline", size: "sm" })}
                >
                  {t("checkout.login_register", "Log In / Register")}
                </Link>
              </div>
            )}

            <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
              <FormField label={t("common.full_name", "Full name")} htmlFor="contact-name">
                <Input
                  id="contact-name"
                  value={contact.full_name}
                  onChange={(e) => setContact((c) => ({ ...c, full_name: e.target.value }))}
                />
              </FormField>
              <FormField label={t("common.email", "Email")} htmlFor="contact-email">
                <Input
                  id="contact-email"
                  type="email"
                  value={contact.email}
                  onChange={(e) => setContact((c) => ({ ...c, email: e.target.value }))}
                />
              </FormField>
              <FormField label={t("common.phone", "Phone")} htmlFor="contact-phone">
                <Input
                  id="contact-phone"
                  type="tel"
                  required
                  value={contact.phone}
                  onChange={(e) => setContact((c) => ({ ...c, phone: e.target.value }))}
                />
              </FormField>
            </div>

            {isAuthenticated && savedAddresses.length > 0 && (
              <FormField label={t("checkout.shipping_address", "Shipping address")} htmlFor="saved-address" className="mt-6">
                <Select id="saved-address" value={selectedSavedAddressId} onChange={(e) => selectSavedAddress(e.target.value)}>
                  {savedAddresses.map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.label || a.recipient_name} — {[a.street_name, a.street_no].filter(Boolean).join(" ")}, {a.city}
                    </option>
                  ))}
                  <option value="">{t("checkout.enter_new_address", "Enter a new address")}</option>
                </Select>
              </FormField>
            )}

            {(!isAuthenticated || savedAddresses.length === 0 || selectedSavedAddressId === "") && (
              <div className="mt-6">
                <AddressForm value={shippingAddress} onChange={setShippingAddress} idPrefix="ship" />
              </div>
            )}

            <label className="mt-6 flex items-center gap-2 text-sm text-stone-700">
              <input
                type="checkbox"
                checked={billingSameAsShipping}
                onChange={(e) => setBillingSameAsShipping(e.target.checked)}
                className="h-4 w-4 rounded-sm border-stone-300"
              />
              {t("checkout.billing_same_as_shipping", "Billing address same as shipping")}
            </label>

            {!billingSameAsShipping && (
              <>
                <Text className="mt-4 font-medium">{t("checkout.billing_address", "Billing address")}</Text>
                <div className="mt-2">
                  <AddressForm value={billingAddress} onChange={setBillingAddress} idPrefix="bill" />
                </div>
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
                {t("checkout.continue_to_delivery", "Continue to Delivery")}
              </Button>
            </div>
          </Card>
        )}

        {step === "delivery" && (
          <Card className="p-6">
            <Heading as="h2" size="sm">
              {t("checkout.delivery_method", "Delivery Method")}
            </Heading>
            <div className="mt-4 flex flex-col gap-3">
              {(deliveryMethods ?? []).map((method) => (
                <SelectableOption
                  key={method.code}
                  selected={deliveryMethod === method.code}
                  onClick={() => setDeliveryMethod(method.code)}
                  title={method.name}
                  description={method.fee.amount === 0 ? t("checkout.free", "Free") : formatMoneyDual(method.fee, storeLocale)}
                />
              ))}
            </div>

            {pickupType && (
              <div className="mt-4 flex flex-col gap-4">
                <FormField label={t("checkout.pickup_city", "City")} htmlFor="pickup-city">
                  <Combobox
                    id="pickup-city"
                    value={officeSite.name}
                    placeholder={t("address.city_placeholder", "Start typing a city…")}
                    emptyText={t("address.no_cities", "No cities found")}
                    onValueChange={(text) => setOfficeSite({ id: 0, name: text })}
                    onSearch={async (q) =>
                      (await searchSites(q)).map<ComboboxOption>((s) => ({ id: s.id, label: s.name, sublabel: s.region }))
                    }
                    onSelect={(opt) => setOfficeSite({ id: Number(opt.id), name: opt.label })}
                  />
                </FormField>
                <FormField label={pickupCopy.choose} htmlFor="pickup-office">
                  {officesError ? (
                    <Text size="sm" tone="danger">
                      {officesError}
                    </Text>
                  ) : officeSite.id === 0 ? (
                    <Text size="sm" tone="muted">
                      {pickupCopy.pickCity}
                    </Text>
                  ) : offices === null ? (
                    <Text size="sm" tone="muted">
                      {pickupCopy.loading}
                    </Text>
                  ) : offices.length === 0 ? (
                    <Text size="sm" tone="muted">
                      {pickupCopy.none}
                    </Text>
                  ) : (
                    <Select id="pickup-office" value={officeId} onChange={(e) => setOfficeId(e.target.value)}>
                      <option value="">{pickupCopy.select}</option>
                      {offices.map((o) => (
                        <option key={o.id} value={o.id}>
                          {o.name}
                        </option>
                      ))}
                    </Select>
                  )}
                </FormField>
              </div>
            )}

            <div className="mt-6 flex justify-between">
              <Button variant="outline" onClick={() => setStep("details")}>
                {t("common.back", "Back")}
              </Button>
              <Button
                variant="primary"
                disabled={!deliveryMethod || (pickupType !== null && !officeId)}
                onClick={() => setStep("payment")}
              >
                {t("checkout.continue_to_payment", "Continue to Payment")}
              </Button>
            </div>
          </Card>
        )}

        {step === "payment" && (
          <Card className="p-6">
            {paymentInitiation ? (
              <>
                <Heading as="h2" size="sm">
                  {t("checkout.pay_by_card", "Pay by Card")}
                </Heading>
                <Text size="sm" tone="muted" className="mt-1">
                  {t("checkout.pay_amount_prompt", "Enter your card details, or use Apple Pay / Google Pay.")}
                </Text>
                <div className="mt-6">
                  <RevolutPaymentStep
                    token={paymentInitiation.revolut_order_token}
                    orderNumber={paymentInitiation.order_number}
                    cardHolderName={contact.full_name}
                    email={contact.email}
                    payLabel={`${t("checkout.pay", "Pay")} ${formatMoneyDual(paymentInitiation.amount, storeLocale)}`}
                    onPaid={handleCardPaid}
                    onFailed={handleCardFailed}
                  />
                </div>
                <div className="mt-4">
                  <Button variant="ghost" size="sm" onClick={() => void cancelCardPayment()} disabled={isCancelling}>
                    {isCancelling
                      ? t("checkout.cancelling_payment", "Cancelling…")
                      : t("checkout.use_different_method", "Use a different payment method")}
                  </Button>
                </div>
              </>
            ) : (
              <>
                <Heading as="h2" size="sm">
                  {t("checkout.payment_method", "Payment Method")}
                </Heading>
                <div className="mt-4 flex flex-col gap-3">
                  {allowedPaymentMethods.map((method) => (
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
                  <Text size="sm" tone="muted" className="mt-4">
                    {t("checkout.card_online_next_step", "You'll enter your card, or use Apple Pay / Google Pay, on the next step.")}
                  </Text>
                )}

                {placeError && (
                  <Text size="sm" tone="danger" className="mt-4">
                    {placeError}
                  </Text>
                )}

                <div className="mt-6 flex justify-between">
                  <Button variant="outline" onClick={() => setStep("delivery")} disabled={isPlacing}>
                    {t("common.back", "Back")}
                  </Button>
                  {paymentMethod === "card_online" ? (
                    <Button variant="primary" onClick={submitOrder} disabled={isPlacing}>
                      {isPlacing ? t("checkout.starting_payment", "Starting payment…") : t("checkout.continue_to_pay", "Continue to Payment")}
                    </Button>
                  ) : (
                    <Button variant="primary" disabled={!paymentMethod} onClick={() => setStep("confirmation")}>
                      {t("checkout.review_order", "Review Order")}
                    </Button>
                  )}
                </div>
              </>
            )}
          </Card>
        )}

        {step === "confirmation" && (
          <Card className="p-6">
            {placedOrder ? (
              <>
                <Heading as="h2" size="sm">
                  {t("checkout.order_placed", "Order Placed")}
                </Heading>
                <Text className="mt-2" tone="muted">
                  {t("checkout.order_confirmed_prefix", "Thank you! Your order")}{" "}
                  <span className="font-medium text-stone-900">{placedOrder.order_number}</span>{" "}
                  {t("checkout.order_confirmed_suffix", "has been")}{" "}
                  {placedOrder.status === "paid"
                    ? t("checkout.order_paid_placed", "paid and placed")
                    : t("checkout.order_placed_fallback", "placed")}.
                </Text>
                <OrderSummary order={placedOrder} totalLabel={t("checkout.total", "Total")} />
              </>
            ) : (
              <>
                <Heading as="h2" size="sm">
                  {t("checkout.review_complete", "Review & Complete")}
                </Heading>
                <div className="mt-4 flex flex-col gap-2 text-sm text-stone-700">
                  <SummaryRow label={t("checkout.delivery_label", "Delivery")} value={selectedDeliveryMethod?.name ?? ""} />
                  <SummaryRow label={t("checkout.payment_label", "Payment")} value={paymentMethod ? paymentMethodLabels[paymentMethod] : ""} />
                  <SummaryRow label={t("checkout.subtotal", "Subtotal")} value={formatMoneyDual(subtotal, storeLocale)} />
                  {appliedDiscount && (
                    <SummaryRow
                      label={`${t("checkout.discount_label", "Discount")} (${appliedDiscount.code})`}
                      value={`−${formatMoneyDual({ amount: discountAmount, currency: subtotal.currency }, storeLocale)}`}
                    />
                  )}
                  <SummaryRow
                    label={t("checkout.delivery_fee", "Delivery fee")}
                    value={selectedDeliveryMethod?.fee.amount ? formatMoneyDual(selectedDeliveryMethod.fee, storeLocale) : t("checkout.free", "Free")}
                  />
                  <SummaryRow label={t("checkout.total", "Total")} value={formatMoneyDual(grandTotal, storeLocale)} emphasize />
                </div>
                {placeError && (
                  <Text size="sm" tone="danger" className="mt-4">
                    {placeError}
                  </Text>
                )}
                <div className="mt-6 flex justify-between">
                  <Button variant="outline" onClick={() => setStep("payment")} disabled={isPlacing}>
                    {t("common.back", "Back")}
                  </Button>
                  <Button variant="primary" onClick={submitOrder} disabled={isPlacing}>
                    {isPlacing ? t("checkout.placing_order", "Placing order…") : t("checkout.complete_order", "Complete Order")}
                  </Button>
                </div>
              </>
            )}
          </Card>
        )}
      </div>

      <div className="flex flex-col gap-6">
        {/* Spacer matching the step indicator's height (single line of text-sm =
            20px) so the summary drops to align with the step content rather than
            the breadcrumbs. Fixed height (not a clone of StepIndicator, which
            would wrap taller in this 320px column). Only on lg, where the two
            columns sit side by side; collapses when the layout stacks. */}
        <div aria-hidden className="hidden h-5 lg:block" />
        <Card className="h-fit p-6">
          <Heading as="h2" size="sm">
            {t("checkout.order_summary", "Order Summary")}
          </Heading>
        <ul className="mt-4 flex flex-col gap-3">
          {summaryLines.map((line) => (
            <li key={line.key} className="flex items-center justify-between text-sm">
              <span className="text-stone-700">
                {line.product_name}
                {line.variant_label ? ` — ${line.variant_label}` : ""}
                <span className="ml-2 text-stone-400">× {line.quantity}</span>
              </span>
              <span className="text-stone-600">{formatMoneyDual(line.line_total, storeLocale)}</span>
            </li>
          ))}
        </ul>
        <div className="mt-4 flex items-center justify-between border-t border-stone-200 pt-4">
          <Text size="sm" className="font-medium">
            {t("checkout.subtotal", "Subtotal")}
          </Text>
          <Text size="sm" className="font-medium">
            {formatMoneyDual(summarySubtotal, storeLocale)}
          </Text>
        </div>
        <div className="mt-2 flex items-center justify-between">
          <Text size="sm" tone="muted">
            {t("checkout.delivery_label", "Delivery")}
          </Text>
          <Text size="sm" tone="muted">
            {summaryDeliveryFee ? (summaryDeliveryFee.amount ? formatMoneyDual(summaryDeliveryFee, storeLocale) : t("checkout.free", "Free")) : "–"}
          </Text>
        </div>

        <div className="mt-4 border-t border-stone-200 pt-4">
          {payingByCard ? (
            // Cart is cleared during payment: the discount is locked in on the
            // pending order, so show it read-only (or nothing) rather than the
            // editable code input.
            summaryDiscount && (
              <div className="flex items-center justify-between">
                <Text size="sm" tone="muted">
                  {t("checkout.discount_label", "Discount")}{summaryDiscount.code ? ` (${summaryDiscount.code})` : ""}
                </Text>
                <Text size="sm" tone="muted">
                  −{formatMoneyDual(summaryDiscount.amount, storeLocale)}
                </Text>
              </div>
            )
          ) : appliedDiscount ? (
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Text size="sm" tone="muted">
                  {t("checkout.discount_label", "Discount")} ({appliedDiscount.code})
                </Text>
                <button
                  type="button"
                  onClick={() => { setAppliedDiscount(null); setDiscountCodeInput(""); }}
                  className="text-xs text-stone-400 hover:text-stone-700"
                >
                  ✕
                </button>
              </div>
              <Text size="sm" tone="muted">
                −{formatMoneyDual({ amount: discountAmount, currency: subtotal.currency }, storeLocale)}
              </Text>
            </div>
          ) : (
            <div className="flex gap-2">
              <Input
                placeholder={t("checkout.discount_code", "Discount code")}
                value={discountCodeInput}
                onChange={(e) => setDiscountCodeInput(e.target.value)}
                onKeyDown={(e) => { if (e.key === "Enter") void applyDiscountCode(); }}
                className="flex-1 text-sm"
              />
              <Button
                variant="outline"
                size="sm"
                onClick={() => void applyDiscountCode()}
                disabled={isValidatingCode || !discountCodeInput.trim()}
              >
                {isValidatingCode ? "…" : t("checkout.apply", "Apply")}
              </Button>
            </div>
          )}
          {discountError && !payingByCard && (
            <Text size="sm" tone="danger" className="mt-1">
              {discountError}
            </Text>
          )}
        </div>

        <div className="mt-2 flex items-center justify-between border-t border-stone-200 pt-4">
          <Text className="font-medium">{t("checkout.total", "Total")}</Text>
          <Text className="font-medium">{formatMoneyDual(summaryTotal, storeLocale)}</Text>
        </div>
        </Card>
      </div>
    </div>
  );
}

function StepIndicator({ step }: { step: Step }) {
  const { t } = useLanguage();
  const steps: { key: Step; label: string }[] = [
    { key: "details", label: t("checkout.step_details", "Details") },
    { key: "delivery", label: t("checkout.step_delivery", "Delivery") },
    { key: "payment", label: t("checkout.step_payment", "Payment") },
    { key: "confirmation", label: t("checkout.step_review", "Review") },
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

function OrderSummary({ order, totalLabel }: { order: PlacedOrder; totalLabel: string }) {
  const { storeLocale } = useStoreBranding();
  return (
    <div className="mt-6 flex flex-col gap-2 text-sm">
      {order.items.map((item, i) => (
        <div key={i} className="flex items-center justify-between">
          <span className="text-stone-700">
            {item.product_name}
            {item.variant_label ? ` — ${item.variant_label}` : ""}
            <span className="ml-2 text-stone-400">× {item.quantity}</span>
          </span>
          <span className="text-stone-600">{formatMoneyDual(item.unit_price, storeLocale)}</span>
        </div>
      ))}
      <div className="mt-2 flex items-center justify-between border-t border-stone-200 pt-2 font-medium text-stone-900">
        <span>{totalLabel}</span>
        <span>{formatMoneyDual(order.total, storeLocale)}</span>
      </div>
    </div>
  );
}

