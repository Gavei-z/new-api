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

import {
  CLAUDE_CODE_EXAMPLE,
  CODEX_CONFIG_EXAMPLE,
  DOCS_SECTIONS,
  PYTHON_SDK_EXAMPLE,
  RESPONSE_CURL_EXAMPLE,
  TYPESCRIPT_SDK_EXAMPLE,
  UNIROUTERS_API_BASE_URL,
  UNIROUTERS_DOCS_PATH,
} from '../constants'

describe('uniRouters documentation contract', () => {
  test('publishes the branded docs route and API base URL', () => {
    assert.equal(UNIROUTERS_DOCS_PATH, '/docs')
    assert.equal(UNIROUTERS_API_BASE_URL, 'https://unirouters.cc/v1')
  })

  test('keeps every public example on the uniRouters gateway', () => {
    const examples = [
      RESPONSE_CURL_EXAMPLE,
      PYTHON_SDK_EXAMPLE,
      TYPESCRIPT_SDK_EXAMPLE,
      CODEX_CONFIG_EXAMPLE,
      CLAUDE_CODE_EXAMPLE,
    ]

    for (const example of examples) {
      assert.ok(example.includes('unirouters'))
      assert.ok(!example.includes('docs.newapi.pro'))
      assert.ok(!example.includes('relayrouter.io'))
      assert.ok(!example.includes('onerouter.one'))
    }
  })

  test('exposes unique anchors for the responsive docs navigation', () => {
    const sectionIds = DOCS_SECTIONS.map((section) => section.id)

    assert.equal(new Set(sectionIds).size, sectionIds.length)
    assert.ok(sectionIds.includes('quick-start'))
    assert.ok(sectionIds.includes('coding-agents'))
    assert.ok(sectionIds.includes('troubleshooting'))
  })
})
