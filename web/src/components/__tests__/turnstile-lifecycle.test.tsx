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
import { after, beforeEach, describe, test } from 'node:test'

import { Window } from 'happy-dom'

const domWindow = new Window({
  url: 'https://unirouters.cc/sign-in',
  settings: { disableJavaScriptFileLoading: true },
})
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLScriptElement',
  'HTMLButtonElement',
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

// Happy DOM attempts to fetch connected external scripts immediately. Keep the
// Turnstile script inert so each test can deterministically dispatch load/error.
const appendToHead = document.head.appendChild.bind(document.head)
document.head.appendChild = (<T extends Node>(node: T): T => {
  if (node instanceof HTMLScriptElement && node.id === 'cf-turnstile') {
    node.type = 'application/json'
  }
  return appendToHead(node) as T
}) as typeof document.head.appendChild

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { Turnstile } = await import('../turnstile')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type TurnstileOptions = {
  callback?: (token: string) => void
}

async function createTestI18n() {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    resources: {
      en: {
        translation: {
          'Please wait a moment, human check is initializing...':
            'Human verification is loading...',
          'Human verification failed to load. Please retry.':
            'Human verification failed to load. Please retry.',
          Retry: 'Retry',
        },
      },
    },
  })
  return i18n
}

describe('Turnstile lifecycle', () => {
  beforeEach(() => {
    document.head.replaceChildren()
    document.body.replaceChildren()
    Reflect.deleteProperty(domWindow, 'turnstile')
  })

  after(() => {
    domWindow.close()
  })

  test('renders the current component when the shared script finishes after a route remount', async () => {
    const i18n = await createTestI18n()
    const firstTokens: string[] = []
    const secondTokens: string[] = []
    const removedWidgets: string[] = []
    let renderedElement: HTMLElement | undefined
    let renderedOptions: TurnstileOptions | undefined

    const firstContainer = document.createElement('div')
    document.body.append(firstContainer)
    const firstRoot = createRoot(firstContainer)
    await act(async () => {
      firstRoot.render(
        <I18nextProvider i18n={i18n}>
          <Turnstile
            siteKey='test-site-key'
            onVerify={(token) => firstTokens.push(token)}
          />
        </I18nextProvider>
      )
    })

    const pendingScript =
      document.querySelector<HTMLScriptElement>('#cf-turnstile')
    assert.ok(pendingScript)
    await act(async () => firstRoot.unmount())
    firstContainer.remove()

    const secondContainer = document.createElement('div')
    document.body.append(secondContainer)
    const secondRoot = createRoot(secondContainer)
    await act(async () => {
      secondRoot.render(
        <I18nextProvider i18n={i18n}>
          <Turnstile
            siteKey='test-site-key'
            onVerify={(token) => secondTokens.push(token)}
          />
        </I18nextProvider>
      )
    })
    assert.equal(document.querySelectorAll('#cf-turnstile').length, 1)

    Object.defineProperty(domWindow, 'turnstile', {
      configurable: true,
      value: {
        render: (element: HTMLElement, options: TurnstileOptions) => {
          renderedElement = element
          renderedOptions = options
          return 'current-widget'
        },
        remove: (widgetId: string) => removedWidgets.push(widgetId),
        reset: () => undefined,
      },
    })
    await act(async () => {
      pendingScript.dispatchEvent(new Event('load'))
      await Promise.resolve()
    })

    assert.equal(renderedElement?.parentElement, secondContainer)
    await act(async () => renderedOptions?.callback?.('verified-token'))
    assert.deepEqual(firstTokens, [])
    assert.deepEqual(secondTokens, ['verified-token'])

    await act(async () => secondRoot.unmount())
    assert.deepEqual(removedWidgets, ['current-widget'])
    secondContainer.remove()
  })

  test('shows a retry action after script failure and renders from a fresh script', async () => {
    const i18n = await createTestI18n()
    const errors: string[] = []
    const removedWidgets: string[] = []
    let renderCount = 0

    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <Turnstile
            siteKey='test-site-key'
            onVerify={() => undefined}
            onError={() => errors.push('failed')}
          />
        </I18nextProvider>
      )
    })

    const failedScript =
      document.querySelector<HTMLScriptElement>('#cf-turnstile')
    assert.ok(failedScript)
    await act(async () => {
      failedScript.dispatchEvent(new Event('error'))
      await Promise.resolve()
    })

    assert.match(
      container.textContent ?? '',
      /Human verification failed to load/
    )
    assert.deepEqual(errors, ['failed'])
    const retryButton = [...container.querySelectorAll('button')].find(
      (button) => button.textContent === 'Retry'
    )
    assert.ok(retryButton)

    await act(async () => retryButton.click())
    const retryScript =
      document.querySelector<HTMLScriptElement>('#cf-turnstile')
    assert.ok(retryScript)
    assert.notEqual(retryScript, failedScript)

    Object.defineProperty(domWindow, 'turnstile', {
      configurable: true,
      value: {
        render: () => {
          renderCount += 1
          return 'retry-widget'
        },
        remove: (widgetId: string) => removedWidgets.push(widgetId),
        reset: () => undefined,
      },
    })
    await act(async () => {
      retryScript.dispatchEvent(new Event('load'))
      await Promise.resolve()
    })
    assert.equal(renderCount, 1)

    await act(async () => root.unmount())
    assert.deepEqual(removedWidgets, ['retry-widget'])
    container.remove()
  })

  test('keeps the latest verification callback without recreating the widget', async () => {
    const i18n = await createTestI18n()
    const firstTokens: string[] = []
    const secondTokens: string[] = []
    const removedWidgets: string[] = []
    let renderCount = 0
    let renderedOptions: TurnstileOptions | undefined

    Object.defineProperty(domWindow, 'turnstile', {
      configurable: true,
      value: {
        render: (_element: HTMLElement, options: TurnstileOptions) => {
          renderCount += 1
          renderedOptions = options
          return 'stable-widget'
        },
        remove: (widgetId: string) => removedWidgets.push(widgetId),
      },
    })

    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <Turnstile
            siteKey='test-site-key'
            onVerify={(token) => firstTokens.push(token)}
          />
        </I18nextProvider>
      )
    })
    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <Turnstile
            siteKey='test-site-key'
            onVerify={(token) => secondTokens.push(token)}
          />
        </I18nextProvider>
      )
    })

    assert.equal(renderCount, 1)
    await act(async () => renderedOptions?.callback?.('fresh-token'))
    assert.deepEqual(firstTokens, [])
    assert.deepEqual(secondTokens, ['fresh-token'])

    await act(async () => root.unmount())
    assert.deepEqual(removedWidgets, ['stable-widget'])
    container.remove()
  })
})
