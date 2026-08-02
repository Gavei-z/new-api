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
import { ModelCardsGrid } from '../components/landing-page'

async function renderModelCards(language: string, resources: Resource) {
  const instance = createInstance()
  await instance.init({
    lng: language,
    fallbackLng: false,
    resources,
    interpolation: { escapeValue: false },
  })

  return renderToStaticMarkup(
    <I18nextProvider i18n={instance}>
      <ModelCardsGrid />
    </I18nextProvider>
  )
}

describe('uniRouters home model showcase', () => {
  test('renders five model cards and all GPT Image 2 pricing lanes', async () => {
    const markup = await renderModelCards('en', { en: englishMessages })

    assert.equal(markup.match(/<article/g)?.length, 5)
    assert.match(markup, /gpt-image-2/)
    assert.match(markup, /Text Input/)
    assert.match(markup, /Image input/)
    assert.match(markup, /Image output/)
    assert.match(markup, /\$5\.00/)
    assert.match(markup, /\$8\.00/)
    assert.match(markup, /\$30\.00/)
  })

  test('renders the GPT Image 2 pricing labels in Chinese', async () => {
    const markup = await renderModelCards('zhCN', { zhCN: chineseMessages })

    assert.match(markup, /文字输入/)
    assert.match(markup, /图片输入/)
    assert.match(markup, /图片输出/)
    assert.match(markup, /我们的售价/)
  })
})
