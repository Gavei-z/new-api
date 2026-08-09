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

import { normalizeStripeFxQuote, parsePaymentAmountQuote } from '../stripe-fx'

const rawQuote = {
  pay_amount_minor: 1354,
  currency: 'cny',
  exchange_rate: '6.7655',
  source: 'boc_spot_selling',
  pricing_date: '2026-08-09',
  source_published_at: 1_786_244_200,
  quote_version: 'boc-fx-2026-08-09-v1',
}

describe('Stripe foreign-exchange quote parsing', () => {
  test('normalizes a valid server quote without converting its decimal rate', () => {
    const quote = normalizeStripeFxQuote(rawQuote)

    assert.equal(quote?.currency, 'CNY')
    assert.equal(quote?.exchange_rate, '6.7655')
    assert.equal(quote?.pay_amount_minor, 1354)
    assert.equal(quote?.source_published_at, 1_786_244_200)
    assert.equal(Object.isFrozen(quote), true)
  })

  test('keeps the standard amount response compatible without FX metadata', () => {
    const quote = parsePaymentAmountQuote(
      { message: 'success', data: '2.00' },
      'standard'
    )

    assert.deepEqual(quote, { amount: 2 })
  })

  test('rejects a WeChat quote when display amount and minor amount disagree', () => {
    const quote = parsePaymentAmountQuote(
      { message: 'success', data: '13.55', quote: rawQuote },
      'wechat_pay'
    )

    assert.equal(quote, null)
  })

  test('rejects malformed monetary and pricing metadata', () => {
    assert.equal(
      normalizeStripeFxQuote({
        ...rawQuote,
        exchange_rate: 'NaN',
      }),
      null
    )
    assert.equal(
      normalizeStripeFxQuote({
        ...rawQuote,
        pricing_date: '09/08/2026',
      }),
      null
    )
    assert.equal(
      normalizeStripeFxQuote({
        ...rawQuote,
        pay_amount_minor: 13.54,
      }),
      null
    )
  })
})
