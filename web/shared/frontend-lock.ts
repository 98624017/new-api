/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

export const FRONTEND_LOCK_STORAGE_KEY = "new_api_frontend_lock_unlocked";
export const FRONTEND_LOCK_UNLOCK_TTL_MS = 30 * 24 * 60 * 60 * 1000;

export interface FrontendLockStorage {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
  removeItem(key: string): void;
}

interface FrontendLockCache {
  expiresAt: number;
  passwordFingerprint: string;
}

export function getFrontendLockPassword(injectedPassword: unknown, fallbackPassword = ""): string {
  return typeof injectedPassword === "string" ? injectedPassword : fallbackPassword;
}

export function isFrontendLockEnabled(password: string): boolean {
  return password.trim() !== "";
}

export function getFrontendLockPasswordFingerprint(password: string): string {
  let hash = 2166136261;

  for (let index = 0; index < password.length; index += 1) {
    hash ^= password.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }

  return `${password.length}:${(hash >>> 0).toString(16)}`;
}

export function getBrowserFrontendLockStorage(): FrontendLockStorage | null {
  if (typeof window === "undefined") return null;

  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

function clearFrontendLockCache(storage: FrontendLockStorage | null): void {
  try {
    storage?.removeItem(FRONTEND_LOCK_STORAGE_KEY);
  } catch {
    // 浏览器禁用 localStorage 时，当前页面仍可临时解锁。
  }
}

function parseFrontendLockCache(rawCache: string): FrontendLockCache | null {
  try {
    const cache: unknown = JSON.parse(rawCache);
    if (!cache || typeof cache !== "object") return null;

    const value = cache as Record<string, unknown>;
    if (typeof value.expiresAt !== "number" || typeof value.passwordFingerprint !== "string") {
      return null;
    }

    return {
      expiresAt: value.expiresAt,
      passwordFingerprint: value.passwordFingerprint,
    };
  } catch {
    return null;
  }
}

export function isFrontendLockUnlocked(
  password: string,
  storage = getBrowserFrontendLockStorage(),
  now = Date.now(),
): boolean {
  if (!isFrontendLockEnabled(password)) return true;
  if (!storage) return false;

  let rawCache: string | null;
  try {
    rawCache = storage.getItem(FRONTEND_LOCK_STORAGE_KEY);
  } catch {
    return false;
  }

  if (!rawCache) return false;

  const cache = parseFrontendLockCache(rawCache);
  if (
    !cache ||
    cache.expiresAt <= now ||
    cache.passwordFingerprint !== getFrontendLockPasswordFingerprint(password)
  ) {
    clearFrontendLockCache(storage);
    return false;
  }

  return true;
}

export function unlockFrontendSession(
  password: string,
  storage = getBrowserFrontendLockStorage(),
  now = Date.now(),
): void {
  if (!isFrontendLockEnabled(password) || !storage) return;

  try {
    storage.setItem(
      FRONTEND_LOCK_STORAGE_KEY,
      JSON.stringify({
        expiresAt: now + FRONTEND_LOCK_UNLOCK_TTL_MS,
        passwordFingerprint: getFrontendLockPasswordFingerprint(password),
      }),
    );
  } catch {
    // 写入失败时仅解锁当前 React 会话，下次刷新会再次要求密码。
  }
}

export function verifyFrontendLockPassword(input: string, password: string): boolean {
  return input === password;
}
