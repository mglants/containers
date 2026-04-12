import express from 'express';

import { timingSafeEqual } from './utils.js';

export function createAuthModule({ config }) {
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
    const cookieValue = cookies[config.appAuthCookieName];

    if (!cookieValue || !config.appAuthCookieValue) {
      return false;
    }

    return timingSafeEqual(cookieValue, config.appAuthCookieValue);
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
      `${config.appAuthCookieName}=${config.appAuthCookieValue}`,
      `Max-Age=${config.appAuthCookieMaxAgeSeconds}`,
      'Path=/',
      'HttpOnly',
      `SameSite=${config.appAuthCookieSameSite}`
    ];

    if (config.appAuthCookieSecure) {
      parts.push('Secure');
    }

    return parts.join('; ');
  }

  function requirePageAuth(req, res, next) {
    if (
      !config.appAuthEnabled ||
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

  const authRouter = express.Router();

  authRouter.get('/login', (req, res) => {
    if (!config.appAuthEnabled) {
      return res.redirect('/');
    }

    if (hasValidAuthCookie(req)) {
      return res.redirect('/');
    }

    return res.status(200).type('html').send(renderAuthPage());
  });

  authRouter.get('/status', (req, res) => {
    if (!config.appAuthEnabled) {
      return res.json({ authenticated: true, authEnabled: false });
    }

    return res.json({
      authenticated: hasValidAuthCookie(req),
      authEnabled: true
    });
  });

  authRouter.post('/session', (req, res) => {
    if (!config.appAuthEnabled) {
      return res.status(200).json({ authenticated: true });
    }

    const password = String(req.body?.password ?? '');

    if (!timingSafeEqual(password, config.appAuthPassword)) {
      return res.status(401).json({ error: 'Invalid password' });
    }

    res.setHeader('Set-Cookie', buildAuthCookie());
    return res.status(200).json({ authenticated: true });
  });

  authRouter.post('/login', (req, res) => {
    if (!config.appAuthEnabled) {
      return res.redirect('/');
    }

    const password = String(req.body?.password ?? '');
    if (timingSafeEqual(password, config.appAuthPassword)) {
      res.setHeader('Set-Cookie', buildAuthCookie());
      return res.redirect('/');
    }

    return res.status(401).type('html').send(renderAuthPage('Invalid password.'));
  });

  return {
    authRouter,
    requirePageAuth
  };
}
