import React from 'react';

import PageHero from './PageHero.jsx';
import SubscriptionResultPanel from './SubscriptionResultPanel.jsx';

export default function SubscriptionScreen({
  locale,
  onLocaleChange,
  trafficLabel,
  expiresLabel,
  submitting,
  hasExistingSubscription,
  onSubmit,
  result,
  copyState,
  onCopySubscriptionUrl,
  activationBotUrl,
  t
}) {
  return (
    <main className="page-shell">
      <PageHero
        eyebrow={t.eyebrow}
        languageLabel={t.languageLabel}
        locale={locale}
        onLocaleChange={onLocaleChange}
        title={t.title}
        copy={t.heroCopy}
      >
        <div className="summary-grid">
          <article className="summary-item">
            <span className="summary-label">{t.traffic}</span>
            <strong>{trafficLabel}</strong>
          </article>
          <article className="summary-item">
            <span className="summary-label">{t.expires}</span>
            <strong>{expiresLabel}</strong>
          </article>
        </div>
      </PageHero>

      <section className="form-card">
        <form onSubmit={onSubmit} className="subscription-form">
          <p className="form-copy">{t.formCopy}</p>
          <p className="single-subscription-note">{t.singleSubscriptionNotice}</p>
          <button type="submit" disabled={submitting || hasExistingSubscription}>
            {submitting
              ? t.creatingButton
              : hasExistingSubscription
                ? t.alreadyCreatedButton
                : t.createButton}
          </button>
        </form>

        <SubscriptionResultPanel
          result={result}
          copyState={copyState}
          onCopySubscriptionUrl={onCopySubscriptionUrl}
          activationBotUrl={activationBotUrl}
          t={t}
        />
      </section>
    </main>
  );
}
