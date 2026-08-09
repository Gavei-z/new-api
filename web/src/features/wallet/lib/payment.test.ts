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

import { PAYMENT_TYPES } from '../constants'
import {
  createPaymentConfirmationSnapshot,
  dispatchSelectedPayment,
  dispatchPaymentConfirmation,
  isStripePayment,
  isWaffoPayment,
  isWaffoPancakePayment,
  createStripeTopupPayload,
  getPaymentLoadingKey,
} from './payment'

describe('payment type classification', () => {
  test('keeps Waffo and Waffo Pancake on their dedicated flows', () => {
    assert.equal(isWaffoPayment(PAYMENT_TYPES.WAFFO), true)
    assert.equal(isWaffoPayment(PAYMENT_TYPES.WAFFO_PANCAKE), false)
    assert.equal(isWaffoPancakePayment(PAYMENT_TYPES.WAFFO_PANCAKE), true)
    assert.equal(isWaffoPancakePayment(PAYMENT_TYPES.WAFFO), false)
    assert.equal(isStripePayment(PAYMENT_TYPES.STRIPE), true)
  })
})

describe('Stripe checkout variants', () => {
  test('keeps WeChat Pay on the Stripe accounting provider', () => {
    const method = {
      name: 'WeChat Pay',
      type: PAYMENT_TYPES.STRIPE,
      checkout_method: 'wechat_pay' as const,
    }

    assert.equal(getPaymentLoadingKey(method), 'stripe:wechat_pay')
    assert.deepEqual(
      createStripeTopupPayload(5, 'wechat_pay', 'boc-fx-2026-08-09-v1'),
      {
        amount: 5,
        payment_method: 'stripe',
        checkout_method: 'wechat_pay',
        quote_version: 'boc-fx-2026-08-09-v1',
      }
    )
  })

  test('passes the selected Stripe checkout variant through confirmation', async () => {
    let checkoutMethod: string | undefined
    const success = await dispatchSelectedPayment(
      {
        name: 'WeChat Pay',
        type: PAYMENT_TYPES.STRIPE,
        checkout_method: 'wechat_pay',
      },
      5,
      null,
      {
        regular: async (_amount, _paymentType, selectedCheckoutMethod) => {
          checkoutMethod = selectedCheckoutMethod
          return true
        },
        waffo: async () => false,
        waffoPancake: async () => false,
      }
    )

    assert.equal(success, true)
    assert.equal(checkoutMethod, 'wechat_pay')
  })

  test('keeps display and submission bound to the clicked confirmation snapshot', async () => {
    const snapshot = createPaymentConfirmationSnapshot(
      2,
      13.54,
      {
        name: 'WeChat Pay',
        type: PAYMENT_TYPES.STRIPE,
        checkout_method: 'wechat_pay',
      },
      null,
      {
        pay_amount_minor: 1354,
        currency: 'CNY',
        exchange_rate: '6.7655',
        source: 'boc_spot_selling',
        pricing_date: '2026-08-09',
        source_published_at: 1_786_244_200,
        quote_version: 'boc-fx-2026-08-09-v1',
      }
    )
    const formAmountAfterClick = 5
    let submittedAmount = 0
    let submittedCheckoutMethod: string | undefined
    let submittedQuoteVersion: string | undefined

    const success = await dispatchPaymentConfirmation(snapshot, {
      regular: async (amount, _type, checkoutMethod, quoteVersion) => {
        submittedAmount = amount
        submittedCheckoutMethod = checkoutMethod
        submittedQuoteVersion = quoteVersion
        return true
      },
      waffo: async () => false,
      waffoPancake: async () => false,
    })

    assert.equal(formAmountAfterClick, 5)
    assert.equal(snapshot.creditAmount, 2)
    assert.equal(snapshot.quotedPaymentAmount, 13.54)
    assert.equal(snapshot.currencyCode, 'CNY')
    assert.equal(snapshot.checkoutMethod, 'wechat_pay')
    assert.deepEqual(snapshot.stripeFxQuote, {
      pay_amount_minor: 1354,
      currency: 'CNY',
      exchange_rate: '6.7655',
      source: 'boc_spot_selling',
      pricing_date: '2026-08-09',
      source_published_at: 1_786_244_200,
      quote_version: 'boc-fx-2026-08-09-v1',
    })
    assert.equal(submittedAmount, 2)
    assert.equal(submittedCheckoutMethod, 'wechat_pay')
    assert.equal(submittedQuoteVersion, 'boc-fx-2026-08-09-v1')
    assert.equal(success, true)
    assert.equal(Object.isFrozen(snapshot), true)
    assert.equal(Object.isFrozen(snapshot.paymentMethod), true)
    assert.equal(Object.isFrozen(snapshot.stripeFxQuote), true)
  })
})

describe('payment dispatch', () => {
  test('keeps the selected Waffo method index through confirmation', async () => {
    const calls: string[] = []
    const success = await dispatchSelectedPayment(
      { name: 'Waffo Card', type: PAYMENT_TYPES.WAFFO },
      120,
      3,
      {
        regular: async () => {
          calls.push('regular')
          return false
        },
        waffo: async (amount, index) => {
          calls.push(`waffo:${amount}:${index}`)
          return true
        },
        waffoPancake: async () => {
          calls.push('pancake')
          return false
        },
      }
    )

    assert.equal(success, true)
    assert.deepEqual(calls, ['waffo:120:3'])
  })

  test('does not create a Waffo order without a selected method index', async () => {
    let called = false
    const success = await dispatchSelectedPayment(
      { name: 'Waffo Card', type: PAYMENT_TYPES.WAFFO },
      120,
      null,
      {
        regular: async () => false,
        waffo: async () => {
          called = true
          return true
        },
        waffoPancake: async () => false,
      }
    )

    assert.equal(success, false)
    assert.equal(called, false)
  })

  test('does not dispatch a fractional top-up amount', async () => {
    const calls: string[] = []
    const success = await dispatchSelectedPayment(
      { name: 'Bank transfer', type: 'custom' },
      10.5,
      null,
      {
        regular: async () => {
          calls.push('regular')
          return true
        },
        waffo: async () => {
          calls.push('waffo')
          return true
        },
        waffoPancake: async () => {
          calls.push('pancake')
          return true
        },
      }
    )

    assert.equal(success, false)
    assert.deepEqual(calls, [])
  })
})
