import {
  isRouteErrorResponse,
  Links,
  Meta,
  Outlet,
  Scripts,
  ScrollRestoration,
  useRouteLoaderData,
} from "react-router";

import type { Route } from "./+types/root";
import "./app.css";
import { GoogleAnalytics } from "./features/analytics/GoogleAnalytics";
import { MetaPixel } from "./features/analytics/MetaPixel";
import { usePageViews } from "./features/analytics/usePageViews";
import { AuthProvider } from "./features/auth/AuthContext";
import { CartProvider } from "./features/cart/CartContext";
import { ConsentBanner } from "./features/consent/ConsentBanner";
import { ConsentProvider } from "./features/consent/ConsentContext";
import { LanguageProvider } from "./features/i18n/LanguageContext";
import { StoreSettingsProvider } from "./features/store-settings/StoreSettingsContext";
import { WishlistProvider } from "./features/wishlist/WishlistContext";
import { resolvedRuntimeConfig } from "./lib/config/runtime";

/** Runs on the server for every page: the settings the server resolved, which the page hands to the browser. */
export function loader() {
  return { config: resolvedRuntimeConfig() };
}

export const links: Route.LinksFunction = () => [
  { rel: "preconnect", href: "https://fonts.googleapis.com" },
  {
    rel: "preconnect",
    href: "https://fonts.gstatic.com",
    crossOrigin: "anonymous",
  },
  {
    rel: "stylesheet",
    href: "https://fonts.googleapis.com/css2?family=Inter:ital,opsz,wght@0,14..32,100..900;1,14..32,100..900&family=Cormorant+Garamond:ital,wght@0,400;0,500;0,600;1,400&display=swap",
  },
];

export function Layout({ children }: { children: React.ReactNode }) {
  const root = useRouteLoaderData<typeof loader>("root");
  // Escaping "<" keeps the JSON from closing the script early.
  const configScript = root ? `window.__VERANI_CONFIG__=${JSON.stringify(root.config).replace(/</g, "\\u003c")};` : undefined;
  return (
    <html lang="en">
      <head>
        <meta charSet="utf-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1" />
        {configScript ? <script dangerouslySetInnerHTML={{ __html: configScript }} /> : null}
        <Meta />
        <Links />
        <GoogleAnalytics />
        <MetaPixel />
      </head>
      <body>
        {children}
        <ScrollRestoration />
        <Scripts />
      </body>
    </html>
  );
}

export default function App() {
  usePageViews();
  return (
    <LanguageProvider>
      <ConsentProvider>
        <StoreSettingsProvider>
          <AuthProvider>
            <CartProvider>
              <WishlistProvider>
                <Outlet />
              </WishlistProvider>
            </CartProvider>
          </AuthProvider>
        </StoreSettingsProvider>
        <ConsentBanner />
      </ConsentProvider>
    </LanguageProvider>
  );
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
  let message = "Oops!";
  let details = "An unexpected error occurred.";
  let stack: string | undefined;

  if (isRouteErrorResponse(error)) {
    message = error.status === 404 ? "404" : "Error";
    details =
      error.status === 404
        ? "The requested page could not be found."
        : error.statusText || details;
  } else if (import.meta.env.DEV && error && error instanceof Error) {
    details = error.message;
    stack = error.stack;
  }

  return (
    <main className="pt-16 p-4 container mx-auto">
      <h1>{message}</h1>
      <p>{details}</p>
      {stack && (
        <pre className="w-full p-4 overflow-x-auto">
          <code>{stack}</code>
        </pre>
      )}
    </main>
  );
}
