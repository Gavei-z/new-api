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

import englishMessages from '../../../i18n/locales/en.json'
import chineseMessages from '../../../i18n/locales/zh.json'
import { UNIROUTERS_SUPPORT_EMAIL } from '../../../lib/constants'
import { PUBLIC_FOOTER_COLUMNS } from '../components/footer-links'

const english = englishMessages.translation as Record<string, string>
const chinese = chineseMessages.translation as Record<string, string>

describe('public footer navigation', () => {
  test('exposes permanent terms, privacy, and refund destinations', () => {
    const links = PUBLIC_FOOTER_COLUMNS.flatMap((column) => column.links)
    const legalDestinations = links
      .filter((link) =>
        ['/terms', '/privacy', '/refund-policy'].includes(link.href)
      )
      .map((link) => link.href)

    assert.deepEqual(legalDestinations, [
      '/terms',
      '/privacy',
      '/refund-policy',
    ])
  })

  test('provides four vertical groups and direct support contact', () => {
    assert.equal(PUBLIC_FOOTER_COLUMNS.length, 4)
    assert.deepEqual(
      PUBLIC_FOOTER_COLUMNS.map((column) => column.title),
      [
        'footer.columns.product.title',
        'footer.columns.resources.title',
        'footer.columns.support.title',
        'footer.columns.legal.title',
      ]
    )

    const supportLinks = PUBLIC_FOOTER_COLUMNS[2].links
    assert.ok(
      supportLinks.some((link) =>
        link.href.startsWith(`mailto:${UNIROUTERS_SUPPORT_EMAIL}`)
      )
    )
  })

  test('translates every footer group and link in English and Chinese', () => {
    const keys = PUBLIC_FOOTER_COLUMNS.flatMap((column) => [
      column.title,
      ...column.links.map((link) => link.text),
    ])

    for (const key of keys) {
      assert.ok(english[key]?.trim())
      assert.ok(chinese[key]?.trim())
    }
  })
})
