import {
  createHttpError,
  generateUsername,
  getRwUserFromResponse,
  getTomorrowIso,
  parseJsonResponse
} from './utils.js';
import { buildErrorMeta, stringifyForLog } from './logger.js';

const BYTES_IN_MEGABYTE = 1024 * 1024;

export function createRemnawaveClient({ config, log }) {
  function assertConfigured(requestId) {
    if (config.rwApiBaseUrl && config.rwApiToken) {
      return;
    }

    log('error', 'Missing RW API configuration', {
      requestId,
      hasRwApiBaseUrl: Boolean(config.rwApiBaseUrl),
      hasRwApiToken: Boolean(config.rwApiToken)
    });

    throw createHttpError('Missing RW API configuration', 500, {
      clientResponse: {
        error: 'Missing RW API configuration',
        details: 'Set RW_API_BASE_URL and RW_API_TOKEN in your environment.'
      }
    });
  }

  function buildUsersUrl() {
    return `${config.rwApiBaseUrl.replace(/\/$/, '')}/api/users`;
  }

  function buildResolveUrl() {
    return `${config.rwApiBaseUrl.replace(/\/$/, '')}/api/users/resolve`;
  }

  function buildUsersByTelegramIdUrl(telegramId) {
    return `${config.rwApiBaseUrl.replace(/\/$/, '')}/api/users/by-telegram-id/${encodeURIComponent(telegramId)}`;
  }

  function buildTemporaryUserPayload({ username }) {
    const payload = {
      username,
      description: 'Auto-generated temporary subscription',
      status: 'ACTIVE',
      trafficLimitBytes: Math.round(
        config.tempSubscriptionTrafficLimitMb * BYTES_IN_MEGABYTE
      ),
      trafficLimitStrategy: 'NO_RESET',
      expireAt: getTomorrowIso()
    };

    if (config.rwSquadUuid) {
      payload.activeInternalSquads = [config.rwSquadUuid];
    }

    return payload;
  }

  async function createTemporarySubscription({ requestId }) {
    assertConfigured(requestId);

    const username = generateUsername();
    const payload = buildTemporaryUserPayload({ username });
    const rwUsersUrl = buildUsersUrl();

    log('info', 'Creating temporary subscription', {
      requestId,
      username,
      rwUsersUrl,
      expireAt: payload.expireAt,
      trafficLimitMb: config.tempSubscriptionTrafficLimitMb,
      hasSquad: Boolean(config.rwSquadUuid)
    });

    try {
      const response = await fetch(rwUsersUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${config.rwApiToken}`
        },
        body: JSON.stringify(payload)
      });
      const responseBody = await parseJsonResponse(response);

      if (!response.ok) {
        log('warn', 'RW API returned non-2xx response', {
          requestId,
          username,
          rwStatus: response.status,
          rwResponsePreview: stringifyForLog(responseBody)
        });

        throw createHttpError('RW API request failed', response.status, {
          clientResponse: {
            error: 'RW API request failed',
            rwStatus: response.status,
            rwResponse: responseBody,
            payloadSent: payload
          }
        });
      }

      const rwUser = getRwUserFromResponse(responseBody);

      return {
        username,
        payload,
        trafficLimitMb: config.tempSubscriptionTrafficLimitMb,
        rwStatus: response.status,
        rwUser,
        responseBody,
        subscriptionUrl: rwUser?.subscriptionUrl ?? responseBody?.subscriptionUrl ?? null
      };
    } catch (error) {
      if (error?.httpStatus) {
        throw error;
      }

      log('error', 'RW API fetch failed', {
        requestId,
        username,
        rwUsersUrl,
        ...buildErrorMeta(error)
      });

      throw createHttpError('Failed to reach RW API', 502, {
        clientResponse: {
          error: 'Failed to reach RW API',
          details: error instanceof Error ? error.message : 'Unknown error',
          payloadSent: payload
        }
      });
    }
  }

  async function resolveUserByUsername({ requestId, username }) {
    assertConfigured(requestId);

    const rwResolveUrl = buildResolveUrl();
    const response = await fetch(rwResolveUrl, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${config.rwApiToken}`
      },
      body: JSON.stringify({ username })
    });
    const responseBody = await parseJsonResponse(response);

    if (!response.ok) {
      log('warn', 'Failed to resolve RW user by username', {
        requestId,
        username,
        rwStatus: response.status,
        rwResponsePreview: stringifyForLog(responseBody)
      });

      throw createHttpError(`RW resolve failed with status ${response.status}`, response.status, {
        rwStatus: response.status,
        rwResponse: responseBody
      });
    }

    return getRwUserFromResponse(responseBody);
  }

  async function getUsersByTelegramId({ requestId, telegramId }) {
    assertConfigured(requestId);

    const response = await fetch(buildUsersByTelegramIdUrl(telegramId), {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${config.rwApiToken}`
      }
    });
    const responseBody = await parseJsonResponse(response);

    if (!response.ok) {
      log('warn', 'Failed to fetch RW users by Telegram ID', {
        requestId,
        telegramId,
        rwStatus: response.status,
        rwResponsePreview: stringifyForLog(responseBody)
      });

      throw createHttpError(
        `RW lookup by Telegram ID failed with status ${response.status}`,
        response.status,
        {
          rwStatus: response.status,
          rwResponse: responseBody
        }
      );
    }

    const users = Array.isArray(responseBody?.response) ? responseBody.response : [];
    return users;
  }

  async function updateUserWithTelegram({ requestId, rwUserUuid, rwUsername, telegramId }) {
    assertConfigured(requestId);

    const userUuid =
      rwUserUuid ||
      (await resolveUserByUsername({ requestId, username: rwUsername }))?.uuid;

    if (!userUuid) {
      throw createHttpError('Unable to resolve RW user UUID before update', 502);
    }

    const response = await fetch(buildUsersUrl(), {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${config.rwApiToken}`
      },
      body: JSON.stringify({
        uuid: userUuid,
        telegramId: telegramId.asNumber
      })
    });
    const responseBody = await parseJsonResponse(response);

    if (!response.ok) {
      log('warn', 'Failed to update RW user with Telegram identity', {
        requestId,
        rwUserUuid: userUuid,
        telegramId: telegramId.asString,
        rwStatus: response.status,
        rwResponsePreview: stringifyForLog(responseBody)
      });

      throw createHttpError(`RW update failed with status ${response.status}`, response.status, {
        rwStatus: response.status,
        rwResponse: responseBody
      });
    }

    return {
      rwUser: getRwUserFromResponse(responseBody),
      rwResponse: responseBody,
      rwUserUuid: userUuid
    };
  }

  return {
    createTemporarySubscription,
    getUsersByTelegramId,
    resolveUserByUsername,
    updateUserWithTelegram
  };
}
