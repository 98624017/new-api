/*
Copyright (C) 2025 QuantumNous

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

import { beforeAll, beforeEach, describe, expect, mock, test } from 'bun:test';

const post = mock(async () => ({ data: { success: true } }));

mock.module('../helpers', () => ({
  API: { post },
}));

let updateChannelStatus;

describe('classic channel status API', () => {
  beforeAll(async () => {
    ({ updateChannelStatus } = await import('./channel'));
  });

  beforeEach(() => {
    post.mockClear();
  });

  test.each([
    ['enable', 1],
    ['disable', 2],
  ])('uses the dedicated endpoint to %s a channel', async (_, status) => {
    const response = await updateChannelStatus(42, status);

    expect(response).toEqual({ data: { success: true } });
    expect(post).toHaveBeenCalledTimes(1);
    expect(post).toHaveBeenCalledWith('/api/channel/42/status', { status });
  });
});
