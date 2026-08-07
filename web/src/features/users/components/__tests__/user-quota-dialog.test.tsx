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
  'ResizeObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
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
const { api } = await import('@/lib/api')
const { UserQuotaDialog } = await import('../user-quota-dialog')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('administrator quota adjustment dialog', () => {
  after(() => {
    domWindow.close()
  })

  test('preserves one idempotency key and locks the intent while a request is in flight', async () => {
    const i18n = createInstance()
    await i18n.use(initReactI18next).init({
      lng: 'en',
      fallbackLng: false,
      resources: { en: englishMessages },
      interpolation: { escapeValue: false },
    })

    const payloads: Array<Record<string, unknown>> = []
    let resolveFirst:
      | ((response: { data: { success: boolean; message?: string } }) => void)
      | undefined
    const originalPost = api.post
    api.post = (async (_url: string, payload: Record<string, unknown>) => {
      payloads.push(payload)
      if (payloads.length === 1) {
        return await new Promise<{
          data: { success: boolean; message?: string }
        }>((resolve) => {
          resolveFirst = resolve
        })
      }
      return { data: { success: true } }
    }) as typeof api.post

    const openChanges: boolean[] = []
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    try {
      await act(async () => {
        root.render(
          <I18nextProvider i18n={i18n}>
            <UserQuotaDialog
              open
              onOpenChange={(open) => openChanges.push(open)}
              userId={7}
              currentQuota={100}
              onSuccess={() => undefined}
            />
          </I18nextProvider>
        )
      })

      const input = document.body.querySelector<HTMLInputElement>(
        'input[type="number"]'
      )
      assert.ok(input)
      const setInputValue = Object.getOwnPropertyDescriptor(
        domWindow.HTMLInputElement.prototype,
        'value'
      )?.set
      assert.ok(setInputValue)
      await act(async () => {
        setInputValue.call(input, '2')
        input.dispatchEvent(new Event('input', { bubbles: true }))
      })
      const buttons = [
        ...document.body.querySelectorAll<HTMLButtonElement>('button'),
      ]
      const confirm = buttons.find((button) =>
        button.textContent?.includes('Confirm')
      )
      const cancel = buttons.find((button) =>
        button.textContent?.includes('Cancel')
      )
      assert.ok(confirm)
      assert.ok(cancel)

      await act(async () => {
        confirm.click()
      })
      assert.equal(payloads.length, 1)
      assert.equal(input.disabled, true)
      assert.equal(cancel.disabled, true)
      assert.equal(
        document.body.querySelector('[data-slot="dialog-close"]'),
        null
      )
      assert.equal(openChanges.length, 0)

      await act(async () => {
        resolveFirst?.({ data: { success: false, message: 'retry' } })
      })
      assert.equal(input.disabled, false)

      await act(async () => {
        confirm.click()
      })
      assert.equal(payloads.length, 2)
      assert.equal(payloads[0]?.idempotency_key, payloads[1]?.idempotency_key)
      assert.equal(openChanges.at(-1), false)
    } finally {
      api.post = originalPost
      await act(async () => root.unmount())
      container.remove()
    }
  })
})
