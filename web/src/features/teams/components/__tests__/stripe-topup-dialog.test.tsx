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

import type { AxiosAdapter, AxiosResponse } from 'axios'
import { Window } from 'happy-dom'

import englishMessages from '../../../../i18n/locales/en.json'
import type {
  PaymentAmountQuote,
  StripeFxQuote,
  TopupInfo,
} from '../../../wallet/types'

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
const { api } = await import('../../../../lib/api')
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

const stripeFxQuote = {
  pay_amount_minor: 1354,
  currency: 'CNY',
  exchange_rate: '6.7655',
  source: 'boc_spot_selling',
  pricing_date: '2026-08-09',
  source_published_at: 1_786_244_200,
  quote_version: 'boc-fx-2026-08-09-v1',
} satisfies StripeFxQuote

const standardQuote = { amount: 2 } satisfies PaymentAmountQuote
const weChatQuote = {
  amount: 13.54,
  stripeFxQuote,
} satisfies PaymentAmountQuote

function createAxiosResponse(
  config: Parameters<AxiosAdapter>[0],
  data: unknown
): AxiosResponse {
  return {
    data,
    status: 200,
    statusText: 'OK',
    headers: {},
    config,
  }
}

type QuoteFixture = {
  standard?: PaymentAmountQuote
  weChat?: PaymentAmountQuote
}

async function renderTeamTopupDialog(
  overrideTopupInfo: TopupInfo = topupInfo,
  quoteFixture: QuoteFixture = {
    standard: standardQuote,
    weChat: weChatQuote,
  }
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
      standard: { amount: 0 },
      weChat: weChatQuote,
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
      standard: standardQuote,
      weChat: { amount: 0 },
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

  test('refreshes an expired WeChat quote before enabling checkout again', async () => {
    const rendered = await renderTeamTopupDialog()
    const previousAdapter = api.defaults.adapter
    const quoteKey = ['team-stripe-amount', 2, 'wechat_pay'] as const
    let resolveRefreshData!: (data: unknown) => void
    let markRefreshStarted!: () => void
    const refreshData = new Promise<unknown>((resolve) => {
      resolveRefreshData = resolve
    })
    const refreshStarted = new Promise<void>((resolve) => {
      markRefreshStarted = resolve
    })
    let refreshRequestCount = 0

    const adapter: AxiosAdapter = async (config) => {
      if (config.url === '/api/team/stripe/pay') {
        return createAxiosResponse(config, {
          message: 'error',
          code: 'stripe_fx_quote_expired',
          data: 'WeChat Pay quote expired; request a new quote',
        })
      }
      if (config.url === '/api/team/stripe/amount') {
        refreshRequestCount += 1
        markRefreshStarted()
        return createAxiosResponse(config, await refreshData)
      }
      throw new Error(`Unexpected request: ${config.url}`)
    }
    api.defaults.adapter = adapter

    try {
      const dialog = document.body.querySelector<HTMLElement>(
        '[data-slot="dialog-content"]'
      )
      if (!dialog) throw new Error('Team top-up dialog did not render')
      const weChatButton = [
        ...dialog.querySelectorAll<HTMLButtonElement>('button'),
      ].find((button) => button.textContent?.includes('WeChat Pay'))
      assert.ok(weChatButton)

      await act(async () => {
        weChatButton.click()
        await refreshStarted
      })

      assert.equal(refreshRequestCount, 1)
      assert.equal(weChatButton.disabled, true)
      assert.equal(
        rendered.queryClient.getQueryState(quoteKey)?.fetchStatus,
        'fetching'
      )

      const mutationSettled = new Promise<void>((resolve) => {
        const unsubscribe = rendered.queryClient
          .getMutationCache()
          .subscribe(() => {
            const mutation = rendered.queryClient
              .getMutationCache()
              .getAll()
              .at(-1)
            if (mutation?.state.status === 'error') {
              unsubscribe()
              resolve()
            }
          })
      })
      const checkoutEnabled = new Promise<void>((resolve) => {
        const observer = new MutationObserver(() => {
          const currentWeChatButton = [
            ...dialog.querySelectorAll<HTMLButtonElement>('button'),
          ].find((button) => button.textContent?.includes('WeChat Pay'))
          if (currentWeChatButton && !currentWeChatButton.disabled) {
            observer.disconnect()
            resolve()
          }
        })
        observer.observe(dialog, {
          attributes: true,
          attributeFilter: ['disabled'],
          subtree: true,
        })
      })

      await act(async () => {
        resolveRefreshData({
          message: 'success',
          data: '13.55',
          quote: {
            ...stripeFxQuote,
            pay_amount_minor: 1355,
            exchange_rate: '6.7750',
            quote_version: 'boc-fx-2026-08-09-v2',
          },
        })
        await mutationSettled
        await checkoutEnabled
      })

      const refreshedQuote = rendered.queryClient.getQueryData(
        quoteKey
      ) as PaymentAmountQuote
      assert.equal(
        refreshedQuote.stripeFxQuote?.quote_version,
        'boc-fx-2026-08-09-v2'
      )
      const refreshedWeChatButton = [
        ...dialog.querySelectorAll<HTMLButtonElement>('button'),
      ].find((button) => button.textContent?.includes('WeChat Pay'))
      assert.equal(refreshedWeChatButton?.disabled, false)
      assert.equal(dialog.textContent?.includes('¥13.55 CNY'), true)
    } finally {
      api.defaults.adapter = previousAdapter
      await act(async () => rendered.root.unmount())
      rendered.queryClient.clear()
      rendered.container.remove()
    }
  })
})
