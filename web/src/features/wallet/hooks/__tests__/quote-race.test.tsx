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
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

import { PAYMENT_TYPES, STRIPE_CHECKOUT_METHODS } from '../../constants'
import { createPaymentConfirmationSnapshot } from '../../lib/payment'
import type { AmountResponse, StripeCheckoutMethod } from '../../types'
import type { PaymentAmountCalculators } from '../use-payment'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'Node',
  'Element',
  'Event',
  'MutationObserver',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { usePayment } = await import('../use-payment')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type PaymentHookState = ReturnType<typeof usePayment>

interface PaymentHookHarnessProps {
  calculators: PaymentAmountCalculators
  onRender: (state: PaymentHookState) => void
}

function PaymentHookHarness(props: PaymentHookHarnessProps) {
  const payment = usePayment(props.calculators)
  props.onRender(payment)
  return <output data-slot='shared-payment-amount'>{payment.amount}</output>
}

function createDeferredAmountResponse() {
  let resolve!: (response: AmountResponse) => void
  const promise = new Promise<AmountResponse>((resolvePromise) => {
    resolve = resolvePromise
  })
  return { promise, resolve }
}

function weChatAmountResponse(
  data = '13.54',
  quoteVersion = 'boc-fx-2026-08-09-v1'
): AmountResponse {
  return {
    success: true,
    data,
    quote: {
      pay_amount_minor: Math.round(Number(data) * 100),
      currency: 'cny',
      exchange_rate: '6.7655',
      source: 'boc_spot_selling',
      pricing_date: '2026-08-09',
      source_published_at: 1_786_244_200,
      quote_version: quoteVersion,
    },
  }
}

async function renderPaymentHook(calculators: PaymentAmountCalculators) {
  let state: PaymentHookState | undefined
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <PaymentHookHarness
        calculators={calculators}
        onRender={(nextState) => {
          state = nextState
        }}
      />
    )
  })

  return {
    container,
    root,
    getState: () => {
      assert.ok(state)
      return state
    },
  }
}

function calculatorsWithStripe(
  stripe: PaymentAmountCalculators['stripe']
): PaymentAmountCalculators {
  return {
    regular: async () => ({ success: true, data: '1.00' }),
    stripe,
    waffo: async () => ({ success: true, data: '1.00' }),
    waffoPancake: async () => ({ success: true, data: '1.00' }),
  }
}

describe('payment quote race handling', () => {
  after(() => {
    domWindow.close()
  })

  test('late standard preview cannot overwrite a newer WeChat confirmation quote', async () => {
    const delayedStandard = createDeferredAmountResponse()
    const calculators = calculatorsWithStripe(async (request) => {
      const checkoutMethod = request.checkout_method as StripeCheckoutMethod
      if (checkoutMethod === STRIPE_CHECKOUT_METHODS.STANDARD) {
        return delayedStandard.promise
      }
      return weChatAmountResponse()
    })
    const rendered = await renderPaymentHook(calculators)

    let standardPreview!: Promise<number>
    let weChatQuoteResult!: Awaited<
      ReturnType<PaymentHookState['quotePaymentAmount']>
    >
    await act(async () => {
      standardPreview = rendered
        .getState()
        .calculatePaymentAmount(
          2,
          PAYMENT_TYPES.STRIPE,
          STRIPE_CHECKOUT_METHODS.STANDARD
        )
      weChatQuoteResult = await rendered
        .getState()
        .quotePaymentAmount(
          2,
          PAYMENT_TYPES.STRIPE,
          STRIPE_CHECKOUT_METHODS.WECHAT_PAY
        )
    })

    assert.equal(weChatQuoteResult.status, 'success')
    assert.ok(weChatQuoteResult.status === 'success')
    assert.equal(weChatQuoteResult.quote.amount, 13.54)
    assert.equal(rendered.getState().amount, 0)
    const confirmation = createPaymentConfirmationSnapshot(
      2,
      weChatQuoteResult.quote.amount,
      {
        name: 'WeChat Pay',
        type: PAYMENT_TYPES.STRIPE,
        checkout_method: STRIPE_CHECKOUT_METHODS.WECHAT_PAY,
      },
      null,
      weChatQuoteResult.quote.stripeFxQuote
    )

    await act(async () => {
      delayedStandard.resolve({ success: true, data: '2.00' })
      await standardPreview
    })

    assert.equal(rendered.getState().amount, 0)
    assert.equal(confirmation.creditAmount, 2)
    assert.equal(confirmation.quotedPaymentAmount, 13.54)
    assert.equal(confirmation.currencyCode, 'CNY')
    assert.equal(
      confirmation.stripeFxQuote?.quote_version,
      'boc-fx-2026-08-09-v1'
    )
    assert.equal(rendered.container.querySelector('output')?.textContent, '0')

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('only the latest background amount request can update the shared preview', async () => {
    const delayedTwoDollarQuote = createDeferredAmountResponse()
    const delayedFiveDollarQuote = createDeferredAmountResponse()
    const calculators = calculatorsWithStripe(async (request) => {
      return request.amount === 2
        ? delayedTwoDollarQuote.promise
        : delayedFiveDollarQuote.promise
    })
    const rendered = await renderPaymentHook(calculators)

    let firstRequest!: Promise<number>
    let secondRequest!: Promise<number>
    await act(async () => {
      firstRequest = rendered
        .getState()
        .calculatePaymentAmount(2, PAYMENT_TYPES.STRIPE)
      secondRequest = rendered
        .getState()
        .calculatePaymentAmount(5, PAYMENT_TYPES.STRIPE)
    })

    await act(async () => {
      delayedFiveDollarQuote.resolve({ success: true, data: '5.00' })
      await secondRequest
    })
    assert.equal(rendered.getState().amount, 5)

    await act(async () => {
      delayedTwoDollarQuote.resolve({ success: true, data: '2.00' })
      await firstRequest
    })
    assert.equal(rendered.getState().amount, 5)

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })

  test('a delayed checkout quote is marked superseded after a newer quote completes', async () => {
    const delayedWeChatQuote = createDeferredAmountResponse()
    const calculators = calculatorsWithStripe(async (request) => {
      if (request.amount === 2) return delayedWeChatQuote.promise
      return { success: true, data: '5.00' }
    })
    const rendered = await renderPaymentHook(calculators)

    let firstQuote!: ReturnType<PaymentHookState['quotePaymentAmount']>
    let latestQuote!: Awaited<
      ReturnType<PaymentHookState['quotePaymentAmount']>
    >
    await act(async () => {
      firstQuote = rendered
        .getState()
        .quotePaymentAmount(
          2,
          PAYMENT_TYPES.STRIPE,
          STRIPE_CHECKOUT_METHODS.WECHAT_PAY
        )
      latestQuote = await rendered
        .getState()
        .quotePaymentAmount(
          5,
          PAYMENT_TYPES.STRIPE,
          STRIPE_CHECKOUT_METHODS.STANDARD
        )
    })

    assert.equal(latestQuote.status, 'success')
    if (latestQuote.status === 'success') {
      assert.equal(latestQuote.quote.amount, 5)
    }

    let staleQuote!: Awaited<typeof firstQuote>
    await act(async () => {
      delayedWeChatQuote.resolve(weChatAmountResponse())
      staleQuote = await firstQuote
    })

    assert.equal(staleQuote.status, 'superseded')

    await act(async () => rendered.root.unmount())
    rendered.container.remove()
  })
})
