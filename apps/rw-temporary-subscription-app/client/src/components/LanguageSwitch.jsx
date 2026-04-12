import React from 'react';

export default function LanguageSwitch({
  ariaLabel,
  locale,
  onLocaleChange
}) {
  return (
    <div className="language-switch" role="group" aria-label={ariaLabel}>
      <button
        type="button"
        className={`lang-button ${locale === 'ru' ? 'active' : ''}`}
        onClick={() => onLocaleChange('ru')}
      >
        RU
      </button>
      <button
        type="button"
        className={`lang-button ${locale === 'en' ? 'active' : ''}`}
        onClick={() => onLocaleChange('en')}
      >
        EN
      </button>
    </div>
  );
}
