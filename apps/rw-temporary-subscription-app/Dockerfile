FROM node:22-alpine AS deps
WORKDIR /app

COPY package.json package-lock.json ./
COPY client/package.json client/package.json
COPY server/package.json server/package.json
RUN npm ci

FROM deps AS build
WORKDIR /app

ARG VITE_API_BASE_URL
ARG VITE_TELEGRAM_BOT_USERNAME
ENV VITE_API_BASE_URL=$VITE_API_BASE_URL
ENV VITE_TELEGRAM_BOT_USERNAME=$VITE_TELEGRAM_BOT_USERNAME

COPY client ./client
COPY server ./server
RUN npm run build --workspace client
RUN npm prune --omit=dev

FROM node:22-alpine AS runtime
WORKDIR /app

ENV NODE_ENV=production

COPY package.json package-lock.json ./
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/server ./server
COPY --from=build /app/client/dist ./client/dist

EXPOSE 3001

CMD ["npm", "run", "start", "--workspace", "server"]
