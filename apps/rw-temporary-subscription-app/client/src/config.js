function readRuntimeConfig(name) {
  if (typeof window === 'undefined') {
    return '';
  }

  const runtimeConfig = window.__RW_RUNTIME_CONFIG;
  const value =
    runtimeConfig && typeof runtimeConfig === 'object' ? runtimeConfig[name] : '';

  return typeof value === 'string' ? value : '';
}

const DEFAULT_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB = 100;
const MEGABYTES_IN_GIGABYTE = 1024;

function parseTrafficLimitMb() {
  const rawMegabytes =
    readRuntimeConfig('VITE_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB') ||
    import.meta.env.VITE_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB ||
    String(DEFAULT_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB);
  const parsedMegabytes = Number.parseFloat(rawMegabytes);

  if (Number.isFinite(parsedMegabytes) && parsedMegabytes > 0) {
    return parsedMegabytes;
  }

  return DEFAULT_TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB;
}

export const API_BASE_URL =
  readRuntimeConfig('VITE_API_BASE_URL') ||
  import.meta.env.VITE_API_BASE_URL ||
  (import.meta.env.DEV ? 'http://localhost:3001' : '');
export const TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB = parseTrafficLimitMb();

const TELEGRAM_BOT_USERNAME_RAW =
  readRuntimeConfig('VITE_TELEGRAM_BOT_USERNAME') ||
  import.meta.env.VITE_TELEGRAM_BOT_USERNAME ||
  '';

export const TELEGRAM_BOT_USERNAME = TELEGRAM_BOT_USERNAME_RAW.replace(/^@/, '');
export const TELEGRAM_BOT_URL = TELEGRAM_BOT_USERNAME
  ? `https://t.me/${TELEGRAM_BOT_USERNAME}`
  : null;

export function formatTrafficLimit(megabytes, locale = 'en') {
  const normalizedMegabytes =
    Number.isFinite(megabytes) && megabytes > 0 ? megabytes : 0;
  const numberFormatter = new Intl.NumberFormat(
    locale === 'ru' ? 'ru-RU' : 'en-US',
    {
      maximumFractionDigits: 2
    }
  );

  if (normalizedMegabytes >= MEGABYTES_IN_GIGABYTE) {
    return `${numberFormatter.format(
      normalizedMegabytes / MEGABYTES_IN_GIGABYTE
    )} ${locale === 'ru' ? 'ГБ' : 'GB'}`;
  }

  return `${numberFormatter.format(normalizedMegabytes)} ${locale === 'ru' ? 'МБ' : 'MB'}`;
}
