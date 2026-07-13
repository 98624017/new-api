const FRONTEND_LOCK_STORAGE_KEY = 'new_api_frontend_lock_unlocked';
const FRONTEND_LOCK_UNLOCK_TTL_MS = 2592000 * 1000;

export function getFrontendLockPassword() {
  if (
    typeof window !== 'undefined' &&
    typeof window.__FRONTEND_LOCK_PASSWORD__ === 'string'
  ) {
    return window.__FRONTEND_LOCK_PASSWORD__;
  }
  return import.meta.env.VITE_FRONTEND_LOCK_PASSWORD || '';
}

export function isFrontendLockEnabled() {
  return getFrontendLockPassword().trim() !== '';
}

function getFrontendLockStorage() {
  if (typeof window === 'undefined') {
    return null;
  }

  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

function getFrontendLockPasswordFingerprint() {
  const password = getFrontendLockPassword();
  let hash = 2166136261;

  for (let i = 0; i < password.length; i += 1) {
    hash ^= password.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }

  return `${password.length}:${(hash >>> 0).toString(16)}`;
}

function clearFrontendLockCache(storage) {
  try {
    storage?.removeItem(FRONTEND_LOCK_STORAGE_KEY);
  } catch {
    // 忽略浏览器禁用 localStorage 的场景。
  }
}

export function isFrontendLockUnlocked() {
  if (!isFrontendLockEnabled()) {
    return true;
  }

  const storage = getFrontendLockStorage();
  if (!storage) {
    return false;
  }

  let rawCache = '';
  try {
    rawCache = storage.getItem(FRONTEND_LOCK_STORAGE_KEY);
  } catch {
    return false;
  }

  if (!rawCache) {
    return false;
  }

  let cache;
  try {
    cache = JSON.parse(rawCache);
  } catch {
    clearFrontendLockCache(storage);
    return false;
  }

  if (
    !cache ||
    typeof cache.expiresAt !== 'number' ||
    cache.expiresAt <= Date.now() ||
    cache.passwordFingerprint !== getFrontendLockPasswordFingerprint()
  ) {
    clearFrontendLockCache(storage);
    return false;
  }

  return true;
}

export function unlockFrontendSession() {
  if (!isFrontendLockEnabled()) {
    return;
  }

  const storage = getFrontendLockStorage();
  if (!storage) {
    return;
  }

  try {
    storage.setItem(
      FRONTEND_LOCK_STORAGE_KEY,
      JSON.stringify({
        expiresAt: Date.now() + FRONTEND_LOCK_UNLOCK_TTL_MS,
        passwordFingerprint: getFrontendLockPasswordFingerprint(),
      }),
    );
  } catch {
    // 写入失败时下次加载重新要求输入密码。
  }
}

export function verifyFrontendLockPassword(input) {
  return input === getFrontendLockPassword();
}
