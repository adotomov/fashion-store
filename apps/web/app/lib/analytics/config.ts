// Central place to read the Google Analytics configuration. The Measurement ID
// is inlined by Vite at build time from VITE_GA_MEASUREMENT_ID (baked into the
// image via --build-arg in the deploy workflows). When it is empty — local dev,
// devbox, or any environment that hasn't set it — analytics is OFF: the gtag
// script is never injected and every tracking helper becomes a no-op.
//
// A valid GA4 Measurement ID looks like "G-XXXXXXXXXX"; we sanity-check the
// prefix so a mis-set value (e.g. a stray quote or a UA-… id) doesn't inject a
// broken tag.
export const GA_MEASUREMENT_ID = (import.meta.env.VITE_GA_MEASUREMENT_ID ?? "").trim();

export const isAnalyticsEnabled = GA_MEASUREMENT_ID.startsWith("G-");
