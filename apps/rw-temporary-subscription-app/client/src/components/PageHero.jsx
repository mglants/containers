import React from 'react';

import LanguageSwitch from './LanguageSwitch.jsx';

export default function PageHero({
  eyebrow,
  languageLabel,
  locale,
  onLocaleChange,
  title,
  copy,
  children
}) {
  return (
    <section className="hero-card">
      <div className="hero-head">
        <p className="eyebrow">{eyebrow}</p>
        <LanguageSwitch
          ariaLabel={languageLabel}
          locale={locale}
          onLocaleChange={onLocaleChange}
        />
      </div>
      <h1>{title}</h1>
      <p className="hero-copy">{copy}</p>
      {children}
    </section>
  );
}
