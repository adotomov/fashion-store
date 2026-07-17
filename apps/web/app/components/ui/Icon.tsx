import {
  Bell,
  Boxes,
  Check,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronsUpDown,
  CreditCard,
  Globe,
  Heart,
  LayoutDashboard,
  Lock,
  LogOut,
  Mail,
  MapPin,
  Menu,
  Minus,
  Package,
  Pencil,
  type LucideProps,
  Phone,
  Plus,
  Receipt,
  Search,
  Settings,
  ShieldCheck,
  ShoppingBag,
  SlidersHorizontal,
  Star,
  Tag,
  Trash2,
  Truck,
  User,
  Users,
  X,
} from "lucide-react";

// lucide-react v1 dropped brand/logo glyphs (trademark reasons), so the two
// social icons used in the footer are minimal hand-rolled SVGs kept to the
// same LucideProps shape for a consistent call site.
function Instagram({ size = 24, strokeWidth = 1.75, color = "currentColor", ...props }: LucideProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <rect x="3" y="3" width="18" height="18" rx="5" />
      <circle cx="12" cy="12" r="4" />
      <circle cx="17.5" cy="6.5" r="0.5" fill={color} stroke="none" />
    </svg>
  );
}

function Facebook({ size = 24, strokeWidth = 1.75, color = "currentColor", ...props }: LucideProps) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke={color}
      strokeWidth={strokeWidth}
      strokeLinecap="round"
      strokeLinejoin="round"
      {...props}
    >
      <path d="M15 4h-2a4 4 0 0 0-4 4v3H7v3h2v6h3v-6h2.5l.5-3H12V8a1 1 0 0 1 1-1h2z" />
    </svg>
  );
}

// Centralized icon registry: pages reference icons by semantic name
// (`<Icon name="cart" />`) instead of importing from lucide-react directly,
// so the underlying icon set can be swapped without touching call sites.
const icons = {
  cart: ShoppingBag,
  wishlist: Heart,
  profile: User,
  search: Search,
  menu: Menu,
  close: X,
  chevronDown: ChevronDown,
  chevronLeft: ChevronLeft,
  chevronRight: ChevronRight,
  star: Star,
  minus: Minus,
  plus: Plus,
  trash: Trash2,
  pencil: Pencil,
  check: Check,
  filters: SlidersHorizontal,
  shipping: Truck,
  facebook: Facebook,
  instagram: Instagram,
  mail: Mail,
  phone: Phone,
  mapPin: MapPin,
  dashboard: LayoutDashboard,
  settings: Settings,
  users: Users,
  catalog: Boxes,
  inventory: Package,
  invoices: Receipt,
  payment: CreditCard,
  logout: LogOut,
  bell: Bell,
  chevronsUpDown: ChevronsUpDown,
  globe: Globe,
  tag: Tag,
  lock: Lock,
  shieldCheck: ShieldCheck,
} as const;

export type IconName = keyof typeof icons;

type IconProps = Omit<LucideProps, "ref"> & {
  name: IconName;
};

export function Icon({ name, size = 20, strokeWidth = 1.75, ...props }: IconProps) {
  const Component = icons[name];
  return <Component size={size} strokeWidth={strokeWidth} {...props} />;
}
