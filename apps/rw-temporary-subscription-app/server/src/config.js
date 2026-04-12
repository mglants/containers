import crypto from 'crypto';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const DEFAULT_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB = 100;

function parsePositiveNumber(value, fallback) {
  const parsed = Number.parseFloat(String(value ?? '').trim());

  if (Number.isFinite(parsed) && parsed > 0) {
    return parsed;
  }

  return fallback;
}

function parseTrafficLimitMb(env) {
  return Number(
    parsePositiveNumber(
      env.RW_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB,
      DEFAULT_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB
    ).toFixed(2)
  );
}

export function loadConfig(env = process.env) {
  const appAuthPassword = env.APP_AUTH_PASSWORD || '';
  const appAuthEnabled = Boolean(appAuthPassword);
  const appAuthCookieSameSite = env.APP_AUTH_COOKIE_SAME_SITE?.trim() || 'Lax';
  const appAuthCookieSecure = appAuthCookieSameSite.toLowerCase() === 'none';
  const telegramBotUsername = (
    env.TELEGRAM_BOT_USERNAME ||
    env.VITE_TELEGRAM_BOT_USERNAME ||
    ''
  ).replace(/^@/, '');
  const telegramBotToken = env.TELEGRAM_BOT_TOKEN || '';
  const registrationBotUsername = (
    env.REGISTRATION_BOT_USERNAME ||
    env.VITE_REGISTRATION_BOT_USERNAME ||
    ''
  ).replace(/^@/, '');
  const tempSubscriptionTrafficLimitMb = parseTrafficLimitMb(env);

  return {
    port: env.PORT || 3001,
    rwApiBaseUrl: env.RW_API_BASE_URL,
    rwApiToken: env.RW_API_TOKEN,
    rwSquadUuid: env.RW_SQUAD_UUID?.trim(),
    appAuthPassword,
    appAuthEnabled,
    appAuthCookieName: 'rw_app_auth',
    appAuthCookieMaxAgeSeconds: 60 * 60 * 24 * 30,
    appAuthCookieSameSite,
    appAuthCookieSecure,
    appAuthCookieValue: appAuthEnabled
      ? crypto.createHash('sha256').update(appAuthPassword).digest('hex')
      : '',
    defaultTempSubscriptionTrafficLimitMb:
      DEFAULT_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB,
    tempSubscriptionTrafficLimitMb,
    clientDistPath: path.resolve(__dirname, '../../client/dist'),
    clientIndexPath: path.resolve(__dirname, '../../client/dist/index.html'),
    tempSubscriptionLinksPath:
      env.TEMP_SUBSCRIPTION_LINKS_PATH?.trim() ||
      path.resolve(__dirname, '../data/temporary-subscriptions.json'),
    tempSubscriptionRetentionMs: 1000 * 60 * 60 * 48,
    telegramBotUsername,
    telegramBotToken,
    telegramBotPollTimeoutSeconds: Math.max(
      1,
      Number.parseInt(env.TELEGRAM_BOT_POLL_TIMEOUT_SECONDS || '30', 10) || 30
    ),
    telegramBotApiBaseUrl: telegramBotToken
      ? `https://api.telegram.org/bot${telegramBotToken}`
      : '',
    registrationBotUrl:
      env.REGISTRATION_BOT_URL?.trim() ||
      (registrationBotUsername ? `https://t.me/${registrationBotUsername}` : null),
    runtimeClientConfig: {
      VITE_API_BASE_URL: env.VITE_API_BASE_URL || '',
      VITE_TELEGRAM_BOT_USERNAME: telegramBotUsername,
      VITE_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB: String(
        tempSubscriptionTrafficLimitMb
      )
    }
  };
}
