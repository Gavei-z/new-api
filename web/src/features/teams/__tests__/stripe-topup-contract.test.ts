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
  createTeamStripeAmountPayload,
  createTeamStripeTopupPayload,
} from '../lib'

describe('team Stripe top-up target contract', () => {
  test('sends the allowlisted checkout method while the server derives the team', () => {
    const payload = createTeamStripeTopupPayload(50, 'wechat_pay')

    assert.deepEqual(payload, {
      amount: 50,
      payment_method: 'stripe',
      checkout_method: 'wechat_pay',
    })
    assert.equal('team_id' in payload, false)
    assert.equal('user_id' in payload, false)
  })

  test('keeps the regular team checkout explicit and backward-safe', () => {
    assert.deepEqual(createTeamStripeTopupPayload(2, 'standard'), {
      amount: 2,
      payment_method: 'stripe',
      checkout_method: 'standard',
    })
  })

  test('keeps standard and WeChat quote requests separated by method', () => {
    assert.deepEqual(createTeamStripeAmountPayload(2, 'standard'), {
      amount: 2,
      checkout_method: 'standard',
    })
    assert.deepEqual(createTeamStripeAmountPayload(2, 'wechat_pay'), {
      amount: 2,
      checkout_method: 'wechat_pay',
    })
  })
})
