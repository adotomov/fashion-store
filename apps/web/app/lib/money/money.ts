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

// Fixed BGN peg to the euro (Bulgaria's ERM II central rate). Bulgarian law
// requires showing prices in both EUR and BGN, converted at this fixed rate.
export const EUR_TO_BGN_RATE = 1.95583;

// Converts an EUR amount (integer minor units) to BGN, rounding to the nearest
// stotinka. Returns the input unchanged if it isn't in EUR.
export function toBGN(money: Money): Money {
  if (money.currency !== "EUR") return money;
  return { amount: Math.round(money.amount * EUR_TO_BGN_RATE), currency: "BGN" };
}

// True when the store locale designates Bulgaria, where prices must be shown in
// both EUR and BGN. Accepts both the short "bg" and BCP-47 "bg-BG" forms.
export function isBulgarianLocale(locale: string): boolean {
  return locale.toLowerCase().startsWith("bg");
}

// Formats an EUR price with its BGN equivalent appended inline (e.g.
// "€10.00 / 19,56 лв."), for string contexts where a stacked <Price> tag
// doesn't fit. Returns the plain EUR string unless the locale is Bulgarian
// and the price is EUR-denominated.
export function formatMoneyDual(money: Money, locale: string): string {
  const eur = formatMoney(money);
  if (!isBulgarianLocale(locale) || money.currency !== "EUR") return eur;
  return `${eur} / ${formatMoney(toBGN(money), "bg-BG")}`;
}
