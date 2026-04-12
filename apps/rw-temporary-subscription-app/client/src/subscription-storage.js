const SUBSCRIPTION_STORAGE_KEY = 'rw-temp-subscription';

export function getStoredSuccessResult() {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    const raw = window.localStorage.getItem(SUBSCRIPTION_STORAGE_KEY);

    if (!raw) {
      return null;
    }

    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === 'object'
      ? { type: 'success', data: parsed }
      : null;
  } catch {
    return null;
  }
}

export function storeSuccessResult(data) {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    window.localStorage.setItem(
      SUBSCRIPTION_STORAGE_KEY,
      JSON.stringify(data)
    );
  } catch {
    // Ignore storage errors and keep runtime state only.
  }
}
