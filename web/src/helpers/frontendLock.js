const FRONTEND_LOCK_SESSION_KEY = 'new_api_frontend_lock_unlocked';

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

export function isFrontendLockUnlocked() {
  if (!isFrontendLockEnabled() || typeof sessionStorage === 'undefined') {
    return !isFrontendLockEnabled();
  }
  return sessionStorage.getItem(FRONTEND_LOCK_SESSION_KEY) === '1';
}

export function unlockFrontendSession() {
  if (typeof sessionStorage !== 'undefined') {
    sessionStorage.setItem(FRONTEND_LOCK_SESSION_KEY, '1');
  }
}

export function verifyFrontendLockPassword(input) {
  return input === getFrontendLockPassword();
}
