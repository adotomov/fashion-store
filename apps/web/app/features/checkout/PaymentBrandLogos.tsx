// Inline SVG marks for the payment brands we accept, rendered inside small
// white chips so they read like the standard "accepted cards" row on a hosted
// checkout page. Kept as self-contained SVGs (no external image requests, no
// bundler asset wiring) so they render instantly and theme cleanly.

type LogoProps = { className?: string };

function Chip({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <span
      role="img"
      aria-label={label}
      title={label}
      className="inline-flex h-7 w-11 items-center justify-center rounded-sm border border-stone-200 bg-white shadow-sm"
    >
      {children}
    </span>
  );
}

function VisaLogo({ className }: LogoProps) {
  return (
    <svg viewBox="0 0 48 16" className={className} aria-hidden="true">
      <text
        x="24"
        y="13"
        textAnchor="middle"
        fontFamily="Georgia, 'Times New Roman', serif"
        fontSize="14"
        fontStyle="italic"
        fontWeight="700"
        letterSpacing="0.5"
        fill="#1A1F71"
      >
        VISA
      </text>
    </svg>
  );
}

function MastercardLogo({ className }: LogoProps) {
  return (
    <svg viewBox="0 0 40 24" className={className} aria-hidden="true">
      <circle cx="15" cy="12" r="9" fill="#EB001B" />
      <circle cx="25" cy="12" r="9" fill="#F79E1B" />
      <path d="M20 5.2a9 9 0 0 0 0 13.6 9 9 0 0 0 0-13.6Z" fill="#FF5F00" />
    </svg>
  );
}

function ApplePayLogo({ className }: LogoProps) {
  return (
    <svg viewBox="0 0 48 20" className={className} aria-hidden="true" fill="#000">
      {/* Apple mark */}
      <path d="M9.9 6.1c-.5.6-1.3 1-2 .95-.1-.8.28-1.65.73-2.17.5-.6 1.36-1.03 2.05-1.06.08.84-.25 1.66-.78 2.28Zm.77 1.22c-1.13-.07-2.1.64-2.64.64-.55 0-1.38-.6-2.28-.59-1.17.02-2.26.68-2.86 1.74-1.22 2.11-.32 5.24.87 6.96.58.85 1.28 1.8 2.2 1.76.87-.03 1.2-.57 2.26-.57 1.05 0 1.35.57 2.27.55.94-.02 1.53-.86 2.11-1.71.66-.98.93-1.93.95-1.98-.02-.01-1.82-.7-1.84-2.78-.02-1.74 1.42-2.57 1.49-2.62-.82-1.2-2.08-1.34-2.53-1.37Z" />
      {/* "Pay" wordmark */}
      <text x="19" y="15" fontFamily="'Helvetica Neue', Arial, sans-serif" fontSize="12" fontWeight="600">
        Pay
      </text>
    </svg>
  );
}

function GooglePayLogo({ className }: LogoProps) {
  // The Google "G": a ring drawn as four coloured arcs with a blue bar into the
  // centre. Built from stroked circle segments (via stroke-dasharray) so it
  // stays crisp at any size without a fragile hand-tuned path.
  const c = 10;
  const r = 6;
  const sw = 3;
  const circ = 2 * Math.PI * r;
  const quarter = circ / 4;
  return (
    <svg viewBox="0 0 52 20" className={className} aria-hidden="true">
      <g fill="none" strokeWidth={sw}>
        {/* blue (right), green (bottom), yellow (left), red (top) quadrants */}
        <circle cx={c} cy={c} r={r} stroke="#4285F4" strokeDasharray={`${quarter} ${circ - quarter}`} transform={`rotate(-45 ${c} ${c})`} />
        <circle cx={c} cy={c} r={r} stroke="#34A853" strokeDasharray={`${quarter} ${circ - quarter}`} transform={`rotate(45 ${c} ${c})`} />
        <circle cx={c} cy={c} r={r} stroke="#FBBC04" strokeDasharray={`${quarter} ${circ - quarter}`} transform={`rotate(135 ${c} ${c})`} />
        <circle cx={c} cy={c} r={r} stroke="#EA4335" strokeDasharray={`${quarter} ${circ - quarter}`} transform={`rotate(-135 ${c} ${c})`} />
      </g>
      {/* blue bar into the centre */}
      <rect x={c} y={c - sw / 2} width={r + 1} height={sw} fill="#4285F4" />
      {/* mask the bar's right tip to keep the G opening */}
      <rect x={c + r - 1} y={c - sw / 2} width="2" height={sw} fill="#fff" />
      {/* "Pay" wordmark */}
      <text x="20" y="15" fontFamily="'Helvetica Neue', Arial, sans-serif" fontSize="12" fontWeight="500" fill="#5F6368">
        Pay
      </text>
    </svg>
  );
}

// Renders the row of accepted-brand chips. Purely decorative; the surrounding
// copy carries the accessible meaning.
export function PaymentBrandLogos({ className }: LogoProps) {
  return (
    <div className={className}>
      <Chip label="Visa">
        <VisaLogo className="h-3.5 w-9" />
      </Chip>
      <Chip label="Mastercard">
        <MastercardLogo className="h-5 w-8" />
      </Chip>
      <Chip label="Apple Pay">
        <ApplePayLogo className="h-4 w-9" />
      </Chip>
      <Chip label="Google Pay">
        <GooglePayLogo className="h-4 w-10" />
      </Chip>
    </div>
  );
}
