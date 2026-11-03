# syntax=docker/dockerfile:1

# ---- build ---------------------------------------------------------------
FROM node:26-alpine AS build
WORKDIR /app

# Install dependencies from the lockfile before copying sources for better caching.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm npm ci --no-audit --no-fund

COPY . .
RUN npm run build

# ---- runtime -------------------------------------------------------------
FROM nginx:stable-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
