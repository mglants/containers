# RW Temporary Subscription App
Due to Telegram fully blocked in Russia, need a way to get subscription
Mini full-stack app for creating a temporary RW user subscription with a configurable traffic limit in MB and next-day expiration, then handing it off to Telegram for permanent registration.

## What is implemented

- [`server/src/index.js`](server/src/index.js) bootstraps the backend and starts the built-in Telegram handoff bot
- [`server/src/app.js`](server/src/app.js) exposes `POST /api/subscriptions/temporary` plus auth, health, and runtime config routes
- [`client/src/App.jsx`](client/src/App.jsx) creates only one temporary subscription per browser, persists it across page refresh, copies `subscriptionUrl`, and shows the Telegram handoff flow
- The backend proxies the request to the RW API and keeps the RW token out of the browser
- The RW request mapping follows the schema in [`api.json`](api.json:25012) for `CreateUserRequestDto` and `UpdateUserRequestDto`

## Project structure

- [`package.json`](package.json) - root workspace config
- [`server/package.json`](server/package.json) - Express backend package
- [`server/src/index.js`](server/src/index.js) - startup wiring
- [`server/src/app.js`](server/src/app.js) - Express app and HTTP routes
- [`server/src/auth.js`](server/src/auth.js) - password-only page auth
- [`server/src/config.js`](server/src/config.js) - environment parsing and derived config
- [`server/src/remnawave-client.js`](server/src/remnawave-client.js) - RW API client
- [`server/src/subscription-service.js`](server/src/subscription-service.js) - temporary subscription workflow
- [`server/src/telegram-bot.js`](server/src/telegram-bot.js) - built-in Telegram long-poll bot
- [`server/src/temp-subscription-store.js`](server/src/temp-subscription-store.js) - persisted temp ID store
- [`client/package.json`](client/package.json) - React/Vite frontend package
- [`client/src/App.jsx`](client/src/App.jsx) - client state and orchestration
- [`client/src/components/`](client/src/components) - presentational React components
- [`client/src/config.js`](client/src/config.js) - runtime client config helpers
- [`client/src/texts.js`](client/src/texts.js) - localized copy and date helpers
- [`client/src/subscription-storage.js`](client/src/subscription-storage.js) - browser persistence for the created subscription
- [`client/src/styles.css`](client/src/styles.css) - page styling
- [`Dockerfile`](Dockerfile) - production image (frontend build + backend runtime)
- [`docker-compose.yml`](docker-compose.yml) - local Docker orchestration
- [`.env.example`](.env.example) - environment template

## Environment setup

Copy [`.env.example`](.env.example) to [`.env`](.env) and configure the values:

```bash
cp .env.example .env
```

Variables:

- `PORT` - backend server port
- `RW_API_BASE_URL` - RW API base URL
- `RW_API_TOKEN` - RW bearer token for backend proxy requests
- `RW_SQUAD_UUID` - optional internal squad UUID; when set, each created user is added to this squad
- `RW_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB` - traffic limit in megabytes for newly created temporary users; defaults to `100` when omitted
- `APP_AUTH_PASSWORD` - password-only protection for page/API access; if empty, auth is disabled
- `APP_AUTH_COOKIE_SAME_SITE` - cookie SameSite policy for auth; use `None` behind cross-site proxies/iframes, which also enables `Secure`
- `TELEGRAM_BOT_USERNAME` - Telegram handoff bot username without `@`; backend uses it to build `https://t.me/<bot>?start=<temp_id>`
- `TELEGRAM_BOT_TOKEN` - Telegram Bot API token for the built-in handoff bot poller
- `TELEGRAM_BOT_POLL_TIMEOUT_SECONDS` - long-poll timeout for Telegram `getUpdates`; defaults to `30`
- `REGISTRATION_BOT_URL` - full URL of the next bot that should receive the user after the handoff step
- `TEMP_SUBSCRIPTION_LINKS_PATH` - JSON file used to persist temporary handoff records between the page and Telegram bot
- `VITE_API_BASE_URL` - optional frontend API base URL override; if empty in production, frontend uses same origin (`/api/...`)
- `VITE_TELEGRAM_BOT_USERNAME` - optional frontend fallback for the Telegram handoff bot username when you are not serving runtime config from the backend

## Install dependencies

```bash
npm install
```

## NixOS setup

This workspace includes [`shell.nix`](shell.nix) for a reproducible development shell with [`nodejs_22`](shell.nix:5).

Enter the shell:

```bash
nix-shell
```

Install dependencies and build inside the shell:

```bash
npm install
npm run build
```

Or run both in one command:

```bash
nix-shell --run 'npm install && npm run build'
```

## Run locally

Start backend and frontend together:

```bash
npm run dev
```

Or run each side separately:

```bash
npm run dev:server
npm run dev:client
```

## Docker

Build production image:

```bash
npm run docker:build
```

Note: `VITE_*` variables are injected at build time. Rebuild the image after changing them.

Run container (uses values from `.env`):

```bash
npm run docker:run
```

Or use Compose:

```bash
docker compose up --build
```

Compose passes `VITE_API_BASE_URL` and `VITE_TELEGRAM_BOT_USERNAME` as Docker build args and uses `.env` as runtime env for backend variables.
Both `npm run docker:run` and Compose persist temporary handoff records in the named Docker volume `rw-temp-subscription-data`.
The image also includes a container healthcheck against `/api/health`.

Then open:

```text
http://localhost:3001
```

## RW API mapping

The backend sends a request to [`/api/users`](api.json:2570) using the documented `CreateUserRequestDto` shape built in [`buildTemporaryUserPayload()`](server/src/remnawave-client.js):

- `username`
- `description`
- `status`
- `trafficLimitBytes`
- `trafficLimitStrategy`
- `expireAt`
- `activeInternalSquads` (when `RW_SQUAD_UUID` is set)

For the requested behavior, the app sends:

- `trafficLimitBytes = RW_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB * 1048576`
- `trafficLimitStrategy = NO_RESET`
- `expireAt = now + 1 day`

The app keeps the configured limit in megabytes internally and converts it to `trafficLimitBytes` only when building the Remnawave API request.

When the built-in Telegram bot handles `/start <tempSubscriptionId>`, the backend updates the same RW user via `PATCH /api/users` using:

- `uuid`
- `telegramId = <telegram_id>`

## Telegram bot flow

The built-in Telegram bot finishes the handoff itself when `TELEGRAM_BOT_TOKEN` is set. It polls Telegram with `getUpdates`, reads `/start <tempSubscriptionId>`, links the RW user internally, and replies to the user with `REGISTRATION_BOT_URL`.
Telegram replies use the Telegram account language when available, and fall back to the frontend locale saved with the temporary subscription.

## Expected user flow

1. Open the React app
2. Click create temporary subscription
3. Backend generates a temporary RW username, creates a request with the configured traffic limit and next-day expiration, and returns a Telegram deep link with `start=<tempSubscriptionId>`
4. The page stores the successful subscription in browser storage, forwards the selected page locale to the backend, and prevents creating another one in the same browser
5. User opens the personal Telegram deep link
6. The built-in bot reads the Telegram ID from `/start`, and the backend links that Telegram ID to the temporary RW user
7. The built-in bot sends `registrationBotUrl` back to the user in Telegram
8. After page refresh, the saved subscription is displayed again with the same temporary subscription data

If Remnawave already has a different user with the same Telegram ID, the handoff is rejected instead of reassigning that Telegram ID to the new temporary user.

## Notes

- Secrets stay on the backend and are not exposed to the browser
- Temporary Telegram handoff records are persisted in `server/data/temporary-subscriptions.json` by default
- The built-in bot uses long polling, so do not run a webhook for the same bot token at the same time
- The app is intentionally minimal and focused on one creation flow only
- In container runtime, the backend serves both `/api/*` and built frontend from `client/dist`
- When `APP_AUTH_PASSWORD` is set, entry is protected by a password page (`/auth/login`) and a server cookie (except `/api/health`)
- Before production use, add authentication/rate limiting if this page will be publicly reachable
