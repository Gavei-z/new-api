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
import type { TopupInfo } from '../../../wallet/types'

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
Object.defineProperty(globalThis, 'requestAnimationFrame', {
  configurable: true,
  value: domWindow.requestAnimationFrame.bind(domWindow),
})
Object.defineProperty(globalThis, 'cancelAnimationFrame', {
  configurable: true,
  value: domWindow.cancelAnimationFrame.bind(domWindow),
})

const { QueryClient, QueryClientProvider } =
  await import('@tanstack/react-query')
const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { TeamStripeTopupDialog } = await import('../team-stripe-topup-dialog')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const topupInfo = {
  enable_online_topup: false,
  enable_stripe_topup: true,
  enable_stripe_wechat_pay: true,
  pay_methods: [{ name: 'Stripe', type: 'stripe' }],
  min_topup: 2,
  stripe_min_topup: 2,
  stripe_max_topup: 50000,
  stripe_amount_options: [2, 5, 10, 50, 200, 500],
  amount_options: [],
  discount: {},
} satisfies TopupInfo

type QuoteFixture = {
  standard?: string
  weChat?: string
}

async function renderTeamTopupDialog(
  overrideTopupInfo: TopupInfo = topupInfo,
  quoteFixture: QuoteFixture = { standard: '2.00', weChat: '14.40' }
) {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    resources: { en: englishMessages },
    interpolation: { escapeValue: false },
  })
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, staleTime: Number.POSITIVE_INFINITY },
      mutations: { retry: false },
    },
  })
  if (quoteFixture.standard !== undefined) {
    queryClient.setQueryData(
      ['team-stripe-amount', 2, 'standard'],
      quoteFixture.standard
    )
  }
  if (quoteFixture.weChat !== undefined) {
    queryClient.setQueryData(
      ['team-stripe-amount', 2, 'wechat_pay'],
      quoteFixture.weChat
    )
  }

  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)
  await act(async () => {
    root.render(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <TeamStripeTopupDialog
            open
            onOpenChange={() => undefined}
            topupInfo={overrideTopupInfo}
          />
        </I18nextProvider>
      </QueryClientProvider>
    )
  })

  return { container, queryClient, root }
}

describe('team Stripe top-up checkout actions', () => {
  after(() => {
    domWindow.close()
  })

  test('offers neutral and WeChat actions without exposing Stripe branding', async () => {
    const rendered = await renderTeamTopupDialog()
    const dialog = document.body.querySelector<HTMLElement>(
      '[data-slot="dialog-content"]'
    )
    const buttons = [
      ...(dialog?.querySelectorAll<HTMLButtonElement>('button') ?? []),
    ]
    const payButton = buttons.find((button) => button.textContent === 'Pay')
    const weChatButton = buttons.find((button) =>
      button.textContent?.includes('WeChat Pay')
    )

    assert.ok(dialog)
    assert.ok(payButton)
    assert.ok(weChatButton)
    assert.equal(payButton.disabled, false)
    assert.equal(weChatButton.disabled, false)
    assert.equal(dialog.textContent?.includes('Stripe'), false)
    assert.equal(dialog.textContent?.includes('USD credit amount'), true)
    assert.equal(dialog.textContent?.includes('$2.00 USD'), true)
    assert.equal(dialog.textContent?.includes('¥14.40 CNY'), true)
    assert.equal(dialog.textContent?.includes('$14.40 USD'), false)
    const standardQuote = dialog.querySelector<HTMLElement>(
      '[data-slot="team-standard-payment-quote"]'
    )
    const weChatQuote = dialog.querySelector<HTMLElement>(
      '[data-slot="team-wechat-payment-quote"]'
    )
    assert.equal(standardQuote?.getAttribute('aria-live'), 'polite')
    assert.equal(weChatQuote?.getAttribute('aria-live'), 'polite')
    assert.equal(weChatButton.classList.contains('text-[#067a3b]'), true)
    assert.equal(weChatButton.classList.contains('text-[#079447]'), false)
    assert.equal(
      dialog.textContent?.includes(
        'WeChat checkout displays the final amount in CNY; your wallet is credited with the selected USD amount.'
      ),
      true
    )

    await act(async () => rendered.root.unmount())
    rendered.queryClient.clear()
    rendered.container.remove()
  })

  test('uses the two-column footer when WeChat capability is absent', async () => {
    const rendered = await renderTeamTopupDialog({
      ...topupInfo,
      enable_stripe_wechat_pay: false,
    })
    const dialog = document.body.querySelector<HTMLElement>(
      '[data-slot="dialog-content"]'
    )
    const footer = dialog?.querySelector<HTMLElement>(
      '[data-slot="dialog-footer"]'
    )

    assert.ok(dialog)
    assert.ok(footer)
    assert.equal(dialog.textContent?.includes('WeChat Pay'), false)
    assert.equal(footer.classList.contains('sm:grid-cols-[auto_1fr]'), true)

    await act(async () => rendered.root.unmount())
    rendered.queryClient.clear()
    rendered.container.remove()
  })

  test('disables only the standard action when its quote is invalid', async () => {
    const rendered = await renderTeamTopupDialog(topupInfo, {
      standard: '0',
      weChat: '14.40',
    })
    const dialog = document.body.querySelector<HTMLElement>(
      '[data-slot="dialog-content"]'
    )
    const buttons = [
      ...(dialog?.querySelectorAll<HTMLButtonElement>('button') ?? []),
    ]
    const payButton = buttons.find((button) => button.textContent === 'Pay')
    const weChatButton = buttons.find((button) =>
      button.textContent?.includes('WeChat Pay')
    )

    assert.equal(payButton?.disabled, true)
    assert.equal(weChatButton?.disabled, false)

    await act(async () => rendered.root.unmount())
    rendered.queryClient.clear()
    rendered.container.remove()
  })

  test('disables only the WeChat action when its CNY quote is invalid', async () => {
    const rendered = await renderTeamTopupDialog(topupInfo, {
      standard: '2.00',
      weChat: '0',
    })
    const dialog = document.body.querySelector<HTMLElement>(
      '[data-slot="dialog-content"]'
    )
    const buttons = [
      ...(dialog?.querySelectorAll<HTMLButtonElement>('button') ?? []),
    ]
    const payButton = buttons.find((button) => button.textContent === 'Pay')
    const weChatButton = buttons.find((button) =>
      button.textContent?.includes('WeChat Pay')
    )

    assert.equal(payButton?.disabled, false)
    assert.equal(weChatButton?.disabled, true)

    await act(async () => rendered.root.unmount())
    rendered.queryClient.clear()
    rendered.container.remove()
  })
})
