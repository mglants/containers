import { buildErrorMeta, stringifyForLog } from './logger.js';
import { normalizeLocale, parseJsonResponse, sleep } from './utils.js';

const TELEGRAM_TEXTS = {
  en: {
    bareStartPrompt:
      'Open the personal Telegram link from the temporary subscription page so I can identify your subscription.',
    alreadyLinked:
      'This temporary subscription is already linked to your Telegram account.',
    linked: 'Temporary subscription linked to your Telegram account.',
    usernameLabel: 'Remnawave username',
    continueRegistration: 'Continue registration here',
    registrationMissing:
      'Registration bot link is not configured yet. Ask the administrator to set REGISTRATION_BOT_URL.',
    notFound:
      'Temporary subscription was not found or already expired. Open the website and create a new one.',
    existingTelegram:
      'A Remnawave user with this Telegram account already exists. Use your existing account or contact the administrator.',
    linkedDifferent:
      'This temporary subscription is already linked to a different Telegram account.',
    genericError:
      'Failed to finish the handoff. Please try again later or contact the administrator.'
  },
  ru: {
    bareStartPrompt:
      'Откройте персональную ссылку Telegram со страницы временной подписки, чтобы я смог определить вашу подписку.',
    alreadyLinked:
      'Эта временная подписка уже привязана к вашему Telegram-аккаунту.',
    linked: 'Временная подписка привязана к вашему Telegram-аккаунту.',
    usernameLabel: 'Имя пользователя Remnawave',
    continueRegistration: 'Продолжить регистрацию здесь',
    registrationMissing:
      'Ссылка на регистрационного бота пока не настроена. Попросите администратора указать REGISTRATION_BOT_URL.',
    notFound:
      'Временная подписка не найдена или уже истекла. Откройте сайт и создайте новую.',
    existingTelegram:
      'В Remnawave уже есть пользователь с этим Telegram-аккаунтом. Используйте существующий аккаунт или обратитесь к администратору.',
    linkedDifferent:
      'Эта временная подписка уже привязана к другому Telegram-аккаунту.',
    genericError:
      'Не удалось завершить привязку. Попробуйте позже или обратитесь к администратору.'
  }
};

export function createTelegramBot({
  config,
  log,
  linkTemporarySubscriptionToTelegram,
  getTempSubscriptionRecord
}) {
  async function callTelegramApi(method, payload = {}) {
    if (!config.telegramBotApiBaseUrl) {
      throw new Error('TELEGRAM_BOT_TOKEN is not configured');
    }

    const response = await fetch(`${config.telegramBotApiBaseUrl}/${method}`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(payload)
    });
    const responseBody = await parseJsonResponse(response);

    if (!response.ok || !responseBody?.ok) {
      const error = new Error(`Telegram API ${method} failed`);
      error.httpStatus = response.status;
      error.telegramResponse = responseBody;
      throw error;
    }

    return responseBody.result;
  }

  function extractTelegramStartParam(text) {
    if (typeof text !== 'string') {
      return null;
    }

    const match = text.trim().match(/^\/start(?:@\w+)?(?:\s+(\S+))?/i);
    return match?.[1] ?? null;
  }

  function resolveReplyLocale(telegramLanguageCode, storedLocale) {
    return (
      normalizeLocale(telegramLanguageCode, null) ||
      normalizeLocale(storedLocale, 'en')
    );
  }

  function buildTelegramStartReply(result, locale) {
    const t = TELEGRAM_TEXTS[locale] || TELEGRAM_TEXTS.en;
    const lines = [];

    if (result.status === 'already-linked') {
      lines.push(t.alreadyLinked);
    } else {
      lines.push(t.linked);
    }

    if (result.user?.username) {
      lines.push(`${t.usernameLabel}: ${result.user.username}`);
    }

    if (result.registrationBotUrl) {
      lines.push(`${t.continueRegistration}: ${result.registrationBotUrl}`);
    } else {
      lines.push(t.registrationMissing);
    }

    return lines.join('\n');
  }

  async function processTelegramMessage(update) {
    const message = update?.message;
    const chatId = message?.chat?.id;
    const fromId = message?.from?.id;
    const text = message?.text;

    if (!chatId || !fromId || typeof text !== 'string') {
      return;
    }

    const startParam = extractTelegramStartParam(text);

    if (!startParam) {
      const replyLocale = resolveReplyLocale(message?.from?.language_code);

      if (/^\/start(?:@\w+)?$/i.test(text.trim())) {
        await callTelegramApi('sendMessage', {
          chat_id: chatId,
          text: (TELEGRAM_TEXTS[replyLocale] || TELEGRAM_TEXTS.en).bareStartPrompt
        });
      }
      return;
    }

    const requestId = `telegram-update-${update.update_id ?? Date.now()}`;
    const storedPreferredLocale =
      getTempSubscriptionRecord?.(startParam)?.preferredLocale || null;
    const replyLocale = resolveReplyLocale(
      message?.from?.language_code,
      storedPreferredLocale
    );
    const t = TELEGRAM_TEXTS[replyLocale] || TELEGRAM_TEXTS.en;

    try {
      const result = await linkTemporarySubscriptionToTelegram({
        requestId,
        tempSubscriptionId: startParam,
        telegramId: String(fromId)
      });

      await callTelegramApi('sendMessage', {
        chat_id: chatId,
        text: buildTelegramStartReply(result, replyLocale)
      });
    } catch (error) {
      log('error', 'Failed to process Telegram /start handoff', {
        requestId,
        startParam,
        chatId,
        telegramUserId: String(fromId),
        ...buildErrorMeta(error),
        telegramResponsePreview: stringifyForLog(error?.telegramResponse),
        rwStatus: error?.rwStatus,
        rwResponsePreview: stringifyForLog(error?.rwResponse)
      });

      const messageText =
        error?.httpStatus === 404
          ? t.notFound
          : error?.conflictingUser
            ? t.existingTelegram
            : error?.httpStatus === 409
              ? t.linkedDifferent
              : t.genericError;

      try {
        await callTelegramApi('sendMessage', {
          chat_id: chatId,
          text: messageText
        });
      } catch (sendError) {
        log('error', 'Failed to send Telegram error reply', {
          requestId,
          chatId,
          telegramUserId: String(fromId),
          ...buildErrorMeta(sendError),
          telegramResponsePreview: stringifyForLog(sendError?.telegramResponse)
        });
      }
    }
  }

  function startPolling() {
    if (!config.telegramBotToken || !config.telegramBotUsername) {
      log('info', 'Telegram bot polling is disabled', {
        hasTelegramBotToken: Boolean(config.telegramBotToken),
        hasTelegramBotUsername: Boolean(config.telegramBotUsername)
      });
      return;
    }

    let offset = 0;

    async function poll() {
      try {
        const updates = await callTelegramApi('getUpdates', {
          offset,
          timeout: config.telegramBotPollTimeoutSeconds,
          allowed_updates: ['message']
        });

        for (const update of updates) {
          offset = Math.max(offset, Number(update.update_id) + 1);
          await processTelegramMessage(update);
        }
      } catch (error) {
        log('error', 'Telegram bot polling failed', {
          ...buildErrorMeta(error),
          telegramResponsePreview: stringifyForLog(error?.telegramResponse)
        });
        await sleep(3000);
      }

      setImmediate(poll);
    }

    callTelegramApi('deleteWebhook', { drop_pending_updates: false })
      .catch((error) => {
        log('warn', 'Failed to delete Telegram webhook before polling', {
          ...buildErrorMeta(error),
          telegramResponsePreview: stringifyForLog(error?.telegramResponse)
        });
      })
      .finally(() => {
        log('info', 'Starting Telegram bot polling', {
          telegramBotUsername: config.telegramBotUsername,
          pollTimeoutSeconds: config.telegramBotPollTimeoutSeconds
        });
        poll();
      });
  }

  return {
    startPolling
  };
}
