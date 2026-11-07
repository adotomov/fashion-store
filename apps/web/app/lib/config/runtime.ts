/**
 * Settings that differ between environments: where the API is, the Google client, the payment mode, the
 * analytics ID and the site address. They are read when the server runs, not when the image is built, so one
 * image serves dev and prod. The server sends its values to the browser in the page (see root.tsx).
 *
 * The VITE_* values baked in at build time are only the fallback for local development.
 */

export type RuntimeConfig = {
  apiBaseUrl: string;
  googleClientId: string;
  revolutEnv: "sandbox" | "prod";
  gaMeasurementId: string;
  metaPixelId: string;
  siteUrl: string;
};

declare global {
  interface Window {
    __VERANI_CONFIG__?: Partial<RuntimeConfig>;
  }
}

const buildDefaults: RuntimeConfig = {
  apiBaseUrl: import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080",
  googleClientId: import.meta.env.VITE_GOOGLE_CLIENT_ID ?? "",
  revolutEnv: (import.meta.env.VITE_REVOLUT_ENV as "sandbox" | "prod" | undefined) ?? "sandbox",
  gaMeasurementId: (import.meta.env.VITE_GA_MEASUREMENT_ID ?? "").trim(),
  metaPixelId: (import.meta.env.VITE_META_PIXEL_ID ?? "").trim(),
  siteUrl: (import.meta.env.VITE_SITE_URL ?? "https://verani.bg").replace(/\/$/, ""),
};

/** The server's values, from its environment variables. An unset or empty variable keeps the fallback. */
function serverOverrides(): Partial<RuntimeConfig> {
  const env = typeof process === "undefined" ? undefined : process.env;
  if (!env) return {};
  return {
    apiBaseUrl: env.API_BASE_URL,
    googleClientId: env.GOOGLE_CLIENT_ID,
    revolutEnv: env.REVOLUT_ENV as RuntimeConfig["revolutEnv"] | undefined,
    gaMeasurementId: env.GA_MEASUREMENT_ID,
    metaPixelId: env.META_PIXEL_ID,
    siteUrl: env.SITE_URL,
  };
}

/** The values the server resolved, for the page to hand to the browser. */
export function resolvedRuntimeConfig(): RuntimeConfig {
  const overrides = typeof window === "undefined" ? serverOverrides() : (window.__VERANI_CONFIG__ ?? {});
  const applied = Object.fromEntries(Object.entries(overrides).filter(([, value]) => typeof value === "string" && value !== ""));
  return { ...buildDefaults, ...applied };
}

let cached: RuntimeConfig | undefined;

/** The settings in effect. Read once: the server's environment and the page's values do not change while it runs. */
export function runtimeConfig(): RuntimeConfig {
  cached ??= resolvedRuntimeConfig();
  return cached;
}
