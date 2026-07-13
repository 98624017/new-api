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

import assert from 'node:assert/strict'
import { afterEach, beforeEach, describe, test } from 'node:test'

import { FRONTEND_LOCK_STORAGE_KEY } from '../../../shared/frontend-lock'
import { initializeFrontendCache } from './frontend-cache'

class MemoryLocalStorage implements Storage {
  private readonly values = new Map<string, string>()

  get length(): number {
    return this.values.size
  }

  clear(): void {
    this.values.clear()
  }

  getItem(key: string): string | null {
    return this.values.get(key) ?? null
  }

  key(index: number): string | null {
    return [...this.values.keys()][index] ?? null
  }

  removeItem(key: string): void {
    this.values.delete(key)
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value)
  }
}

const originalWindowDescriptor = Object.getOwnPropertyDescriptor(
  globalThis,
  'window'
)

describe('default frontend cache initialization', () => {
  let storage: MemoryLocalStorage

  beforeEach(() => {
    storage = new MemoryLocalStorage()
    Object.defineProperty(globalThis, 'window', {
      configurable: true,
      value: { localStorage: storage },
    })
  })

  afterEach(() => {
    if (originalWindowDescriptor) {
      Object.defineProperty(globalThis, 'window', originalWindowDescriptor)
      return
    }
    Reflect.deleteProperty(globalThis, 'window')
  })

  test('preserves the shared frontend lock while clearing stale UI cache', () => {
    storage.setItem(FRONTEND_LOCK_STORAGE_KEY, 'classic-unlock-cache')
    storage.setItem('legacy-ui-cache', 'stale')

    initializeFrontendCache()

    assert.equal(
      storage.getItem(FRONTEND_LOCK_STORAGE_KEY),
      'classic-unlock-cache'
    )
    assert.equal(storage.getItem('legacy-ui-cache'), null)
    assert.equal(storage.getItem('newapi:default:cache-version'), 'default-v1')
  })
})
