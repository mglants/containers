import {
  buildTelegramBotUrl,
  createHttpError,
  generateTempSubscriptionId,
  normalizeLocale,
  parseTelegramId
} from './utils.js';

export function createSubscriptionService({
  config,
  log,
  tempSubscriptionStore,
  remnawaveClient
}) {
  async function createTemporarySubscription({ requestId, preferredLocale }) {
    const result = await remnawaveClient.createTemporarySubscription({ requestId });
    const tempSubscriptionId = generateTempSubscriptionId();
    const telegramBotUrl = buildTelegramBotUrl(
      config.telegramBotUsername,
      tempSubscriptionId
    );

    tempSubscriptionStore.set({
      tempSubscriptionId,
      createdAt: new Date().toISOString(),
      expireAt: result.payload.expireAt,
      requestId,
      rwUserUuid: result.rwUser?.uuid ?? null,
      rwUserId: result.rwUser?.id ?? null,
      rwShortUuid: result.rwUser?.shortUuid ?? null,
      rwUsername: result.rwUser?.username ?? result.payload.username,
      preferredLocale: normalizeLocale(preferredLocale, 'en'),
      subscriptionUrl: result.subscriptionUrl,
      linkedAt: null,
      linkedTelegramId: null
    });

    log('info', 'Temporary subscription created', {
      requestId,
      username: result.username,
      rwStatus: result.rwStatus,
      tempSubscriptionId,
      hasSubscriptionUrl: Boolean(result.subscriptionUrl),
      hasTelegramBotUrl: Boolean(telegramBotUrl)
    });

    return {
      message: 'Temporary subscription created',
      tempSubscriptionId,
      subscriptionUrl: result.subscriptionUrl,
      telegramBotUrl,
      request: {
        username: result.payload.username,
        description: result.payload.description,
        trafficLimitMb: result.trafficLimitMb,
        expireAt: result.payload.expireAt
      },
      rwResponse: result.responseBody
    };
  }

  async function linkTemporarySubscriptionToTelegram({
    requestId,
    tempSubscriptionId,
    telegramId
  }) {
    if (!tempSubscriptionId) {
      throw createHttpError('tempSubscriptionId is required', 400);
    }

    const normalizedTelegramId =
      typeof telegramId === 'object' && telegramId?.asString
        ? telegramId
        : parseTelegramId(telegramId);

    if (!normalizedTelegramId) {
      throw createHttpError('telegramId must be a numeric Telegram user id', 400);
    }

    const tempSubscription = tempSubscriptionStore.get(tempSubscriptionId);
    if (!tempSubscription) {
      throw createHttpError('Temporary subscription not found', 404);
    }

    const currentRwUserUuid =
      tempSubscription.rwUserUuid ||
      (await remnawaveClient.resolveUserByUsername({
        requestId,
        username: tempSubscription.rwUsername
      }))?.uuid ||
      null;

    if (tempSubscription.linkedTelegramId) {
      if (tempSubscription.linkedTelegramId !== normalizedTelegramId.asString) {
        throw createHttpError(
          'Temporary subscription already linked to another Telegram user',
          409
        );
      }

      return {
        status: 'already-linked',
        registrationBotUrl: config.registrationBotUrl,
        tempSubscription,
        user: {
          rwUserUuid: tempSubscription.rwUserUuid,
          previousUsername: tempSubscription.rwUsername,
          username: tempSubscription.rwUsername,
          telegramId: tempSubscription.linkedTelegramId
        }
      };
    }

    const usersWithSameTelegramId = await remnawaveClient.getUsersByTelegramId({
      requestId,
      telegramId: normalizedTelegramId.asString
    });
    const conflictingUser = usersWithSameTelegramId.find(
      (user) => user?.uuid && user.uuid !== currentRwUserUuid
    );

    if (conflictingUser) {
      log('warn', 'Refused to link Telegram ID already used by another RW user', {
        requestId,
        tempSubscriptionId,
        telegramId: normalizedTelegramId.asString,
        currentRwUserUuid,
        conflictingRwUserUuid: conflictingUser.uuid,
        conflictingRwUsername: conflictingUser.username ?? null
      });

      throw createHttpError(
        'Telegram ID is already linked to another Remnawave user',
        409,
        {
          conflictingUser: {
            uuid: conflictingUser.uuid,
            username: conflictingUser.username ?? null
          }
        }
      );
    }

    const updatedRwUser = await remnawaveClient.updateUserWithTelegram({
      requestId,
      rwUserUuid: currentRwUserUuid,
      rwUsername: tempSubscription.rwUsername,
      telegramId: normalizedTelegramId
    });

    const linkedRecord = tempSubscriptionStore.update(
      tempSubscriptionId,
      (currentRecord) => ({
        ...currentRecord,
        linkedAt: new Date().toISOString(),
        linkedTelegramId: normalizedTelegramId.asString,
        rwUserUuid:
          updatedRwUser.rwUser?.uuid ??
          updatedRwUser.rwUserUuid ??
          currentRecord.rwUserUuid,
        rwUsername: updatedRwUser.rwUser?.username ?? currentRecord.rwUsername
      })
    );

    log('info', 'Temporary subscription linked to Telegram user', {
      requestId,
      tempSubscriptionId,
      rwUserUuid: updatedRwUser.rwUser?.uuid ?? updatedRwUser.rwUserUuid ?? null,
      telegramId: normalizedTelegramId.asString,
      hasRegistrationBotUrl: Boolean(config.registrationBotUrl)
    });

    return {
      status: 'linked',
      registrationBotUrl: config.registrationBotUrl,
      tempSubscription: linkedRecord,
      user: {
        rwUserUuid: updatedRwUser.rwUser?.uuid ?? updatedRwUser.rwUserUuid ?? null,
        previousUsername: tempSubscription.rwUsername,
        username: updatedRwUser.rwUser?.username ?? tempSubscription.rwUsername,
        telegramId: normalizedTelegramId.asString
      },
      rwResponse: updatedRwUser.rwResponse
    };
  }

  return {
    createTemporarySubscription,
    linkTemporarySubscriptionToTelegram
  };
}
