# RW Temporary Subscription App
Due to Telegram fully blocked in Russia, need a way to get subscription
Mini full-stack app for creating a temporary RW user subscription with a fixed 100 MB traffic limit and next-day expiration.

## What is implemented

- [`server/src/index.js`](server/src/index.js) exposes [`POST /api/subscriptions/temporary`](server/src/index.js:40)
- [`client/src/App.jsx`](client/src/App.jsx) creates only one temporary subscription per browser, persists it across page refresh, copies `subscriptionUrl`, and shows activation steps with VPN client links
- The backend proxies the request to the RW API and keeps the RW token out of the browser
- The RW request mapping now follows the schema in [`api-1.json`](api-1.json:25012) for [`CreateUserRequestDto`](api-1.json:25012)

## Project structure

- [`package.json`](package.json) - root workspace config
- [`server/package.json`](server/package.json) - Express backend package
- [`server/src/index.js`](server/src/index.js) - backend API proxy
- [`client/package.json`](client/package.json) - React/Vite frontend package
- [`client/src/App.jsx`](client/src/App.jsx) - main UI
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
- `APP_AUTH_PASSWORD` - password-only protection for page/API access; if empty, auth is disabled
- `APP_AUTH_COOKIE_SAME_SITE` - cookie SameSite policy for auth; use `None` behind cross-site proxies/iframes, which also enables `Secure`
- `VITE_API_BASE_URL` - optional frontend API base URL override; if empty in production, frontend uses same origin (`/api/...`)
- `VITE_TELEGRAM_BOT_USERNAME` - Telegram bot username (without `@`) shown in the activation instructions

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

Then open:

```text
http://localhost:3001
```

## RW API mapping

The backend sends a request to [`/api/users`](api-1.json:2570) using the documented [`CreateUserRequestDto`](api-1.json:25012) shape built in [`buildRwPayload()`](server/src/index.js:17):

- `username`
- `description`
- `status`
- `trafficLimitBytes`
- `trafficLimitStrategy`
- `expireAt`
- `activeInternalSquads` (when `RW_SQUAD_UUID` is set)

For the requested behavior, the app sends:

- `trafficLimitBytes = 104857600` for 100 MB
- `trafficLimitStrategy = NO_RESET`
- `expireAt = now + 1 day`

## Expected user flow

1. Open the React app
2. Click create temporary subscription
3. Backend generates a username and creates a request with 100 MB traffic and next-day expiration
4. The page stores the successful subscription in browser storage and prevents creating another one in the same browser
5. After page refresh, the saved subscription is displayed again
6. The page shows `subscriptionUrl`, a copy-to-clipboard button, instructions to open the Telegram bot, and quick links to VPN clients

## Notes

- Secrets stay on the backend and are not exposed to the browser
- The app is intentionally minimal and focused on one creation flow only
- In container runtime, the backend serves both `/api/*` and built frontend from `client/dist`
- When `APP_AUTH_PASSWORD` is set, entry is protected by a password page (`/auth/login`) and a server cookie (except `/api/health`)
- Before production use, add authentication/rate limiting if this page will be publicly reachable
