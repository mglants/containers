import 'dotenv/config';
import crypto from 'crypto';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import express from 'express';
import cors from 'cors';

const app = express();
const PORT = process.env.PORT || 3001;
const RW_API_BASE_URL = process.env.RW_API_BASE_URL;
const RW_API_TOKEN = process.env.RW_API_TOKEN;
const RW_SQUAD_UUID = process.env.RW_SQUAD_UUID?.trim();
const APP_AUTH_PASSWORD = process.env.APP_AUTH_PASSWORD || '';
const APP_AUTH_ENABLED = Boolean(APP_AUTH_PASSWORD);
const APP_AUTH_COOKIE_NAME = 'rw_app_auth';
const APP_AUTH_COOKIE_MAX_AGE_SECONDS = 60 * 60 * 24 * 30;
const APP_AUTH_COOKIE_SAME_SITE =
  process.env.APP_AUTH_COOKIE_SAME_SITE?.trim() || 'Lax';
const APP_AUTH_COOKIE_SECURE = APP_AUTH_COOKIE_SAME_SITE.toLowerCase() === 'none';
const APP_AUTH_COOKIE_VALUE = APP_AUTH_ENABLED
  ? crypto.createHash('sha256').update(APP_AUTH_PASSWORD).digest('hex')
  : '';
const DEFAULT_TRAFFIC_BYTES = 100 * 1024 * 1024;
const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const CLIENT_DIST_PATH = path.resolve(__dirname, '../../client/dist');
const CLIENT_INDEX_PATH = path.join(CLIENT_DIST_PATH, 'index.html');

function timingSafeEqual(a, b) {
  const left = Buffer.from(a, 'utf8');
  const right = Buffer.from(b, 'utf8');

  if (left.length !== right.length) {
    return false;
  }

  return crypto.timingSafeEqual(left, right);
}

function parseCookies(cookieHeader) {
  if (!cookieHeader) {
    return {};
  }

  return cookieHeader.split(';').reduce((acc, part) => {
    const separatorIndex = part.indexOf('=');
    if (separatorIndex < 0) {
      return acc;
    }

    const key = part.slice(0, separatorIndex).trim();
    const value = decodeURIComponent(part.slice(separatorIndex + 1).trim());
    acc[key] = value;
    return acc;
  }, {});
}

function hasValidAuthCookie(req) {
  const cookies = parseCookies(req.headers.cookie);
  const cookieValue = cookies[APP_AUTH_COOKIE_NAME];

  if (!cookieValue || !APP_AUTH_COOKIE_VALUE) {
    return false;
  }

  return timingSafeEqual(cookieValue, APP_AUTH_COOKIE_VALUE);
}

function renderAuthPage(errorMessage = '') {
  return `<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Enter Password</title>
    <style>
      body { margin: 0; font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; background: #0f172a; color: #e2e8f0; }
      .wrap { min-height: 100vh; display: grid; place-items: center; padding: 16px; }
      .card { width: 100%; max-width: 420px; background: rgba(15,23,42,.9); border: 1px solid rgba(148,163,184,.25); border-radius: 16px; padding: 24px; }
      h1 { margin: 0 0 12px; font-size: 24px; }
      p { margin: 0 0 16px; color: #cbd5e1; }
      label { display: grid; gap: 8px; margin-bottom: 16px; }
      input { width: 100%; box-sizing: border-box; padding: 12px 14px; border-radius: 12px; border: 1px solid #334155; background: #020617; color: #e2e8f0; }
      button { width: 100%; border: none; border-radius: 12px; background: linear-gradient(135deg, #38bdf8 0%, #2563eb 100%); color: #fff; padding: 12px 16px; font-weight: 700; cursor: pointer; }
      .error { margin: 0 0 12px; color: #fca5a5; }
    </style>
  </head>
  <body>
    <div class="wrap">
      <form class="card" method="post" action="/auth/login">
        <h1>Enter Password</h1>
        <p>Access to this page is protected by a password.</p>
        ${errorMessage ? `<p class="error">${errorMessage}</p>` : ''}
        <label>
          <span>Password</span>
          <input type="password" name="password" required autocomplete="current-password" />
        </label>
        <button type="submit">Continue</button>
      </form>
    </div>
  </body>
</html>`;
}

function buildAuthCookie() {
  const parts = [
    `${APP_AUTH_COOKIE_NAME}=${APP_AUTH_COOKIE_VALUE}`,
    `Max-Age=${APP_AUTH_COOKIE_MAX_AGE_SECONDS}`,
    'Path=/',
    'HttpOnly',
    `SameSite=${APP_AUTH_COOKIE_SAME_SITE}`
  ];

  if (APP_AUTH_COOKIE_SECURE) {
    parts.push('Secure');
  }

  return parts.join('; ');
}

function requirePageAuth(req, res, next) {
  if (
    !APP_AUTH_ENABLED ||
    req.path === '/api/health' ||
    req.method === 'OPTIONS' ||
    req.path.startsWith('/auth/')
  ) {
    return next();
  }

  if (hasValidAuthCookie(req)) {
    return next();
  }

  if (req.path.startsWith('/api/')) {
    return res.status(401).json({ error: 'Authentication required' });
  }

  return res.redirect('/auth/login');
}

app.use(
  cors({
    origin: true,
    credentials: true
  })
);
app.use(requirePageAuth);
app.use(express.json());
app.use(express.urlencoded({ extended: false }));

app.get('/auth/login', (req, res) => {
  if (!APP_AUTH_ENABLED) {
    return res.redirect('/');
  }

  if (hasValidAuthCookie(req)) {
    return res.redirect('/');
  }

  return res.status(200).type('html').send(renderAuthPage());
});

app.get('/auth/status', (req, res) => {
  if (!APP_AUTH_ENABLED) {
    return res.json({ authenticated: true, authEnabled: false });
  }

  return res.json({
    authenticated: hasValidAuthCookie(req),
    authEnabled: true
  });
});

app.post('/auth/session', (req, res) => {
  if (!APP_AUTH_ENABLED) {
    return res.status(200).json({ authenticated: true });
  }

  const password = String(req.body?.password ?? '');

  if (!timingSafeEqual(password, APP_AUTH_PASSWORD)) {
    return res.status(401).json({ error: 'Invalid password' });
  }

  res.setHeader(
    'Set-Cookie',
    buildAuthCookie()
  );
  return res.status(200).json({ authenticated: true });
});

app.post('/auth/login', (req, res) => {
  if (!APP_AUTH_ENABLED) {
    return res.redirect('/');
  }

  const password = String(req.body?.password ?? '');
  if (timingSafeEqual(password, APP_AUTH_PASSWORD)) {
    res.setHeader(
      'Set-Cookie',
      buildAuthCookie()
    );
    return res.redirect('/');
  }

  return res.status(401).type('html').send(renderAuthPage('Invalid password.'));
});

app.get('/api/health', (_req, res) => {
  res.json({ status: 'ok' });
});

function getTomorrowIso() {
  const date = new Date();
  date.setDate(date.getDate() + 1);
  return date.toISOString();
}

function generateUsername() {
  const timestamp = Date.now().toString(36);
  const randomSuffix = Math.random().toString(36).slice(2, 8);
  return `temp-${timestamp}-${randomSuffix}`;
}

function buildRwPayload({ username }) {
  const expireAt = getTomorrowIso();
  const payload = {
    username,
    description: 'Auto-generated temporary subscription',
    status: 'ACTIVE',
    trafficLimitBytes: DEFAULT_TRAFFIC_BYTES,
    trafficLimitStrategy: 'NO_RESET',
    expireAt
  };

  if (RW_SQUAD_UUID) {
    payload.activeInternalSquads = [RW_SQUAD_UUID];
  }

  return payload;
}

app.post('/api/subscriptions/temporary', async (req, res) => {
  if (!RW_API_BASE_URL || !RW_API_TOKEN) {
    return res.status(500).json({
      error: 'Missing RW API configuration',
      details: 'Set RW_API_BASE_URL and RW_API_TOKEN in your environment.'
    });
  }

  const username = generateUsername();
  const payload = buildRwPayload({ username });

  try {
    const response = await fetch(`${RW_API_BASE_URL.replace(/\/$/, '')}/api/users`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${RW_API_TOKEN}`
      },
      body: JSON.stringify(payload)
    });

    const rawText = await response.text();
    let responseBody;

    try {
      responseBody = rawText ? JSON.parse(rawText) : null;
    } catch {
      responseBody = { rawText };
    }

    if (!response.ok) {
      return res.status(response.status).json({
        error: 'RW API request failed',
        rwStatus: response.status,
        rwResponse: responseBody,
        payloadSent: payload
      });
    }

    const subscriptionUrl =
      responseBody?.response?.subscriptionUrl ??
      responseBody?.subscriptionUrl ??
      null;

    return res.status(201).json({
      message: 'Temporary subscription created',
      subscriptionUrl,
      request: {
        username: payload.username,
        description: payload.description,
        trafficLimitBytes: payload.trafficLimitBytes,
        trafficLimitMb: 100,
        expireAt: payload.expireAt
      },
      rwResponse: responseBody
    });
  } catch (error) {
    return res.status(502).json({
      error: 'Failed to reach RW API',
      details: error instanceof Error ? error.message : 'Unknown error',
      payloadSent: payload
    });
  }
});

app.use('/api', (_req, res) => {
  res.status(404).json({ error: 'Not found' });
});

if (fs.existsSync(CLIENT_INDEX_PATH)) {
  app.use(express.static(CLIENT_DIST_PATH));

  app.get('*', (_req, res) => {
    res.sendFile(CLIENT_INDEX_PATH);
  });
}

app.listen(PORT, () => {
  console.log(`Server running on port ${PORT}`);
});
