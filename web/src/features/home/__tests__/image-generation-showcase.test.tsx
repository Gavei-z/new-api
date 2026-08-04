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

import englishMessages from '../../../i18n/locales/en.json'
import chineseMessages from '../../../i18n/locales/zh.json'

const domWindow = new Window()
const domGlobals = [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'Node',
  'Element',
  'Event',
  'KeyboardEvent',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

Object.defineProperty(domWindow, 'matchMedia', {
  configurable: true,
  value: () => ({
    matches: true,
    media: '(prefers-reduced-motion: reduce)',
    onchange: null,
    addListener: () => undefined,
    removeListener: () => undefined,
    addEventListener: () => undefined,
    removeEventListener: () => undefined,
    dispatchEvent: () => false,
  }),
})

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const i18next = (await import('i18next')).default
const { initReactI18next } = await import('react-i18next')

await i18next.use(initReactI18next).init({
  lng: 'en',
  fallbackLng: false,
  resources: {
    en: englishMessages,
    zhCN: chineseMessages,
  },
  interpolation: { escapeValue: false },
})

const { ImageGenerationCarousel } =
  await import('../components/image-generation-carousel')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

type RenderedCarousel = {
  container: HTMLDivElement
  root: ReturnType<typeof createRoot>
}

async function renderCarousel(): Promise<RenderedCarousel> {
  const container = document.createElement('div')
  document.body.append(container)
  const root = createRoot(container)

  await act(async () => {
    root.render(<ImageGenerationCarousel />)
  })

  return { container, root }
}

async function unmountCarousel(rendered: RenderedCarousel) {
  await act(async () => rendered.root.unmount())
  rendered.container.remove()
}

function getActiveSlide(container: HTMLElement) {
  return container.querySelector('figure[data-active="true"]')
}

describe('uniRouters image generation carousel', () => {
  beforeEach(async () => {
    await i18next.changeLanguage('en')
  })

  after(() => {
    domWindow.close()
  })

  test('renders five concepts with accessible carousel controls', async () => {
    const rendered = await renderCarousel()
    const carousel = rendered.container.querySelector(
      '[aria-roledescription="carousel"]'
    )
    const images = rendered.container.querySelectorAll('img')
    const previousButton = rendered.container.querySelector<HTMLButtonElement>(
      'button[aria-label="Previous image"]'
    )
    const nextButton = rendered.container.querySelector<HTMLButtonElement>(
      'button[aria-label="Next image"]'
    )

    assert.ok(carousel)
    assert.equal(images.length, 5)
    assert.match(getActiveSlide(rendered.container)?.textContent ?? '', /Night/)
    assert.ok(previousButton)
    assert.ok(nextButton)

    await unmountCarousel(rendered)
  })

  test('moves to the requested concept with arrow and dot controls', async () => {
    const rendered = await renderCarousel()
    const nextButton = rendered.container.querySelector<HTMLButtonElement>(
      'button[aria-label="Next image"]'
    )
    const fifthDot = rendered.container.querySelector<HTMLButtonElement>(
      'button[aria-label="Go to image 5"]'
    )

    assert.ok(nextButton)
    await act(async () => nextButton.click())
    assert.match(
      getActiveSlide(rendered.container)?.textContent ?? '',
      /shared future on Mars/
    )

    assert.ok(fifthDot)
    await act(async () => fifthDot.click())
    assert.match(
      getActiveSlide(rendered.container)?.textContent ?? '',
      /beyond the horizon/
    )
    assert.equal(fifthDot.getAttribute('aria-current'), 'true')

    await unmountCarousel(rendered)
  })

  test('moves to the adjacent concept with keyboard arrow keys', async () => {
    const rendered = await renderCarousel()
    const carousel = rendered.container.querySelector(
      '[aria-roledescription="carousel"]'
    )

    assert.ok(carousel)
    await act(async () => {
      carousel.dispatchEvent(
        new KeyboardEvent('keydown', { key: 'ArrowRight', bubbles: true })
      )
    })
    assert.match(
      getActiveSlide(rendered.container)?.textContent ?? '',
      /shared future on Mars/
    )

    await unmountCarousel(rendered)
  })

  test('updates visible copy and accessible controls for Chinese users', async () => {
    await i18next.changeLanguage('zhCN')
    const rendered = await renderCarousel()

    assert.match(
      getActiveSlide(rendered.container)?.textContent ?? '',
      /大排档的跨界碰杯/
    )
    assert.ok(
      rendered.container.querySelector('button[aria-label="下一张图片"]')
    )
    assert.ok(
      rendered.container.querySelector('button[aria-label="切换到第 5 张图片"]')
    )

    await unmountCarousel(rendered)
  })
})
