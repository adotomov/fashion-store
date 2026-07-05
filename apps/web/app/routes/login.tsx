import { Link, Navigate, useLocation } from "react-router";

import { useAuth } from "../features/auth/AuthContext";
import { useLanguage } from "../features/i18n/LanguageContext";
import { GoogleSignInButton } from "../features/auth/GoogleSignInButton";
import { useStoreBranding } from "../features/store-settings/StoreSettingsContext";
import { Eyebrow, Heading, Text } from "../components/ui/Text";

export default function Login() {
  const { t } = useLanguage();
  const { isAuthenticated, isLoading } = useAuth();
  const { storeName, logoUrl } = useStoreBranding();
  const location = useLocation();
  const from = (location.state as { from?: { pathname: string; search: string } } | null)?.from;
  const redirectTo = from ? `${from.pathname}${from.search}` : "/account";

  if (!isLoading && isAuthenticated) {
    return <Navigate to={redirectTo} replace />;
  }

  return (
    <div className="grid min-h-screen lg:grid-cols-2">
      <div className="relative hidden lg:block">
        <img
          src="https://picsum.photos/seed/maison-login/1200/1600"
          alt=""
          className="absolute inset-0 h-full w-full object-cover"
        />
        <div className="absolute inset-0 bg-gradient-to-t from-stone-950/70 via-stone-950/10 to-stone-950/40" />
        <div className="absolute inset-x-0 bottom-0 p-12">
          <Eyebrow className="text-stone-100/80">{t("login.hero_eyebrow", "New Season")}</Eyebrow>
          <Heading as="h2" size="lg" className="mt-3 text-white">
            {t("login.hero_heading", "Crafted pieces, made to last.")}
          </Heading>
          <Text className="mt-2 max-w-sm text-stone-100/80">
            {t("login.hero_subtext", "Sign in to track orders, save favorites, and check out faster.")}
          </Text>
        </div>
      </div>

      <div className="flex flex-col justify-center px-6 py-16 sm:px-12 lg:px-20">
        <div className="mx-auto w-full max-w-sm">
          <Link to="/" className="flex items-center gap-2">
            {logoUrl && <img src={logoUrl} alt={storeName} className="h-8 w-auto object-contain" />}
            <span className="font-display text-2xl font-medium tracking-wide text-stone-900">{storeName}</span>
          </Link>

          <Heading as="h1" size="lg" className="mt-10">
            {t("login.welcome_back", "Welcome back")}
          </Heading>
          <Text tone="muted" className="mt-2">
            {t("login.subheading", "Sign in to access your account, orders, and wishlist.")}
          </Text>

          <div className="mt-8">
            <GoogleSignInButton />
          </div>

          <Text size="xs" tone="muted" className="mt-8 text-center">
            {t("login.terms_prefix", "By continuing, you agree to")} {storeName}&apos;s{" "}
            <Link to="/legal/terms" className="underline hover:text-stone-700">
              Terms of Service
            </Link>{" "}
            and{" "}
            <Link to="/legal/privacy" className="underline hover:text-stone-700">
              Privacy Policy
            </Link>
            .
          </Text>

          <Text size="sm" tone="muted" className="mt-10 text-center">
            <Link to="/shop" className="font-medium text-stone-900 underline hover:text-stone-700">
              {t("login.continue_browsing", "Continue browsing")}
            </Link>{" "}
            {t("login.without_signin", "without signing in")}
          </Text>
        </div>
      </div>
    </div>
  );
}
