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

import type { TopupInfo } from '../../types'
import {
  DEFAULT_STRIPE_TOPUP_MAX,
  DEFAULT_STRIPE_TOPUP_MIN,
  getStripeTopupBounds,
  getStripeTopupPresets,
  validateStripeTopupAmount,
} from '../stripe-topup'

const stripeTopupInfo = {
  enable_online_topup: false,
  enable_stripe_topup: true,
  pay_methods: [{ name: 'Stripe', type: 'stripe' }],
  min_topup: 2,
  stripe_min_topup: 2,
  stripe_max_topup: 50000,
  amount_options: [2, 5, 10, 50, 200, 500],
  discount: {},
} satisfies TopupInfo

describe('Stripe USD top-up validation', () => {
  test('accepts the inclusive USD boundaries', () => {
    const bounds = getStripeTopupBounds(stripeTopupInfo)

    assert.equal(validateStripeTopupAmount(2, bounds), null)
    assert.equal(validateStripeTopupAmount(50000, bounds), null)
  })

  test('rejects amounts immediately outside the USD boundaries', () => {
    const bounds = getStripeTopupBounds(stripeTopupInfo)

    assert.equal(validateStripeTopupAmount(1, bounds), 'minimum')
    assert.equal(validateStripeTopupAmount(50001, bounds), 'maximum')
  })

  test('rejects fractional amounts because checkout quantities are whole USD', () => {
    const bounds = getStripeTopupBounds(stripeTopupInfo)

    assert.equal(validateStripeTopupAmount(2.5, bounds), 'integer')
  })

  test('uses backend presets and safe bounds, with deployment defaults as fallback', () => {
    assert.deepEqual(
      getStripeTopupPresets(stripeTopupInfo),
      [2, 5, 10, 50, 200, 500]
    )
    assert.deepEqual(getStripeTopupBounds(null), {
      min: DEFAULT_STRIPE_TOPUP_MIN,
      max: DEFAULT_STRIPE_TOPUP_MAX,
    })
    assert.deepEqual(
      getStripeTopupBounds({
        ...stripeTopupInfo,
        stripe_min_topup: 1,
        stripe_max_topup: 50001,
      }),
      {
        min: DEFAULT_STRIPE_TOPUP_MIN,
        max: DEFAULT_STRIPE_TOPUP_MAX,
      }
    )
  })
})
