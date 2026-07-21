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
import { describe, test } from 'node:test'

import {
  formatTaskPayload,
  getTaskAudioClips,
  getTaskFailureReason,
  parseTaskPayload,
} from './task-details'

describe('task detail payload formatting', () => {
  test('formats decoded objects, arrays, and scalars as readable JSON', () => {
    assert.equal(
      formatTaskPayload({ status: 'SUCCESS', progress: 100 }),
      '{\n  "status": "SUCCESS",\n  "progress": 100\n}'
    )
    assert.equal(formatTaskPayload(['a', 2]), '[\n  "a",\n  2\n]')
    assert.equal(formatTaskPayload(false), 'false')
    assert.equal(formatTaskPayload(0), '0')
  })

  test('pretty prints JSON strings and preserves ordinary strings', () => {
    assert.deepEqual(parseTaskPayload('{"id":"upstream-1"}'), {
      id: 'upstream-1',
    })
    assert.equal(
      formatTaskPayload('{"id":"upstream-1"}'),
      '{\n  "id": "upstream-1"\n}'
    )
    assert.equal(
      formatTaskPayload('upstream unavailable'),
      'upstream unavailable'
    )
  })

  test('treats nullish and whitespace-only payloads as empty', () => {
    assert.equal(formatTaskPayload(undefined), '')
    assert.equal(formatTaskPayload(null), '')
    assert.equal(formatTaskPayload('   '), '')
  })

  test('falls back safely when a decoded value cannot be serialized', () => {
    const cyclic: Record<string, unknown> = {}
    cyclic.self = cyclic

    assert.equal(formatTaskPayload(cyclic), '[object Object]')
    assert.equal(formatTaskPayload(10n), '10')
  })

  test('extracts playable audio clips from decoded and encoded arrays', () => {
    const clips = [
      { id: 'clip-1', audio_url: 'https://example.com/audio.mp3' },
      { id: 'clip-2', audio_url: '' },
      { id: 'clip-3' },
      null,
    ]

    assert.deepEqual(getTaskAudioClips(clips), [clips[0]])
    assert.deepEqual(getTaskAudioClips(JSON.stringify(clips)), [clips[0]])
    assert.deepEqual(getTaskAudioClips({ audio_url: 'not-an-array' }), [])
  })

  test('does not treat a legacy successful result URL as a failure', () => {
    const legacyResultUrl = 'https://example.com/result.mp4'

    assert.equal(
      getTaskFailureReason('SUCCESS', legacyResultUrl, legacyResultUrl),
      ''
    )
    assert.equal(
      getTaskFailureReason(
        'SUCCESS',
        `  ${legacyResultUrl}  `,
        legacyResultUrl
      ),
      ''
    )
    assert.equal(
      getTaskFailureReason('FAILURE', 'upstream failed', legacyResultUrl),
      'upstream failed'
    )
    assert.equal(
      getTaskFailureReason('FAILURE', legacyResultUrl, legacyResultUrl),
      legacyResultUrl
    )
    assert.equal(
      getTaskFailureReason(
        'SUCCESS',
        'partial upstream warning',
        legacyResultUrl
      ),
      'partial upstream warning'
    )
  })
})
