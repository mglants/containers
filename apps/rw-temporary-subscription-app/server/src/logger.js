export function log(level, message, meta = {}) {
  const entry = {
    timestamp: new Date().toISOString(),
    level,
    message,
    ...meta
  };
  const line = JSON.stringify(entry);

  if (level === 'error') {
    console.error(line);
    return;
  }

  if (level === 'warn') {
    console.warn(line);
    return;
  }

  console.log(line);
}

export function buildErrorMeta(error) {
  if (!(error instanceof Error)) {
    return { error: String(error) };
  }

  return {
    errorName: error.name,
    errorMessage: error.message,
    errorStack: error.stack
  };
}

export function stringifyForLog(value, maxLength = 1500) {
  if (value == null) {
    return null;
  }

  try {
    const serialized = typeof value === 'string' ? value : JSON.stringify(value);
    if (serialized.length <= maxLength) {
      return serialized;
    }

    return `${serialized.slice(0, maxLength)}...<truncated>`;
  } catch {
    return String(value);
  }
}
