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

import { TASK_STATUS } from '../constants'

export interface TaskAudioClip {
  clip_id?: string
  id?: string
  title?: string
  tags?: string
  duration?: number
  audio_url: string
  image_url?: string
  image_large_url?: string
  metadata?: {
    tags?: string
    duration?: number
  }
}

function stringifyTaskPayload(value: unknown): string {
  try {
    const serialized = JSON.stringify(value, null, 2)
    if (serialized !== undefined) return serialized
  } catch {
    // Fall through to a readable scalar representation.
  }

  try {
    return String(value)
  } catch {
    return ''
  }
}

export function parseTaskPayload(value: unknown): unknown {
  if (typeof value !== 'string') return value

  const trimmed = value.trim()
  if (trimmed === '') return null

  try {
    return JSON.parse(trimmed)
  } catch {
    return value
  }
}

export function formatTaskPayload(value: unknown): string {
  if (value == null) return ''

  const parsed = parseTaskPayload(value)
  if (parsed == null) return ''
  if (typeof parsed === 'string') return parsed
  return stringifyTaskPayload(parsed)
}

export function getTaskAudioClips(value: unknown): TaskAudioClip[] {
  const parsed = parseTaskPayload(value)
  if (!Array.isArray(parsed)) return []

  return parsed.filter(
    (item): item is TaskAudioClip =>
      typeof item === 'object' &&
      item !== null &&
      typeof (item as { audio_url?: unknown }).audio_url === 'string' &&
      (item as { audio_url: string }).audio_url.trim() !== ''
  )
}

export function getTaskFailureReason(
  status: string,
  failReason?: string,
  resultUrl?: string
): string {
  const normalizedFailureReason = failReason?.trim() ?? ''
  const normalizedResultUrl = resultUrl?.trim() ?? ''

  if (
    status === TASK_STATUS.SUCCESS &&
    normalizedFailureReason === normalizedResultUrl
  ) {
    return ''
  }

  return normalizedFailureReason
}
