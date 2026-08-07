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
import type { PaymentMethod } from '../../../types'

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

async function renderDialog(paymentMethod: PaymentMethod) {
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

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <PaymentConfirmDialog
          open
          onOpenChange={() => undefined}
          onConfirm={() => undefined}
          topupAmount={2}
          paymentAmount={2}
          paymentMethod={paymentMethod}
          calculating={false}
          processing={false}
          currencyCode='USD'
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
})
