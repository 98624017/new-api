import { describe, expect, test } from "bun:test";

import {
  FRONTEND_LOCK_STORAGE_KEY,
  FRONTEND_LOCK_UNLOCK_TTL_MS,
  getFrontendLockPassword,
  isFrontendLockUnlocked,
  unlockFrontendSession,
  verifyFrontendLockPassword,
  type FrontendLockStorage,
} from "./frontend-lock";

class MemoryStorage implements FrontendLockStorage {
  private readonly values = new Map<string, string>();

  getItem(key: string): string | null {
    return this.values.get(key) ?? null;
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }

  removeItem(key: string): void {
    this.values.delete(key);
  }
}

describe("frontend lock state", () => {
  test("prefers the injected password and allows an empty configuration", () => {
    expect(getFrontendLockPassword("server-password", "dev-password")).toBe("server-password");
    expect(getFrontendLockPassword(undefined, "dev-password")).toBe("dev-password");
    expect(isFrontendLockUnlocked("", new MemoryStorage(), 100)).toBe(true);
  });

  test("persists a valid unlock for thirty days", () => {
    const storage = new MemoryStorage();
    unlockFrontendSession("secret", storage, 1_000);

    expect(isFrontendLockUnlocked("secret", storage, 1_001)).toBe(true);
    expect(isFrontendLockUnlocked("secret", storage, 1_000 + FRONTEND_LOCK_UNLOCK_TTL_MS)).toBe(
      false,
    );
    expect(storage.getItem(FRONTEND_LOCK_STORAGE_KEY)).toBeNull();
  });

  test("invalidates the cache when the configured password changes", () => {
    const storage = new MemoryStorage();
    unlockFrontendSession("old-password", storage, 1_000);

    expect(isFrontendLockUnlocked("new-password", storage, 1_001)).toBe(false);
    expect(storage.getItem(FRONTEND_LOCK_STORAGE_KEY)).toBeNull();
  });

  test("continues without persistence when storage is unavailable", () => {
    const storage: FrontendLockStorage = {
      getItem: () => {
        throw new Error("blocked");
      },
      setItem: () => {
        throw new Error("blocked");
      },
      removeItem: () => {
        throw new Error("blocked");
      },
    };

    expect(() => unlockFrontendSession("secret", storage, 1_000)).not.toThrow();
    expect(isFrontendLockUnlocked("secret", storage, 1_001)).toBe(false);
    expect(verifyFrontendLockPassword("secret", "secret")).toBe(true);
  });
});
