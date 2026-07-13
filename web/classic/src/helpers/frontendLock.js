import {
  getFrontendLockPassword as resolveFrontendLockPassword,
  isFrontendLockEnabled as resolveFrontendLockEnabled,
  isFrontendLockUnlocked as resolveFrontendLockUnlocked,
  unlockFrontendSession as persistFrontendUnlock,
  verifyFrontendLockPassword as compareFrontendLockPassword,
} from '../../../shared/frontend-lock';

export function getFrontendLockPassword() {
  return resolveFrontendLockPassword(
    typeof window !== 'undefined'
      ? window.__FRONTEND_LOCK_PASSWORD__
      : undefined,
    import.meta.env.VITE_FRONTEND_LOCK_PASSWORD || '',
  );
}

export function isFrontendLockEnabled() {
  return resolveFrontendLockEnabled(getFrontendLockPassword());
}

export function isFrontendLockUnlocked() {
  return resolveFrontendLockUnlocked(getFrontendLockPassword());
}

export function unlockFrontendSession() {
  persistFrontendUnlock(getFrontendLockPassword());
}

export function verifyFrontendLockPassword(input) {
  return compareFrontendLockPassword(input, getFrontendLockPassword());
}
