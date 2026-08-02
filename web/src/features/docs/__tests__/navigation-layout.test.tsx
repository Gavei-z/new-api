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
const i18next = (await import('i18next')).default
const { initReactI18next } = await import('react-i18next')

await i18next.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'docs.navigation.label': 'Documentation navigation',
        'docs.navigation.title': 'Documentation',
        'docs.navigation.baseUrl': 'API base URL',
        'docs.navigation.mobileTitle': 'Browse documentation',
        'docs.navigation.onThisPage': 'On this page',
        'docs.nav.overview': 'Overview',
        'docs.nav.quickStart': 'Quick start',
        'docs.nav.authentication': 'Authentication',
        'docs.nav.openaiSdks': 'OpenAI SDKs',
        'docs.nav.codingAgents': 'Coding agents',
        'docs.nav.apiReference': 'API reference',
        'docs.nav.troubleshooting': 'Troubleshooting',
      },
    },
  },
})

const { DesktopDocsNavigation, MobileDocsNavigation, OnThisPageNavigation } =
  await import('../components/docs-navigation')
const { DOCS_SECTIONS } = await import('../constants')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('Docs navigation layout', () => {
  after(() => {
    domWindow.close()
  })

  test('provides a compact mobile menu and desktop sidebars for the same sections', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <>
          <MobileDocsNavigation />
          <DesktopDocsNavigation />
          <OnThisPageNavigation />
        </>
      )
    })

    const mobileMenu = container.querySelector('details')
    const sidebars = container.querySelectorAll('aside')
    const mobileLinks = mobileMenu?.querySelectorAll('a')
    const desktopLinks = sidebars[0]?.querySelectorAll('a')

    assert.ok(mobileMenu)
    assert.equal(mobileMenu.classList.contains('lg:hidden'), true)
    assert.equal(sidebars.length, 2)
    assert.equal(sidebars[0]?.classList.contains('hidden'), true)
    assert.equal(sidebars[0]?.classList.contains('lg:block'), true)
    assert.equal(sidebars[1]?.classList.contains('xl:block'), true)
    assert.equal(mobileLinks?.length, DOCS_SECTIONS.length)
    assert.equal(desktopLinks?.length, DOCS_SECTIONS.length)

    for (const section of DOCS_SECTIONS) {
      assert.ok(
        mobileMenu.querySelector(`a[href="#${section.id}"]`),
        `missing mobile link for ${section.id}`
      )
      assert.ok(
        sidebars[0]?.querySelector(`a[href="#${section.id}"]`),
        `missing desktop link for ${section.id}`
      )
    }

    await act(async () => root.unmount())
    container.remove()
  })
})
