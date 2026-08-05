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

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'Node',
  'Element',
  'Event',
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
const { LanguageSwitcher } = await import('../language-switcher')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('segmented language switcher', () => {
  after(() => {
    domWindow.close()
  })

  test('shows explicit EN and CN controls and updates the pressed language', async () => {
    const i18n = createInstance()
    await i18n.use(initReactI18next).init({
      lng: 'en',
      fallbackLng: false,
      resources: {
        en: {
          translation: {
            'Change language': 'Change language',
            Chinese: 'Chinese',
            English: 'English',
          },
        },
        zhCN: {
          translation: {
            'Change language': '更改语言',
            Chinese: '中文',
            English: '英文',
          },
        },
      },
    })

    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <LanguageSwitcher variant='segmented' />
        </I18nextProvider>
      )
    })

    const group = container.querySelector('[role="group"]')
    const buttons = container.querySelectorAll<HTMLButtonElement>('button')
    const englishButton = buttons[0]
    const chineseButton = buttons[1]

    assert.equal(group?.getAttribute('aria-label'), 'Change language')
    assert.equal(buttons.length, 2)
    assert.equal(englishButton?.textContent, 'EN')
    assert.equal(chineseButton?.textContent, 'CN')
    assert.equal(englishButton?.getAttribute('aria-pressed'), 'true')
    assert.equal(chineseButton?.getAttribute('aria-pressed'), 'false')

    assert.ok(chineseButton)
    await act(async () => {
      chineseButton.click()
      await Promise.resolve()
    })

    assert.equal(i18n.language, 'zhCN')
    assert.equal(englishButton?.getAttribute('aria-pressed'), 'false')
    assert.equal(chineseButton.getAttribute('aria-pressed'), 'true')

    assert.ok(englishButton)
    await act(async () => {
      englishButton.click()
      await Promise.resolve()
    })

    assert.equal(i18n.language, 'en')
    assert.equal(englishButton.getAttribute('aria-pressed'), 'true')
    assert.equal(chineseButton.getAttribute('aria-pressed'), 'false')

    await act(async () => root.unmount())
    container.remove()
  })
})
