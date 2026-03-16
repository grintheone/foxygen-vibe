FROM node:22-alpine AS builder
WORKDIR /app

COPY client/package.json client/package-lock.json ./
RUN npm ci

COPY client/. .
RUN npm run build

FROM nginx:1.27-alpine
WORKDIR /usr/share/nginx/html

COPY client/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/dist ./

EXPOSE 80
