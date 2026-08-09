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
        'Copy to clipboard': 'Copy to clipboard',
        'Copied!': 'Copied!',
        Copied: 'Copied',
        'docs.actions.copyCode': 'Copy code',
        'docs.agents.title': 'Coding agents',
        'docs.agents.description': 'Configure supported coding agents.',
        'docs.agents.codex.title': 'Codex CLI',
        'docs.agents.codex.description': 'Configure Codex CLI.',
        'docs.agents.codex.installGuide': 'Install Codex CLI',
        'docs.agents.codex.environment': 'Export key and start Codex',
        'docs.agents.codex.envKeyNote': 'Keep the key in the environment.',
        'docs.agents.claudeCode.title': 'Claude Code',
        'docs.agents.claudeCode.description': 'Configure Claude Code.',
        'docs.agents.claudeCode.installGuide': 'Install Claude Code',
        'docs.agents.claudeCode.environment':
          'Set environment and start Claude Code',
        'docs.agents.claudeCode.baseUrlNote':
          'Claude Code appends the messages path.',
      },
    },
  },
})

const { CodingAgentsSection } =
  await import('../components/coding-reference-sections')
const {
  CLAUDE_CODE_EXAMPLE,
  CLAUDE_CODE_INSTALL_URL,
  CODEX_CONFIG_EXAMPLE,
  CODEX_INSTALL_URL,
  CODEX_KEY_EXAMPLE,
} = await import('../constants')
const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

describe('Coding agent setup guidance', () => {
  after(() => {
    domWindow.close()
  })

  test('links to both official installers and presents setup in runnable order', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(<CodingAgentsSection />)
    })

    const codexInstallLink = container.querySelector(
      `a[href="${CODEX_INSTALL_URL}"]`
    )
    const claudeInstallLink = container.querySelector(
      `a[href="${CLAUDE_CODE_INSTALL_URL}"]`
    )

    for (const link of [codexInstallLink, claudeInstallLink]) {
      assert.ok(link)
      assert.equal(link.getAttribute('target'), '_blank')
      assert.equal(link.getAttribute('rel'), 'noopener noreferrer')
    }

    const codeSamples = [...container.querySelectorAll('pre')].map(
      (sample) => sample.textContent
    )

    assert.deepEqual(codeSamples, [
      CODEX_CONFIG_EXAMPLE,
      CODEX_KEY_EXAMPLE,
      CLAUDE_CODE_EXAMPLE,
    ])

    await act(async () => root.unmount())
    container.remove()
  })
})
