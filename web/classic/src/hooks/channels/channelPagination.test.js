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

import { describe, expect, test } from 'bun:test';
import { getValidChannelPage } from './channelPagination';

describe('classic channel pagination', () => {
  test.each([
    ['keeps the requested page when it remains valid', 3, 21, 10, 3],
    ['uses the new last page after the total shrinks', 2, 10, 10, 1],
    ['uses the first page for an empty result', 4, 0, 10, 1],
  ])('%s', (_, requestedPage, total, pageSize, expectedPage) => {
    expect(getValidChannelPage(requestedPage, total, pageSize)).toBe(
      expectedPage,
    );
  });
});
