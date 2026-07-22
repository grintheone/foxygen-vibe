FROM node:22-alpine AS builder
WORKDIR /app

ARG NPM_REGISTRY=https://registry.npmmirror.com

COPY client/package.json client/package-lock.json ./
RUN npm ci \
    --registry="${NPM_REGISTRY}" \
    --fetch-retries=5 \
    --fetch-retry-mintimeout=20000 \
    --fetch-retry-maxtimeout=120000 \
    --fetch-timeout=120000

COPY client/. .
RUN npm run build

FROM nginx:1.27-alpine
WORKDIR /usr/share/nginx/html

COPY client/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist ./

EXPOSE 80
