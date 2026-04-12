export const TEXTS = {
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
      'This page creates a temporary Remnawave user and hands it off to Telegram for permanent registration.',
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
    missingSubscriptionUrl:
      'Subscription URL is not present in backend response.',
    copiedToClipboard: 'Copied to clipboard.',
    copyFailed: 'Copy failed. Copy manually.',
    activationTitle: 'Finish registration',
    activationStepOnePrefix: 'Go to subscription link:',
    activationStepTwo: 'Install VPN client according to the instructions.',
    activationStepThreePrefix: 'Add the subscription link to VPN client and press connect:',
    activationStepFourPrefix: 'Open Telegram via your personal handoff link:',
    activationStepFive:
      'The bot will read your Telegram ID, link it to the temporary Remnawave user, and send the registration link in the next bot.',
    openTelegramBot: 'Open Telegram bot',
    setBotUsernameHint:
      'set TELEGRAM_BOT_USERNAME or VITE_TELEGRAM_BOT_USERNAME in .env',
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
      'Эта страница создает временного пользователя Remnawave и передает его в Telegram для постоянной регистрации.',
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
    missingSubscriptionUrl:
      'В ответе backend отсутствует subscription URL.',
    copiedToClipboard: 'Скопировано в буфер обмена.',
    copyFailed: 'Не удалось скопировать. Скопируйте вручную.',
    activationTitle: 'Как завершить регистрацию',
    activationStepOnePrefix: 'Перейдите по ссылке подписки:',
    activationStepTwo: 'Установите VPN-клиент по инструкции.',
    activationStepThreePrefix: 'Добавьте ссылку подписки в VPN-клиент и нажмите подключиться:',
    activationStepFourPrefix: 'Откройте Telegram по персональной ссылке:',
    activationStepFive:
      'Бот получит ваш Telegram ID, привяжет его к временному пользователю в Remnawave и пришлет ссылку для регистрации в следующем боте.',
    openTelegramBot: 'Открыть Telegram-бота',
    setBotUsernameHint:
      'укажите TELEGRAM_BOT_USERNAME или VITE_TELEGRAM_BOT_USERNAME в .env',
    errorTitle: 'Ошибка создания'
  }
};

export function getDefaultLocale() {
  if (typeof navigator === 'undefined') {
    return 'ru';
  }

  return navigator.language?.toLowerCase().startsWith('ru') ? 'ru' : 'en';
}

export function getTomorrowLabel(locale) {
  const date = new Date();
  date.setDate(date.getDate() + 1);
  return date.toLocaleString(locale === 'ru' ? 'ru-RU' : 'en-US');
}
