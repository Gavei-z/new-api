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

import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { createInstance, type Resource } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'

import englishMessages from '../../../i18n/locales/en.json'
import chineseMessages from '../../../i18n/locales/zh.json'
import { PricingOverview } from '../components/pricing-overview'
import { PRICING_NAV_TITLE_KEY } from '../constants'

async function renderPricingOverview(language: string, resources: Resource) {
  const instance = createInstance()
  await instance.init({
    lng: language,
    fallbackLng: false,
    resources,
    interpolation: { escapeValue: false },
  })

  const rootRoute = createRootRoute({ component: PricingOverview })
  const walletRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/wallet',
    component: () => null,
  })
  const enterpriseRoute = createRoute({
    getParentRoute: () => rootRoute,
    path: '/enterprise',
    component: () => null,
  })
  const router = createRouter({
    routeTree: rootRoute.addChildren([walletRoute, enterpriseRoute]),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })

  await router.load()

  return renderToStaticMarkup(
    <I18nextProvider i18n={instance}>
      <RouterProvider router={router} />
    </I18nextProvider>
  )
}

describe('uniRouters pricing overview', () => {
  test('renders two responsive plan cards with the expected destinations', async () => {
    const markup = await renderPricingOverview('en', { en: englishMessages })

    assert.match(markup, /Simple, transparent pricing\./)
    assert.equal(markup.match(/data-slot="card"/g)?.length, 2)
    assert.match(markup, /md:grid-cols-2/)
    assert.match(markup, /href="\/wallet"/)
    assert.match(markup, /href="\/enterprise"/)
  })

  test('renders a complete comparison table for pay-as-you-go users', async () => {
    const markup = await renderPricingOverview('en', { en: englishMessages })

    assert.match(markup, /Compare plans/)
    assert.equal(markup.match(/<tr/g)?.length, 10)
    assert.match(markup, /No minimum spend or subscription/)
    assert.match(markup, /Real-time usage logs and spend controls/)
  })

  test('renders the pricing proposition and plan names in Chinese', async () => {
    const markup = await renderPricingOverview('zhCN', {
      zhCN: chineseMessages,
    })

    assert.match(markup, /价格简单透明/)
    assert.match(markup, /按量付费/)
    assert.match(markup, /企业大量定制/)
    assert.match(markup, /方案对比/)
  })

  test('uses Pricing as the public navigation title', () => {
    assert.equal(PRICING_NAV_TITLE_KEY, 'Pricing')
    assert.equal(chineseMessages.translation[PRICING_NAV_TITLE_KEY], '定价')
  })
})
