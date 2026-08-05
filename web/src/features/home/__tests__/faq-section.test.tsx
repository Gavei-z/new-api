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

import { createInstance, type Resource } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'

import englishMessages from '../../../i18n/locales/en.json'
import chineseMessages from '../../../i18n/locales/zh.json'
import { FaqSection } from '../components/faq-section'

async function renderFaq(language: string, resources: Resource) {
  const instance = createInstance()
  await instance.init({
    lng: language,
    fallbackLng: false,
    resources,
    interpolation: { escapeValue: false },
  })

  return renderToStaticMarkup(
    <I18nextProvider i18n={instance}>
      <FaqSection />
    </I18nextProvider>
  )
}

describe('uniRouters home FAQ section', () => {
  test('renders all five English questions and billing details', async () => {
    const markup = await renderFaq('en', { en: englishMessages })

    assert.equal(markup.match(/<details/g)?.length, 5)
    assert.match(markup, /How do I use UniRouters?/)
    assert.match(markup, /We buy at wholesale rates/)
    assert.match(markup, /cache reads/)
    assert.match(markup, /USD 40/)
    assert.doesNotMatch(markup, /special VAT/)
    assert.match(markup, /multiple API keys/)
  })

  test('renders the Chinese FAQ translations', async () => {
    const markup = await renderFaq('zhCN', { zhCN: chineseMessages })

    assert.equal(markup.match(/<details/g)?.length, 5)
    assert.match(markup, /如何使用 UniRouters？/)
    assert.match(markup, /批量采购/)
    assert.match(markup, /缓存读取/)
    assert.match(markup, /40 美元/)
    assert.doesNotMatch(markup, /增值税专用/)
    assert.match(markup, /多个 API Key/)
  })
})
