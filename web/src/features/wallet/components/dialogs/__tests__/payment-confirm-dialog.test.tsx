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

import englishMessages from '../../../../../i18n/locales/en.json'
import type { PaymentMethod, StripeFxQuote } from '../../../types'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'Node',
  'Element',
  'Event',
  'MouseEvent',
  'MutationObserver',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}
Object.defineProperty(globalThis, 'requestAnimationFrame', {
  configurable: true,
  value: domWindow.requestAnimationFrame.bind(domWindow),
})
Object.defineProperty(globalThis, 'cancelAnimationFrame', {
  configurable: true,
  value: domWindow.cancelAnimationFrame.bind(domWindow),
})

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { PaymentConfirmDialog } = await import('../payment-confirm-dialog')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

async function renderDialog(
  paymentMethod: PaymentMethod,
  paymentAmount = 2,
  stripeFxQuote?: StripeFxQuote
) {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    resources: { en: englishMessages },
    interpolation: { escapeValue: false },
  })

  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  let currencyCode: 'USD' | 'CNY' | undefined
  if (paymentMethod.type === 'stripe') {
    currencyCode =
      paymentMethod.checkout_method === 'wechat_pay' ? 'CNY' : 'USD'
  }

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <PaymentConfirmDialog
          open
          onOpenChange={() => undefined}
          onConfirm={() => undefined}
          topupAmount={2}
          paymentAmount={paymentAmount}
          paymentMethod={paymentMethod}
          calculating={false}
          processing={false}
          currencyCode={currencyCode}
          stripeCheckoutMethod={paymentMethod.checkout_method}
          stripeFxQuote={stripeFxQuote}
        />
      </I18nextProvider>
    )
  })

  return { container, root }
}

async function unmountDialog(
  rendered: Awaited<ReturnType<typeof renderDialog>>
) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

describe('payment confirmation provider details', () => {
  after(() => {
    domWindow.close()
  })

  test('does not repeat the Stripe provider in the USD confirmation', async () => {
    const rendered = await renderDialog({ name: 'Stripe', type: 'stripe' })
    const dialog = document.body.querySelector<HTMLElement>(
      '[data-slot="alert-dialog-content"]'
    )
    const hasPaymentMethod = dialog?.textContent?.includes('Payment Method')
    const hasStripe = dialog?.textContent?.includes('Stripe')

    assert.ok(dialog)
    assert.equal(dialog.textContent?.includes('USD credit amount'), true)
    assert.equal(dialog.textContent?.includes('$2.00 USD'), true)
    assert.equal(dialog.textContent?.includes('You Pay'), true)
    await unmountDialog(rendered)
    assert.equal(hasPaymentMethod, false)
    assert.equal(hasStripe, false)
  })

  test('keeps provider details for non-Stripe payment methods', async () => {
    const rendered = await renderDialog({
      name: 'Bank transfer',
      type: 'custom',
    })
    const dialog = document.body.querySelector<HTMLElement>(
      '[data-slot="alert-dialog-content"]'
    )
    const hasPaymentMethod = dialog?.textContent?.includes('Payment Method')
    const hasProvider = dialog?.textContent?.includes('Bank transfer')

    assert.ok(dialog)
    await unmountDialog(rendered)
    assert.equal(hasPaymentMethod, true)
    assert.equal(hasProvider, true)
  })

  test('shows the daily CNY quote separately from the selected USD credit', async () => {
    const rendered = await renderDialog(
      {
        name: 'WeChat Pay',
        type: 'stripe',
        checkout_method: 'wechat_pay',
      },
      13.54,
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
    const dialog = document.body.querySelector<HTMLElement>(
      '[data-slot="alert-dialog-content"]'
    )

    assert.ok(dialog)
    assert.equal(dialog.textContent?.includes('WeChat Pay'), true)
    assert.equal(
      dialog.textContent?.includes(
        'WeChat checkout displays the final amount in CNY; your wallet is credited with the selected USD amount.'
      ),
      true
    )
    assert.equal(dialog.textContent?.includes('Stripe'), false)
    assert.equal(dialog.textContent?.includes('USD credit amount'), true)
    assert.equal(dialog.textContent?.includes('$2.00 USD'), true)
    assert.equal(dialog.textContent?.includes('You Pay'), true)
    assert.equal(dialog.textContent?.includes('¥13.54 CNY'), true)
    assert.equal(dialog.textContent?.includes('$13.54 USD'), false)
    assert.equal(dialog.textContent?.includes('1 USD = ¥6.7655 CNY'), true)
    assert.equal(
      dialog.textContent?.includes(
        'Exchange rate source: Bank of China spot exchange selling rate'
      ),
      true
    )
    assert.equal(dialog.textContent?.includes('Pricing date: 2026-08-09'), true)
    const quote = dialog.querySelector<HTMLElement>(
      '[data-slot="payment-confirm-quote"]'
    )
    const accessibleWeChatText = [
      ...dialog.querySelectorAll<HTMLElement>('div'),
    ].find((element) => element.textContent?.trim() === 'WeChat Pay')
    assert.equal(quote?.getAttribute('aria-live'), 'polite')
    assert.equal(
      accessibleWeChatText?.classList.contains('text-[#067a3b]'),
      true
    )

    await unmountDialog(rendered)
  })
})
