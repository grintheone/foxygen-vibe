FROM node:22-alpine

WORKDIR /app

ARG NPM_REGISTRY=https://registry.npmmirror.com

COPY sidecar/package.json sidecar/package-lock.json ./
RUN npm ci --omit=dev \
    --registry="${NPM_REGISTRY}" \
    --fetch-retries=5 \
    --fetch-retry-mintimeout=20000 \
    --fetch-retry-maxtimeout=120000 \
    --fetch-timeout=120000

COPY sidecar/index.js sidecar/latest-ticket.js ./

EXPOSE 5103

CMD ["npm", "start"]
