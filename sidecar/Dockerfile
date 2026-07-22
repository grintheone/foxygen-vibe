FROM node:22-alpine

WORKDIR /app

COPY sidecar/package.json sidecar/package-lock.json ./
RUN npm ci --omit=dev

COPY sidecar/index.js ./

EXPOSE 5103

CMD ["npm", "start"]
