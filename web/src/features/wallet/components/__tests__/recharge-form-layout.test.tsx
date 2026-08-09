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

import englishMessages from '../../../../i18n/locales/en.json'
import type { PaymentMethod, TopupInfo } from '../../types'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'HTMLInputElement',
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

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { RechargeFormCard } = await import('../recharge-form-card')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const stripeMethod = { name: 'Stripe', type: 'stripe' } satisfies PaymentMethod
const stripeTopupInfo = {
  enable_online_topup: false,
  enable_stripe_topup: true,
  enable_stripe_wechat_pay: true,
  pay_methods: [stripeMethod],
  min_topup: 2,
  stripe_min_topup: 2,
  stripe_max_topup: 50000,
  amount_options: [2, 5, 10, 50, 200, 500],
  discount: {},
  enable_redemption: false,
} satisfies TopupInfo
const integerGatewayTopupInfo = {
  ...stripeTopupInfo,
  enable_online_topup: true,
  enable_stripe_topup: false,
  pay_methods: [{ name: 'Bank transfer', type: 'custom' }],
} satisfies TopupInfo

type RenderOptions = {
  topupAmount?: number
  paymentAmount?: number
  paymentLoading?: string | null
  topupInfo?: TopupInfo
  onPaymentMethodSelect?: (method: PaymentMethod) => void
}

async function renderRechargeForm(options: RenderOptions = {}) {
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
        <RechargeFormCard
          topupInfo={options.topupInfo ?? stripeTopupInfo}
          presetAmounts={[]}
          selectedPreset={null}
          onSelectPreset={() => undefined}
          topupAmount={options.topupAmount ?? 2}
          onTopupAmountChange={() => undefined}
          paymentAmount={options.paymentAmount ?? 2}
          calculating={false}
          onPaymentMethodSelect={
            options.onPaymentMethodSelect ?? (() => undefined)
          }
          paymentLoading={options.paymentLoading ?? null}
          redemptionCode=''
          onRedemptionCodeChange={() => undefined}
          onRedeem={() => undefined}
          redeeming={false}
        />
      </I18nextProvider>
    )
  })

  return { container, root }
}

async function unmountRechargeForm(
  rendered: Awaited<ReturnType<typeof renderRechargeForm>>
) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

describe('wallet recharge form layout', () => {
  after(() => {
    domWindow.close()
  })

  test('shows separate neutral and WeChat actions without Stripe branding', async () => {
    const selectedMethods: PaymentMethod[] = []
    const rendered = await renderRechargeForm({
      onPaymentMethodSelect: (method) => {
        selectedMethods.push(method)
      },
    })

    const payButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-slot="wallet-primary-payment-button"]'
    )
    const weChatButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-slot="wallet-wechat-payment-button"]'
    )

    assert.ok(payButton)
    assert.ok(weChatButton)
    assert.equal(payButton.textContent?.trim(), 'Pay')
    assert.equal(weChatButton.textContent?.trim(), 'WeChat Pay')
    assert.equal(
      rendered.container.textContent?.includes('Payment Method'),
      false
    )
    assert.equal(rendered.container.textContent?.includes('Stripe'), false)

    await act(async () => payButton.click())
    await act(async () => weChatButton.click())
    assert.deepEqual(selectedMethods, [
      { ...stripeMethod, checkout_method: 'standard' },
      {
        ...stripeMethod,
        name: 'WeChat Pay',
        checkout_method: 'wechat_pay',
      },
    ])

    await unmountRechargeForm(rendered)
  })

  test('keeps custom amount and payment total compact and top-aligned', async () => {
    const rendered = await renderRechargeForm()
    const amountLayout = rendered.container.querySelector<HTMLElement>(
      '[data-slot="wallet-amount-entry"]'
    )
    const input =
      rendered.container.querySelector<HTMLInputElement>('#topup-amount')
    const total = rendered.container.querySelector<HTMLElement>(
      '[data-slot="wallet-payment-total-display"]'
    )

    assert.ok(amountLayout)
    assert.equal(amountLayout.classList.contains('grid-cols-1'), true)
    assert.equal(
      amountLayout.classList.contains(
        'sm:grid-cols-[minmax(0,18rem)_minmax(0,16rem)]'
      ),
      true
    )
    assert.equal(amountLayout.classList.contains('sm:items-start'), true)
    assert.equal(amountLayout.classList.contains('max-w-2xl'), true)
    assert.ok(input)
    assert.equal(input.classList.contains('h-10'), true)
    assert.ok(total)
    assert.equal(total.classList.contains('h-10'), true)
    assert.equal(total.classList.contains('justify-start'), true)
    assert.equal(total.classList.contains('justify-end'), false)

    await unmountRechargeForm(rendered)
  })

  test('disables the single payment action when the USD amount is invalid', async () => {
    const rendered = await renderRechargeForm({
      topupAmount: 1,
      paymentAmount: 0,
    })
    const input =
      rendered.container.querySelector<HTMLInputElement>('#topup-amount')
    const payButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-slot="wallet-primary-payment-button"]'
    )
    const weChatButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-slot="wallet-wechat-payment-button"]'
    )

    assert.equal(input?.getAttribute('aria-invalid'), 'true')
    assert.equal(payButton?.disabled, true)
    assert.equal(weChatButton?.disabled, true)

    await unmountRechargeForm(rendered)
  })

  test('prevents duplicate payment selection while checkout is loading', async () => {
    const rendered = await renderRechargeForm({
      paymentLoading: 'stripe:wechat_pay',
    })
    const payButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-slot="wallet-primary-payment-button"]'
    )
    const weChatButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-slot="wallet-wechat-payment-button"]'
    )

    assert.equal(payButton?.disabled, true)
    assert.equal(weChatButton?.disabled, true)
    assert.equal(payButton?.querySelector('svg.animate-spin'), null)
    assert.ok(weChatButton?.querySelector('svg.animate-spin'))

    await unmountRechargeForm(rendered)
  })

  test('shows loading only on the standard action when standard checkout starts', async () => {
    const rendered = await renderRechargeForm({
      paymentLoading: 'stripe:standard',
    })
    const payButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-slot="wallet-primary-payment-button"]'
    )
    const weChatButton = rendered.container.querySelector<HTMLButtonElement>(
      '[data-slot="wallet-wechat-payment-button"]'
    )

    assert.equal(payButton?.disabled, true)
    assert.equal(weChatButton?.disabled, true)
    assert.ok(payButton?.querySelector('svg.animate-spin'))
    assert.equal(weChatButton?.querySelector('svg.animate-spin'), null)

    await unmountRechargeForm(rendered)
  })

  test('keeps the standard action compact when WeChat capability is absent', async () => {
    const rendered = await renderRechargeForm({
      topupInfo: {
        ...stripeTopupInfo,
        enable_stripe_wechat_pay: false,
      },
    })
    const actions = rendered.container.querySelector<HTMLElement>(
      '[data-slot="wallet-stripe-checkout-actions"]'
    )

    assert.ok(actions)
    assert.equal(
      rendered.container.querySelector(
        '[data-slot="wallet-wechat-payment-button"]'
      ),
      null
    )
    assert.equal(actions.classList.contains('sm:grid-cols-1'), true)

    await unmountRechargeForm(rendered)
  })

  test('rejects fractional amounts for integer payment gateways', async () => {
    let selectedMethod: PaymentMethod | undefined
    const rendered = await renderRechargeForm({
      topupAmount: 10.5,
      paymentAmount: 0,
      topupInfo: integerGatewayTopupInfo,
      onPaymentMethodSelect: (method) => {
        selectedMethod = method
      },
    })
    const input =
      rendered.container.querySelector<HTMLInputElement>('#topup-amount')
    const paymentButton = [
      ...rendered.container.querySelectorAll<HTMLButtonElement>('button'),
    ].find((button) => button.textContent?.includes('Bank transfer'))

    assert.equal(input?.getAttribute('aria-invalid'), 'true')
    assert.equal(
      rendered.container.textContent?.includes('Amount must be a whole number'),
      true
    )
    assert.equal(paymentButton?.disabled, true)
    paymentButton?.click()
    assert.equal(selectedMethod, undefined)

    await unmountRechargeForm(rendered)
  })

  test('preserves the payment method picker when multiple methods exist', async () => {
    const rendered = await renderRechargeForm({
      topupInfo: {
        ...stripeTopupInfo,
        enable_online_topup: true,
        pay_methods: [stripeMethod, { name: 'Bank transfer', type: 'custom' }],
      },
    })

    assert.equal(
      rendered.container.textContent?.includes('Payment Method'),
      true
    )
    assert.equal(rendered.container.textContent?.includes('Stripe'), false)
    assert.equal(rendered.container.textContent?.includes('WeChat Pay'), true)
    assert.equal(
      rendered.container.textContent?.includes('Bank transfer'),
      true
    )
    assert.equal(
      rendered.container.querySelector(
        '[data-slot="wallet-primary-payment-button"]'
      ),
      null
    )

    await unmountRechargeForm(rendered)
  })
})
