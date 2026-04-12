import crypto from 'crypto';
import fs from 'fs';

import cors from 'cors';
import express from 'express';

import { createAuthModule } from './auth.js';
import { buildErrorMeta } from './logger.js';

function sendUnexpectedError(log, req, res, error, message) {
  log('error', message, {
    requestId: req.requestId,
    path: req.path,
    ...buildErrorMeta(error)
  });

  return res.status(500).json({
    error: 'Internal server error',
    details: error instanceof Error ? error.message : 'Unknown error'
  });
}

export function createApp({ config, log, subscriptionService }) {
  const app = express();
  const { authRouter, requirePageAuth } = createAuthModule({ config });

  app.use((req, res, next) => {
    const requestId = crypto.randomUUID();
    const startedAt = Date.now();

    req.requestId = requestId;
    res.setHeader('X-Request-Id', requestId);

    log('info', 'Request received', {
      requestId,
      method: req.method,
      path: req.path,
      ip: req.ip
    });

    res.on('finish', () => {
      log('info', 'Request completed', {
        requestId,
        method: req.method,
        path: req.path,
        statusCode: res.statusCode,
        durationMs: Date.now() - startedAt
      });
    });

    next();
  });

  app.use(
    cors({
      origin: true,
      credentials: true
    })
  );
  app.use(requirePageAuth);
  app.use(express.json());
  app.use(express.urlencoded({ extended: false }));

  app.use('/auth', authRouter);

  app.get('/config.js', (_req, res) => {
    res.setHeader('Cache-Control', 'no-store');
    res.type('application/javascript').send(
      `window.__RW_RUNTIME_CONFIG = ${JSON.stringify(config.runtimeClientConfig)};`
    );
  });

  app.get('/api/health', (_req, res) => {
    res.json({ status: 'ok' });
  });

  app.post('/api/subscriptions/temporary', async (req, res) => {
    try {
      const result = await subscriptionService.createTemporarySubscription({
        requestId: req.requestId,
        preferredLocale: req.body?.locale
      });

      return res.status(201).json(result);
    } catch (error) {
      if (error?.clientResponse && error?.httpStatus) {
        return res.status(error.httpStatus).json(error.clientResponse);
      }

      return sendUnexpectedError(
        log,
        req,
        res,
        error,
        'Unexpected failure while creating temporary subscription'
      );
    }
  });

  app.use('/api', (_req, res) => {
    res.status(404).json({ error: 'Not found' });
  });

  if (fs.existsSync(config.clientIndexPath)) {
    app.use(express.static(config.clientDistPath));

    app.get('*', (_req, res) => {
      res.sendFile(config.clientIndexPath);
    });
  }

  return app;
}
