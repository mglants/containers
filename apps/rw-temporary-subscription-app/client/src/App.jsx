import React, { useEffect, useMemo, useState } from 'react';

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:3001';
const SUBSCRIPTION_STORAGE_KEY = 'rw-temp-subscription';
const TELEGRAM_BOT_USERNAME_RAW = import.meta.env.VITE_TELEGRAM_BOT_USERNAME || '';
const TELEGRAM_BOT_USERNAME = TELEGRAM_BOT_USERNAME_RAW.replace(/^@/, '');
const TELEGRAM_BOT_URL = TELEGRAM_BOT_USERNAME ? `https://t.me/${TELEGRAM_BOT_USERNAME}` : null;

const TEXTS = {
  en: {
    languageLabel: 'Language',
    authChecking: 'Checking access...',
    authTitle: 'Enter Password',
    authDescription: 'Access to this page is protected by a password.',
    authPasswordLabel: 'Password',
    authPasswordPlaceholder: 'Enter password',
    authButton: 'Continue',
    authInvalid: 'Invalid password.',
    eyebrow: 'Mini Subscription',
    title: 'Create a temporary subscription',
    heroCopy:
      'This form creates one temporary user for perisistent Telegram registration',
    traffic: 'Traffic',
    expires: 'Expires',
    formCopy: 'Click to create a temporary subscription.',
    singleSubscriptionNotice:
      'Only one temporary subscription can be created.',
    creatingButton: 'Creating...',
    createButton: 'Create temporary subscription',
    alreadyCreatedButton: 'Subscription already created',
    successTitle: 'Subscription created',
    userLabel: 'User',
    expiresAtLabel: 'is set to expire at',
    copyLink: 'Copy link',
    missingSubscriptionUrl: 'Subscription URL is not present in backend response.',
    copiedToClipboard: 'Copied to clipboard.',
    copyFailed: 'Copy failed. Copy manually.',
    activationTitle: 'Activate permanent subscription',
    activationStepOnePrefix: 'Go to subscription link:',
    activationStepTwo: 'Install VPN client according to the instructions.',
    activationStepThreePrefix: 'Add the subscription link to VPN client:',
    activationStepFourPrefix: 'Open Telegram bot and sign up:',
    setBotUsernameHint: 'set VITE_TELEGRAM_BOT_USERNAME in .env',
    errorTitle: 'Creation failed'
  },
  ru: {
    languageLabel: 'Язык',
    authChecking: 'Проверка доступа...',
    authTitle: 'Введите пароль',
    authDescription: 'Доступ к этой странице защищен паролем.',
    authPasswordLabel: 'Пароль',
    authPasswordPlaceholder: 'Введите пароль',
    authButton: 'Продолжить',
    authInvalid: 'Неверный пароль.',
    eyebrow: 'Мини подписка',
    title: 'Создать временную подписку',
    heroCopy:
      'Эта форма создает одного временного пользователя для последующей регистрации через Telegram',
    traffic: 'Трафик',
    expires: 'Истекает',
    formCopy: 'Нажмите кнопку для создания временной подписки.',
    singleSubscriptionNotice:
      'Можно создать только одну временную подписку.',
    creatingButton: 'Создание...',
    createButton: 'Создать временную подписку',
    alreadyCreatedButton: 'Подписка уже создана',
    successTitle: 'Подписка создана',
    userLabel: 'Пользователь',
    expiresAtLabel: 'будет активен до',
    copyLink: 'Копировать ссылку',
    missingSubscriptionUrl: 'В ответе backend отсутствует subscription URL.',
    copiedToClipboard: 'Скопировано в буфер обмена.',
    copyFailed: 'Не удалось скопировать. Скопируйте вручную.',
    activationTitle: 'Как активировать постоянную подписку',
    activationStepOnePrefix: 'Перейдите по ссылке подписки:',
    activationStepTwo: 'Установите VPN-клиент по инструкции.',
    activationStepThreePrefix: 'Добавьте ссылку подписки в VPN-клиент:',
    activationStepFourPrefix: 'Откройте Telegram-бота и зарегистрируйтесь:',
    setBotUsernameHint: 'укажите VITE_TELEGRAM_BOT_USERNAME в .env',
    errorTitle: 'Ошибка создания'
  }
};

function getDefaultLocale() {
  if (typeof navigator === 'undefined') {
    return 'en';
  }

  return navigator.language?.toLowerCase().startsWith('ru') ? 'ru' : 'en';
}

function getTomorrowLabel(locale) {
  const date = new Date();
  date.setDate(date.getDate() + 1);
  return date.toLocaleString(locale === 'ru' ? 'ru-RU' : 'en-US');
}

function getStoredSuccessResult() {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    const raw = window.localStorage.getItem(SUBSCRIPTION_STORAGE_KEY);

    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === 'object' ? { type: 'success', data: parsed } : null;
  } catch {
    return null;
  }
}

function storeSuccessResult(data) {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(SUBSCRIPTION_STORAGE_KEY, JSON.stringify(data));
  } catch {
    // Ignore storage errors and keep runtime state only.
  }
}

export default function App() {
  const [locale, setLocale] = useState(() => getDefaultLocale());
  const [authState, setAuthState] = useState('checking');
  const [authPassword, setAuthPassword] = useState('');
  const [authError, setAuthError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState(() => getStoredSuccessResult());
  const [copyState, setCopyState] = useState('idle');
  const hasExistingSubscription = result?.type === 'success';
  const t = TEXTS[locale] || TEXTS.en;
  const expiresLabel = useMemo(() => getTomorrowLabel(locale), [locale]);

  useEffect(() => {
    let cancelled = false;

    async function checkAuth() {
      try {
        const response = await fetch(`${API_BASE_URL}/auth/status`, {
          credentials: 'include'
        });
        const data = await response.json();

        if (!cancelled) {
          if (data?.authEnabled && !data?.authenticated) {
            setAuthState('required');
          } else {
            setAuthState('authenticated');
          }
        }
      } catch {
        if (!cancelled) {
          setAuthState('required');
        }
      }
    }

    checkAuth();

    return () => {
      cancelled = true;
    };
  }, []);

  async function handleAuthSubmit(event) {
    event.preventDefault();
    setAuthError('');

    try {
      const response = await fetch(`${API_BASE_URL}/auth/session`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ password: authPassword })
      });

      if (!response.ok) {
        setAuthError(t.authInvalid);
        return;
      }

      setAuthPassword('');
      setAuthState('authenticated');
    } catch {
      setAuthError(t.authInvalid);
    }
  }

  async function handleSubmit(event) {
    event.preventDefault();

    if (hasExistingSubscription || authState !== 'authenticated') {
      return;
    }

    setSubmitting(true);
    setResult(null);
    setCopyState('idle');

    try {
      const response = await fetch(`${API_BASE_URL}/api/subscriptions/temporary`, {
        method: 'POST',
        credentials: 'include'
      });
      const data = await response.json();

      if (!response.ok) {
        throw new Error(data?.details || data?.error || 'Request failed');
      }

      const successResult = { type: 'success', data };
      setResult(successResult);
      storeSuccessResult(data);
    } catch (error) {
      setResult({
        type: 'error',
        message: error instanceof Error ? error.message : 'Unknown error'
      });
    } finally {
      setSubmitting(false);
    }
  }

  async function handleCopySubscriptionUrl() {
    const subscriptionUrl = result?.data?.subscriptionUrl;

    if (!subscriptionUrl) {
      return;
    }

    try {
      await navigator.clipboard.writeText(subscriptionUrl);
      setCopyState('copied');
    } catch {
      setCopyState('error');
    }
  }

  if (authState !== 'authenticated') {
    return (
      <main className="page-shell">
        <section className="hero-card">
          <div className="hero-head">
            <p className="eyebrow">{t.eyebrow}</p>
            <div className="language-switch" role="group" aria-label={t.languageLabel}>
              <button
                type="button"
                className={`lang-button ${locale === 'ru' ? 'active' : ''}`}
                onClick={() => setLocale('ru')}
              >
                RU
              </button>
              <button
                type="button"
                className={`lang-button ${locale === 'en' ? 'active' : ''}`}
                onClick={() => setLocale('en')}
              >
                EN
              </button>
            </div>
          </div>
          <h1>{t.authTitle}</h1>
          <p className="hero-copy">{authState === 'checking' ? t.authChecking : t.authDescription}</p>
        </section>

        {authState === 'required' ? (
          <section className="form-card">
            <form onSubmit={handleAuthSubmit} className="subscription-form">
              <label>
                <span>{t.authPasswordLabel}</span>
                <input
                  type="password"
                  value={authPassword}
                  onChange={(event) => setAuthPassword(event.target.value)}
                  placeholder={t.authPasswordPlaceholder}
                  required
                  autoComplete="current-password"
                />
              </label>
              {authError ? <p className="copy-feedback">{authError}</p> : null}
              <button type="submit">{t.authButton}</button>
            </form>
          </section>
        ) : null}
      </main>
    );
  }

  return (
    <main className="page-shell">
      <section className="hero-card">
        <div className="hero-head">
          <p className="eyebrow">{t.eyebrow}</p>
          <div className="language-switch" role="group" aria-label={t.languageLabel}>
            <button
              type="button"
              className={`lang-button ${locale === 'ru' ? 'active' : ''}`}
              onClick={() => setLocale('ru')}
            >
              RU
            </button>
            <button
              type="button"
              className={`lang-button ${locale === 'en' ? 'active' : ''}`}
              onClick={() => setLocale('en')}
            >
              EN
            </button>
          </div>
        </div>
        <h1>{t.title}</h1>
        <p className="hero-copy">{t.heroCopy}</p>

        <div className="summary-grid">
          <article className="summary-item">
            <span className="summary-label">{t.traffic}</span>
            <strong>100 MB</strong>
          </article>
          <article className="summary-item">
            <span className="summary-label">{t.expires}</span>
            <strong>{expiresLabel}</strong>
          </article>
        </div>
      </section>

      <section className="form-card">
        <form onSubmit={handleSubmit} className="subscription-form">
          <p className="form-copy">{t.formCopy}</p>
          <p className="single-subscription-note">{t.singleSubscriptionNotice}</p>
          <button type="submit" disabled={submitting || hasExistingSubscription}>
            {submitting ? t.creatingButton : hasExistingSubscription ? t.alreadyCreatedButton : t.createButton}
          </button>
        </form>

        {result?.type === 'success' ? (
          <section className="result-panel success-panel">
            <h2>{t.successTitle}</h2>
            <p>
              {t.userLabel} <strong>{result.data?.request?.username}</strong> {t.expiresAtLabel}{' '}
              <strong>{result.data?.request?.expireAt}</strong>.
            </p>

            {copyState === 'copied' ? <p className="copy-feedback">{t.copiedToClipboard}</p> : null}
            {copyState === 'error' ? <p className="copy-feedback">{t.copyFailed}</p> : null}

            <section className="activation-guide">
              <h3>{t.activationTitle}</h3>
              {result.data?.subscriptionUrl ? (
                <ol>
                  <li>
                    {t.activationStepOnePrefix}{' '}
                    <a href={result.data.subscriptionUrl} target="_blank" rel="noreferrer">
                      {result.data.subscriptionUrl}
                    </a>
                  </li>
                  <li>{t.activationStepTwo}</li>
                  <li>
                    {t.activationStepThreePrefix}{' '}
                    <button type="button" onClick={handleCopySubscriptionUrl} className="copy-button">
                      {t.copyLink}
                    </button>
                  </li>
                  <li>
                    {t.activationStepFourPrefix}{' '}
                    {TELEGRAM_BOT_URL ? (
                      <a href={TELEGRAM_BOT_URL} target="_blank" rel="noreferrer">
                        @{TELEGRAM_BOT_USERNAME}
                      </a>
                    ) : (
                      t.setBotUsernameHint
                    )}
                  </li>
                </ol>
              ) : (
                <p>{t.missingSubscriptionUrl}</p>
              )}
            </section>
          </section>
        ) : null}

        {result?.type === 'error' ? (
          <section className="result-panel error-panel">
            <h2>{t.errorTitle}</h2>
            <p>{result.message}</p>
          </section>
        ) : null}
      </section>
    </main>
  );
}
