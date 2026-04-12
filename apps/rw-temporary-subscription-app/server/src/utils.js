import crypto from 'crypto';

export function timingSafeEqual(a, b) {
  const left = Buffer.from(a, 'utf8');
  const right = Buffer.from(b, 'utf8');

  if (left.length !== right.length) {
    return false;
  }

  return crypto.timingSafeEqual(left, right);
}

export function buildTelegramBotUrl(botUsername, startParam) {
  if (!botUsername) {
    return null;
  }

  const url = new URL(`https://t.me/${botUsername}`);
  if (startParam) {
    url.searchParams.set('start', startParam);
  }

  return url.toString();
}

export function generateTempSubscriptionId() {
  return crypto.randomBytes(18).toString('base64url');
}

export function parseTelegramId(rawValue) {
  const telegramId = String(rawValue ?? '').trim();

  if (!/^\d{5,20}$/.test(telegramId)) {
    return null;
  }

  const numericTelegramId = Number(telegramId);
  if (!Number.isSafeInteger(numericTelegramId)) {
    return null;
  }

  return {
    asNumber: numericTelegramId,
    asString: telegramId
  };
}

export async function parseJsonResponse(response) {
  const rawText = await response.text();

  try {
    return rawText ? JSON.parse(rawText) : null;
  } catch {
    return rawText ? { rawText } : null;
  }
}

export function getRwUserFromResponse(responseBody) {
  if (!responseBody || typeof responseBody !== 'object') {
    return null;
  }

  if (responseBody.response && typeof responseBody.response === 'object') {
    return responseBody.response;
  }

  if (responseBody.user && typeof responseBody.user === 'object') {
    return responseBody.user;
  }

  return null;
}

export function createHttpError(message, httpStatus, properties = {}) {
  const error = new Error(message);
  error.httpStatus = httpStatus;
  Object.assign(error, properties);
  return error;
}

export function normalizeLocale(rawLocale, fallback = 'en') {
  const locale = String(rawLocale ?? '').trim().toLowerCase();

  if (locale.startsWith('ru')) {
    return 'ru';
  }

  if (locale.startsWith('en')) {
    return 'en';
  }

  return fallback;
}

export function getTomorrowIso() {
  const date = new Date();
  date.setDate(date.getDate() + 1);
  return date.toISOString();
}

export function generateUsername() {
  const timestamp = Date.now().toString(36);
  const randomSuffix = Math.random().toString(36).slice(2, 8);
  return `temp-${timestamp}-${randomSuffix}`;
}

export function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
