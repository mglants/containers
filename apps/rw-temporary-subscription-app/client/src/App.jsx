import React, { useEffect, useMemo, useState } from 'react';

import AuthScreen from './components/AuthScreen.jsx';
import SubscriptionScreen from './components/SubscriptionScreen.jsx';
import {
  API_BASE_URL,
  formatTrafficLimit,
  TELEGRAM_BOT_URL,
  TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB
} from './config.js';
import {
  getStoredSuccessResult,
  storeSuccessResult
} from './subscription-storage.js';
import { getDefaultLocale, getTomorrowLabel, TEXTS } from './texts.js';

export default function App() {
  const [locale, setLocale] = useState(() => getDefaultLocale());
  const [authState, setAuthState] = useState('checking');
  const [authPassword, setAuthPassword] = useState('');
  const [authError, setAuthError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState(() => getStoredSuccessResult());
  const [copyState, setCopyState] = useState('idle');

  const hasExistingSubscription = result?.type === 'success';
  const activationBotUrl = result?.data?.telegramBotUrl || TELEGRAM_BOT_URL;
  const t = TEXTS[locale] || TEXTS.en;
  const expiresLabel = useMemo(() => getTomorrowLabel(locale), [locale]);
  const trafficLabel = useMemo(
    () => formatTrafficLimit(TEMP_SUBSCRIPTION_TRAFFIC_LIMIT_MB, locale),
    [locale]
  );

  useEffect(() => {
    let cancelled = false;

    async function checkAuth() {
      try {
        const response = await fetch(`${API_BASE_URL}/auth/status`, {
          credentials: 'include'
        });
        const data = await response.json();

        if (cancelled) {
          return;
        }

        setAuthState(
          data?.authEnabled && !data?.authenticated
            ? 'required'
            : 'authenticated'
        );
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

  async function handleCreateSubscription(event) {
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
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ locale })
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
      <AuthScreen
        locale={locale}
        onLocaleChange={setLocale}
        authState={authState}
        authPassword={authPassword}
        onAuthPasswordChange={setAuthPassword}
        authError={authError}
        onAuthSubmit={handleAuthSubmit}
        t={t}
      />
    );
  }

  return (
    <SubscriptionScreen
      locale={locale}
      onLocaleChange={setLocale}
      trafficLabel={trafficLabel}
      expiresLabel={expiresLabel}
      submitting={submitting}
      hasExistingSubscription={hasExistingSubscription}
      onSubmit={handleCreateSubscription}
      result={result}
      copyState={copyState}
      onCopySubscriptionUrl={handleCopySubscriptionUrl}
      activationBotUrl={activationBotUrl}
      t={t}
    />
  );
}
