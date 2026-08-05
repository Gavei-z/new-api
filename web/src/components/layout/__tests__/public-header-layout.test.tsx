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
const { PublicHeaderChrome } =
  await import('../components/public-header-chrome')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('public header layout', () => {
  after(() => {
    domWindow.close()
  })

  test('keeps its full width and height after scrolling while adding floating glass styling', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <PublicHeaderChrome scrolled={false}>
          <span>Navigation</span>
        </PublicHeaderChrome>
      )
    })

    const frame = container.querySelector<HTMLElement>(
      '[data-slot="public-header-frame"]'
    )
    const nav = container.querySelector<HTMLElement>(
      '[data-slot="public-header-nav"]'
    )

    assert.ok(frame)
    assert.ok(nav)
    assert.equal(frame.classList.contains('max-w-7xl'), true)
    assert.equal(nav.classList.contains('h-16'), true)
    assert.equal(nav.classList.contains('rounded-2xl'), false)

    await act(async () => {
      root.render(
        <PublicHeaderChrome scrolled>
          <span>Navigation</span>
        </PublicHeaderChrome>
      )
    })

    assert.equal(frame.classList.contains('max-w-7xl'), true)
    assert.equal(frame.classList.contains('max-w-[52rem]'), false)
    assert.equal(nav.classList.contains('h-16'), true)
    assert.equal(nav.classList.contains('h-12'), false)
    assert.equal(nav.classList.contains('rounded-2xl'), true)
    assert.equal(nav.classList.contains('bg-background/60'), true)
    assert.equal(nav.classList.contains('backdrop-blur-2xl'), true)

    await act(async () => root.unmount())
    container.remove()
  })
})
