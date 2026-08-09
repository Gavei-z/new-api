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

import { PAYMENT_TYPES, STRIPE_CHECKOUT_METHODS } from '../constants'
import { requestPaymentAmount } from './use-payment'

describe('payment amount routing', () => {
  test('uses the dedicated Waffo amount calculator', async () => {
    const calls: string[] = []
    const amount = await requestPaymentAmount(
      120,
      PAYMENT_TYPES.WAFFO,
      STRIPE_CHECKOUT_METHODS.STANDARD,
      {
        regular: async () => {
          calls.push('regular')
          return { success: true, data: '1' }
        },
        stripe: async () => {
          calls.push('stripe')
          return { success: true, data: '2' }
        },
        waffo: async (request) => {
          calls.push(`waffo:${request.amount}`)
          return { success: true, data: '18.75' }
        },
        waffoPancake: async () => {
          calls.push('pancake')
          return { success: true, data: '4' }
        },
      }
    )

    assert.equal(amount, 18.75)
    assert.deepEqual(calls, ['waffo:120'])
  })

  test('does not request a quote for a fractional top-up amount', async () => {
    const calls: string[] = []
    const calculators = {
      regular: async () => {
        calls.push('regular')
        return { success: true, data: '1' }
      },
      stripe: async () => {
        calls.push('stripe')
        return { success: true, data: '2' }
      },
      waffo: async () => {
        calls.push('waffo')
        return { success: true, data: '3' }
      },
      waffoPancake: async () => {
        calls.push('pancake')
        return { success: true, data: '4' }
      },
    }

    const amount = await requestPaymentAmount(
      10.5,
      PAYMENT_TYPES.WAFFO,
      STRIPE_CHECKOUT_METHODS.STANDARD,
      calculators
    )

    assert.equal(amount, 0)
    assert.deepEqual(calls, [])
  })

  test('sends the selected Stripe checkout method with each quote request', async () => {
    const stripeRequests: Array<{
      amount: number
      checkout_method?: string
    }> = []
    const calculators = {
      regular: async () => ({ success: true, data: '1' }),
      stripe: async (request: { amount: number; checkout_method?: string }) => {
        stripeRequests.push(request)
        return {
          success: true,
          data:
            request.checkout_method === STRIPE_CHECKOUT_METHODS.WECHAT_PAY
              ? '14.40'
              : '2.00',
        }
      },
      waffo: async () => ({ success: true, data: '1' }),
      waffoPancake: async () => ({ success: true, data: '1' }),
    }

    const standardAmount = await requestPaymentAmount(
      2,
      PAYMENT_TYPES.STRIPE,
      STRIPE_CHECKOUT_METHODS.STANDARD,
      calculators
    )
    const weChatAmount = await requestPaymentAmount(
      2,
      PAYMENT_TYPES.STRIPE,
      STRIPE_CHECKOUT_METHODS.WECHAT_PAY,
      calculators
    )

    assert.equal(standardAmount, 2)
    assert.equal(weChatAmount, 14.4)
    assert.deepEqual(stripeRequests, [
      { amount: 2, checkout_method: 'standard' },
      { amount: 2, checkout_method: 'wechat_pay' },
    ])
  })
})
