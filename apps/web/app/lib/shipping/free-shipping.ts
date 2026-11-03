import { formatMoney } from "../money/money";

// Free shipping, as the checkout applies it. The server decides the fee an order pays; these helpers show the same
// rule before the order is placed. An order whose subtotal (after discounts, before delivery) reaches the store's
// threshold ships free. The threshold is in EUR minor units and is null until the store settings have loaded.

export function qualifiesForFreeShipping(subtotalMinor: number, thresholdMinor: number | null): boolean {
  return thresholdMinor !== null && subtotalMinor >= thresholdMinor;
}

// The fee an order pays for a delivery method: nothing when it qualifies, otherwise the method's own fee.
export function feeFor(fee: { amount: number; currency: string }, subtotalMinor: number, thresholdMinor: number | null) {
  return qualifiesForFreeShipping(subtotalMinor, thresholdMinor) ? { amount: 0, currency: fee.currency } : fee;
}

// Fills the {amount} placeholder of a shipping message with the threshold, formatted for the store's locale.
// Text without a placeholder, or a store with no threshold loaded, comes back unchanged.
export function withThreshold(text: string, thresholdMinor: number | null, locale: string): string {
  if (thresholdMinor === null || !text.includes("{amount}")) return text;
  return text.replace("{amount}", formatMoney({ amount: thresholdMinor, currency: "EUR" }, locale || "en-US"));
}
