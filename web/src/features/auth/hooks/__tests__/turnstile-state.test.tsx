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

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Window } from 'happy-dom'

const domWindow = new Window({ url: 'https://unirouters.cc/sign-in' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'localStorage',
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
const { useTurnstile } = await import('../use-turnstile')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type TurnstileState = ReturnType<typeof useTurnstile>

describe('Turnstile authentication state', () => {
  after(() => {
    domWindow.close()
  })

  test('reset clears a consumed token and advances a stable widget key', async () => {
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    })
    queryClient.setQueryData(['status'], {
      turnstile_check: true,
      turnstile_site_key: 'test-site-key',
    })

    let currentState: TurnstileState | undefined
    function Harness() {
      currentState = useTurnstile()
      return null
    }

    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    await act(async () => {
      root.render(
        <QueryClientProvider client={queryClient}>
          <Harness />
        </QueryClientProvider>
      )
    })

    assert.ok(currentState)
    const initialReset = currentState.resetTurnstile
    assert.equal(currentState.turnstileWidgetKey, 0)

    await act(async () => currentState?.setTurnstileToken('verified-token'))
    assert.equal(currentState.turnstileToken, 'verified-token')
    assert.equal(currentState.resetTurnstile, initialReset)

    await act(async () => currentState?.resetTurnstile())
    assert.equal(currentState.turnstileToken, '')
    assert.equal(currentState.turnstileWidgetKey, 1)
    assert.equal(currentState.resetTurnstile, initialReset)

    await act(async () => root.unmount())
    queryClient.clear()
    container.remove()
  })
})
