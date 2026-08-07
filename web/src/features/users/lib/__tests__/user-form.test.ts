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

import { quotaUnitsToDollars } from '@/lib/format'

import type { User } from '../../types'
import { transformUserToFormDefaults } from '../user-form'

function userWithBalance(overrides: Partial<User> = {}): User {
  return {
    id: 1,
    username: 'balance-user',
    display_name: 'Balance User',
    quota: 10,
    used_quota: 5,
    request_count: 1,
    group: 'default',
    status: 1,
    role: 1,
    ...overrides,
  }
}

describe('administrator user balance form', () => {
  test('uses total quota when a prepaid reserve is present', () => {
    const user = userWithBalance({
      quota: 10,
      reserve_quota: 1_000,
      total_quota: 1_010,
    })

    const defaults = transformUserToFormDefaults(user)

    assert.equal(defaults.quota_dollars, quotaUnitsToDollars(1_010))
  })

  test('falls back to the legacy active quota for older API responses', () => {
    const user = userWithBalance({ quota: 25 })

    const defaults = transformUserToFormDefaults(user)

    assert.equal(defaults.quota_dollars, quotaUnitsToDollars(25))
  })
})
