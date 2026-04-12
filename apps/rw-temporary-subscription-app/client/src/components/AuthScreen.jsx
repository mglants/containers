import React from 'react';

import PageHero from './PageHero.jsx';

export default function AuthScreen({
  locale,
  onLocaleChange,
  authState,
  authPassword,
  onAuthPasswordChange,
  authError,
  onAuthSubmit,
  t
}) {
  return (
    <main className="page-shell">
      <PageHero
        eyebrow={t.eyebrow}
        languageLabel={t.languageLabel}
        locale={locale}
        onLocaleChange={onLocaleChange}
        title={t.authTitle}
        copy={authState === 'checking' ? t.authChecking : t.authDescription}
      />

      {authState === 'required' ? (
        <section className="form-card">
          <form onSubmit={onAuthSubmit} className="subscription-form">
            <label>
              <span>{t.authPasswordLabel}</span>
              <input
                type="password"
                value={authPassword}
                onChange={(event) => onAuthPasswordChange(event.target.value)}
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
