export type Money = {
  amount: number; // integer minor units, e.g. cents
  currency: string;
};

// Formats integer minor units using locale-aware Intl.NumberFormat.
// Never format money with floats.
export function formatMoney(money: Money, locale: string = "en-US"): string {
  return new Intl.NumberFormat(locale, {
    style: "currency",
    currency: money.currency,
  }).format(money.amount / 100);
}
