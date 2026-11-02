import { runtimeConfig } from "../config/runtime";

// Central place to read the Google Analytics configuration. The Measurement ID
// is set per environment with GA_MEASUREMENT_ID on the server (see
// lib/config/runtime.ts). When it is empty — local dev, devbox, or any
// environment that hasn't set it — analytics is OFF: the gtag script is never
// injected and every tracking helper becomes a no-op.
//
// A valid GA4 Measurement ID looks like "G-XXXXXXXXXX"; we sanity-check the
// prefix so a mis-set value (e.g. a stray quote or a UA-… id) doesn't inject a
// broken tag.
export const GA_MEASUREMENT_ID = runtimeConfig().gaMeasurementId;

export const isAnalyticsEnabled = GA_MEASUREMENT_ID.startsWith("G-");
