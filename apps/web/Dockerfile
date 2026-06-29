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
# Vite inlines VITE_* env vars at build time, so they must be baked in here
# rather than passed at container runtime. devbox-cli's build step doesn't
# support custom --build-arg, so these are fixed for the devbox deployment.
ENV VITE_API_BASE_URL=http://api.api.localhost
ENV VITE_GOOGLE_CLIENT_ID=673528779465-pajifaekv8l1odrbd8mpbglo351d7r15.apps.googleusercontent.com
RUN npm run build

FROM node:24-alpine
COPY ./package.json package-lock.json /app/
COPY --from=production-dependencies-env /app/node_modules /app/node_modules
COPY --from=build-env /app/build /app/build
WORKDIR /app
EXPOSE 3000
CMD ["npm", "run", "start"]
