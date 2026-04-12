import React from 'react';

export default function SubscriptionResultPanel({
  result,
  copyState,
  onCopySubscriptionUrl,
  activationBotUrl,
  t
}) {
  if (result?.type === 'success') {
    return (
      <section className="result-panel success-panel">
        <h2>{t.successTitle}</h2>
        <p>
          {t.userLabel} <strong>{result.data?.request?.username}</strong>{' '}
          {t.expiresAtLabel} <strong>{result.data?.request?.expireAt}</strong>.
        </p>

        {copyState === 'copied' ? (
          <p className="copy-feedback">{t.copiedToClipboard}</p>
        ) : null}
        {copyState === 'error' ? (
          <p className="copy-feedback">{t.copyFailed}</p>
        ) : null}

        <section className="activation-guide">
          <h3>{t.activationTitle}</h3>
          {result.data?.subscriptionUrl ? (
            <ol>
              <li>
                {t.activationStepOnePrefix}{' '}
                <a
                  href={result.data.subscriptionUrl}
                  target="_blank"
                  rel="noreferrer"
                >
                  {result.data.subscriptionUrl}
                </a>
              </li>
              <li>{t.activationStepTwo}</li>
              <li>
                {t.activationStepThreePrefix}{' '}
                <button
                  type="button"
                  onClick={onCopySubscriptionUrl}
                  className="copy-button"
                >
                  {t.copyLink}
                </button>
              </li>
              <li>
                {t.activationStepFourPrefix}{' '}
                {activationBotUrl ? (
                  <a href={activationBotUrl} target="_blank" rel="noreferrer">
                    {t.openTelegramBot}
                  </a>
                ) : (
                  t.setBotUsernameHint
                )}
              </li>
              <li>{t.activationStepFive}</li>
            </ol>
          ) : (
            <p>{t.missingSubscriptionUrl}</p>
          )}
        </section>
      </section>
    );
  }

  if (result?.type === 'error') {
    return (
      <section className="result-panel error-panel">
        <h2>{t.errorTitle}</h2>
        <p>{result.message}</p>
      </section>
    );
  }

  return null;
}
