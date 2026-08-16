// Phone helpers mirroring the Speedy-facing normalization on the backend
// (apps/api/.../speedy_client.go). Speedy accepts digits only with an optional
// single leading "+", must start with "0" or "+", so we validate customer input
// against the same shape before an order is placed rather than surfacing a
// carrier rejection after the fact. Bulgaria-only storefront.

// normalizePhone strips separators, maps a leading "00" international prefix to
// "+", and adds the Bulgarian national leading "0" to a bare 9-digit subscriber
// number — matching what the backend sends to Speedy.
export function normalizePhone(raw: string): string {
  const s = raw.trim();
  if (!s) return "";
  let plus = false;
  let rest = s;
  if (s.startsWith("+")) {
    plus = true;
    rest = s.slice(1);
  } else if (s.startsWith("00")) {
    plus = true;
    rest = s.slice(2);
  }
  const digits = rest.replace(/\D/g, "");
  if (!digits) return "";
  if (plus) return `+${digits}`;
  if (digits.length === 9) return `0${digits}`;
  return digits;
}

// isValidPhone reports whether raw normalizes to a shape Speedy will accept: a
// Bulgarian national number (0 + 9 digits) or an international number (+ then
// 8–14 digits).
export function isValidPhone(raw: string): boolean {
  const n = normalizePhone(raw);
  if (n.startsWith("+")) return /^\+\d{8,14}$/.test(n);
  return /^0\d{9}$/.test(n);
}
