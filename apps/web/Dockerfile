FROM node:24-alpine AS development-dependencies-env
COPY . /app
WORKDIR /app
RUN npm ci

FROM node:24-alpine AS production-dependencies-env
COPY ./package.json package-lock.json /app/
WORKDIR /app
RUN npm ci --omit=dev

FROM node:24-alpine AS build-env
COPY . /app/
COPY --from=development-dependencies-env /app/node_modules /app/node_modules
WORKDIR /app
# These VITE_* values are only the local-development fallback. Each environment
# sets its own values at runtime (API_BASE_URL, GOOGLE_CLIENT_ID, REVOLUT_ENV,
# GA_MEASUREMENT_ID, SITE_URL) so one image can be promoted from dev to prod.
# See app/lib/config/runtime.ts.
ARG VITE_API_BASE_URL=http://api.fashion-store.localhost
ARG VITE_GOOGLE_CLIENT_ID=673528779465-pajifaekv8l1odrbd8mpbglo351d7r15.apps.googleusercontent.com
# Revolut checkout widget mode: "sandbox" (default, matches local devbox + dev)
# or "prod". This is the ONLY Revolut value that ships to the browser — no
# publishable key; the per-order token carries auth.
ARG VITE_REVOLUT_ENV=sandbox
# Google Analytics 4 Measurement ID. Empty by default, so analytics stays OFF
# unless the environment sets GA_MEASUREMENT_ID.
ARG VITE_GA_MEASUREMENT_ID=
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_GOOGLE_CLIENT_ID=$VITE_GOOGLE_CLIENT_ID
ENV VITE_REVOLUT_ENV=$VITE_REVOLUT_ENV
ENV VITE_GA_MEASUREMENT_ID=$VITE_GA_MEASUREMENT_ID
RUN npm run build

FROM node:24-alpine
COPY ./package.json package-lock.json /app/
COPY --from=production-dependencies-env /app/node_modules /app/node_modules
COPY --from=build-env /app/build /app/build
WORKDIR /app
EXPOSE 3000
CMD ["npm", "run", "start"]
