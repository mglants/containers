import 'dotenv/config';

import fs from 'fs';

import { createApp } from './app.js';
import { loadConfig } from './config.js';
import { log } from './logger.js';
import { createRemnawaveClient } from './remnawave-client.js';
import { createSubscriptionService } from './subscription-service.js';
import { createTempSubscriptionStore } from './temp-subscription-store.js';
import { createTelegramBot } from './telegram-bot.js';

const config = loadConfig();

const tempSubscriptionStore = createTempSubscriptionStore({
  filePath: config.tempSubscriptionLinksPath,
  retentionMs: config.tempSubscriptionRetentionMs,
  log
});

const remnawaveClient = createRemnawaveClient({
  config,
  log
});

const subscriptionService = createSubscriptionService({
  config,
  log,
  tempSubscriptionStore,
  remnawaveClient
});

const telegramBot = createTelegramBot({
  config,
  log,
  getTempSubscriptionRecord: tempSubscriptionStore.get,
  linkTemporarySubscriptionToTelegram:
    subscriptionService.linkTemporarySubscriptionToTelegram
});

const app = createApp({
  config,
  log,
  subscriptionService
});

app.listen(config.port, () => {
  log('info', 'Server started', {
    port: config.port,
    authEnabled: config.appAuthEnabled,
    rwApiBaseUrl: config.rwApiBaseUrl || null,
    hasRwApiToken: Boolean(config.rwApiToken),
    hasSquad: Boolean(config.rwSquadUuid),
    tempSubscriptionTrafficLimitMb: config.tempSubscriptionTrafficLimitMb,
    hasTelegramBotToken: Boolean(config.telegramBotToken),
    hasTelegramBotUsername: Boolean(config.telegramBotUsername),
    hasRegistrationBotUrl: Boolean(config.registrationBotUrl),
    tempSubscriptionStorePath: config.tempSubscriptionLinksPath,
    staticClientServed: fs.existsSync(config.clientIndexPath)
  });

  telegramBot.startPolling();
});
