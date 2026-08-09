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
import { after, test } from 'node:test'

import { Window } from 'happy-dom'

import englishMessages from '../../../../i18n/locales/en.json'
import chineseMessages from '../../../../i18n/locales/zh.json'

const domWindow = new Window()
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'Node',
  'Element',
  'Event',
  'MutationObserver',
] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { StripeFxQuoteDetails } = await import('../stripe-fx-quote-details')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const quote = {
  pay_amount_minor: 1354,
  currency: 'CNY' as const,
  exchange_rate: '6.7655',
  source: 'boc_spot_selling',
  pricing_date: '2026-08-09',
  source_published_at: 1_786_244_200,
  quote_version: 'boc-fx-2026-08-09-v1',
}

after(() => {
  domWindow.close()
})

test('renders the same authoritative quote details in English and Chinese', async () => {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: 'en',
    resources: { en: englishMessages, zhCN: chineseMessages },
    interpolation: { escapeValue: false },
  })
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <I18nextProvider i18n={i18n}>
        <StripeFxQuoteDetails quote={quote} />
      </I18nextProvider>
    )
  })

  assert.equal(container.textContent?.includes('1 USD = ¥6.7655 CNY'), true)
  assert.equal(
    container.textContent?.includes(
      'Exchange rate source: Bank of China spot exchange selling rate'
    ),
    true
  )
  assert.equal(
    container.textContent?.includes('Pricing date: 2026-08-09'),
    true
  )

  await act(async () => {
    await i18n.changeLanguage('zhCN')
  })

  assert.equal(container.textContent?.includes('1 美元 = ¥6.7655 人民币'), true)
  assert.equal(
    container.textContent?.includes('汇率来源：中国银行美元现汇卖出价'),
    true
  )
  assert.equal(container.textContent?.includes('牌价日期：2026-08-09'), true)

  await act(async () => root.unmount())
  container.remove()
})
