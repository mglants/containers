import fs from 'fs';
import path from 'path';

import { buildErrorMeta } from './logger.js';

export function createTempSubscriptionStore({ filePath, retentionMs, log }) {
  function ensureStore() {
    const storeDir = path.dirname(filePath);
    fs.mkdirSync(storeDir, { recursive: true });

    if (!fs.existsSync(filePath)) {
      fs.writeFileSync(filePath, JSON.stringify({ records: {} }, null, 2));
    }
  }

  function isStaleRecord(record) {
    const now = Date.now();
    const expireAtMs = Date.parse(record?.expireAt ?? '');
    const linkedAtMs = Date.parse(record?.linkedAt ?? '');
    const createdAtMs = Date.parse(record?.createdAt ?? '');

    if (Number.isFinite(expireAtMs) && expireAtMs + retentionMs < now) {
      return true;
    }

    if (Number.isFinite(linkedAtMs) && linkedAtMs + retentionMs < now) {
      return true;
    }

    if (Number.isFinite(createdAtMs) && createdAtMs + retentionMs < now) {
      return true;
    }

    return false;
  }

  function writeStore(store) {
    ensureStore();
    fs.writeFileSync(filePath, JSON.stringify(store, null, 2));
  }

  function readStore() {
    ensureStore();

    try {
      const raw = fs.readFileSync(filePath, 'utf8');
      const parsed = raw ? JSON.parse(raw) : {};
      const records = parsed?.records && typeof parsed.records === 'object' ? parsed.records : {};
      let hasChanges = false;

      for (const [key, value] of Object.entries(records)) {
        if (isStaleRecord(value)) {
          delete records[key];
          hasChanges = true;
        }
      }

      if (hasChanges) {
        writeStore({ records });
      }

      return { records };
    } catch (error) {
      log('warn', 'Failed to read temp subscription store, recreating it', {
        tempStorePath: filePath,
        ...buildErrorMeta(error)
      });

      const nextStore = { records: {} };
      writeStore(nextStore);
      return nextStore;
    }
  }

  function set(record) {
    const store = readStore();
    store.records[record.tempSubscriptionId] = record;
    writeStore(store);
    return record;
  }

  function get(tempSubscriptionId) {
    const store = readStore();
    return store.records[tempSubscriptionId] ?? null;
  }

  function update(tempSubscriptionId, updater) {
    const store = readStore();
    const currentRecord = store.records[tempSubscriptionId];

    if (!currentRecord) {
      return null;
    }

    const nextRecord = updater(currentRecord);
    store.records[tempSubscriptionId] = nextRecord;
    writeStore(store);
    return nextRecord;
  }

  return {
    set,
    get,
    update
  };
}
