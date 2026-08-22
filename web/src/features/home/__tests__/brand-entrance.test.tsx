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

const domWindow = new Window({ url: 'https://unirouters.cc/' })
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLImageElement',
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

let reduceMotion = false
Object.defineProperty(domWindow, 'matchMedia', {
  configurable: true,
  value: (query: string) => ({
    matches: reduceMotion && query === '(prefers-reduced-motion: reduce)',
    media: query,
    onchange: null,
    addListener: () => undefined,
    removeListener: () => undefined,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    dispatchEvent: () => false,
  }),
})

type AnimationCall = {
  element: HTMLElement
  keyframes: Keyframe[]
  options: KeyframeAnimationOptions
}

const animationCalls: AnimationCall[] = []
const neverFinishes = new Promise<Animation>(() => undefined)

Object.defineProperty(domWindow.HTMLElement.prototype, 'animate', {
  configurable: true,
  value: function animate(
    keyframes: Keyframe[],
    options: KeyframeAnimationOptions
  ) {
    animationCalls.push({
      element: this as HTMLElement,
      keyframes,
      options,
    })
    return {
      cancel: () => undefined,
      finished: neverFinishes,
    } as Animation
  },
})

Object.defineProperty(
  domWindow.HTMLElement.prototype,
  'getBoundingClientRect',
  {
    configurable: true,
    value: function getBoundingClientRect() {
      if ((this as HTMLElement).classList.contains('brand-entrance-logo')) {
        return new domWindow.DOMRect(100, 250, 100, 100)
      }
      if ((this as HTMLElement).dataset.slot === 'public-home-brand-logo') {
        return new domWindow.DOMRect(24, 18, 28, 28)
      }
      return new domWindow.DOMRect(0, 0, 0, 0)
    },
  }
)

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { BrandEntrance } = await import('../components/brand-entrance')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type RenderedEntrance = {
  container: HTMLDivElement
  root: ReturnType<typeof createRoot>
}

async function renderEntrance(): Promise<RenderedEntrance> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(
      <>
        <a data-slot='public-home-brand'>
          <span data-slot='public-home-brand-logo' />
          <span>uniRouters</span>
        </a>
        <BrandEntrance />
      </>
    )
  })

  return { container, root }
}

async function unmountEntrance(rendered: RenderedEntrance): Promise<void> {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

describe('uniRouters brand entrance', () => {
  beforeEach(() => {
    animationCalls.length = 0
    reduceMotion = false
    domWindow.sessionStorage.clear()
    document.body.replaceChildren()
  })

  after(() => {
    domWindow.close()
  })

  test('moves the first-visit brand lockup onto the responsive header logo', async () => {
    const rendered = await renderEntrance()
    const overlay = rendered.container.querySelector<HTMLElement>(
      '[data-slot="brand-entrance"]'
    )
    const moverAnimation = animationCalls.find((call) =>
      call.element.classList.contains('brand-entrance-mover')
    )

    assert.ok(overlay)
    assert.equal(overlay.getAttribute('aria-hidden'), 'true')
    assert.equal(overlay.classList.contains('pointer-events-none'), true)
    assert.ok(moverAnimation)
    assert.equal(moverAnimation.options.duration, 1800)
    assert.equal(
      moverAnimation.keyframes.at(-1)?.transform,
      'translate3d(-76px, -268px, 0) scale(0.28)'
    )
    assert.equal(
      domWindow.sessionStorage.getItem('unirouters:brand-entrance:v1'),
      '1'
    )

    await unmountEntrance(rendered)
  })

  test('does not replay after the home route remounts in the same tab', async () => {
    const firstRender = await renderEntrance()
    await unmountEntrance(firstRender)
    animationCalls.length = 0

    const secondRender = await renderEntrance()

    assert.equal(
      secondRender.container.querySelector('[data-slot="brand-entrance"]'),
      null
    )
    assert.equal(animationCalls.length, 0)

    await unmountEntrance(secondRender)
  })

  test('skips motion while still recording the visit for reduced-motion users', async () => {
    reduceMotion = true

    const rendered = await renderEntrance()

    assert.equal(
      rendered.container.querySelector('[data-slot="brand-entrance"]'),
      null
    )
    assert.equal(animationCalls.length, 0)
    assert.equal(
      domWindow.sessionStorage.getItem('unirouters:brand-entrance:v1'),
      '1'
    )

    await unmountEntrance(rendered)
  })
})
