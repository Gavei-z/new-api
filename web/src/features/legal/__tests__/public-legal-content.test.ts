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
import {
  PRIVACY_SECTIONS,
  REFUND_SECTIONS,
  TERMS_SECTIONS,
  type PublicLegalSection,
} from '../public-legal-content'

type TranslationMap = Record<string, string>

const english = englishMessages.translation as TranslationMap
const chinese = chineseMessages.translation as TranslationMap

function collectSectionKeys(sections: readonly PublicLegalSection[]) {
  return sections.flatMap((section) => [
    section.titleKey,
    ...section.paragraphKeys,
    ...(section.bulletKeys ?? []),
  ])
}

describe('public legal content', () => {
  test('provides complete English and Chinese copy for every legal section', () => {
    const keys = [
      'legal.lastUpdated',
      'legal.terms.description',
      'legal.terms.contact',
      'legal.privacy.description',
      'legal.privacy.contact',
      'legal.refund.description',
      'legal.refund.contact',
      ...collectSectionKeys(TERMS_SECTIONS),
      ...collectSectionKeys(PRIVACY_SECTIONS),
      ...collectSectionKeys(REFUND_SECTIONS),
    ]

    for (const key of keys) {
      assert.ok(english[key]?.trim(), `missing English translation: ${key}`)
      assert.ok(chinese[key]?.trim(), `missing Chinese translation: ${key}`)
    }
  })

  test('covers API commerce, privacy, and refund review requirements', () => {
    assert.deepEqual(
      TERMS_SECTIONS.map((section) => section.id).filter((id) =>
        [
          'service',
          'acceptable-use',
          'pricing-and-billing',
          'delivery-refunds-cancellation',
        ].includes(id)
      ),
      [
        'service',
        'acceptable-use',
        'pricing-and-billing',
        'delivery-refunds-cancellation',
      ]
    )
    assert.ok(PRIVACY_SECTIONS.some((section) => section.id === 'api-content'))
    assert.ok(PRIVACY_SECTIONS.some((section) => section.id === 'payments'))
    assert.ok(
      REFUND_SECTIONS.some((section) => section.id === 'digital-delivery')
    )
    assert.ok(REFUND_SECTIONS.some((section) => section.id === 'cancellation'))
  })

  test('uses the approved public support mailbox', () => {
    assert.equal(UNIROUTERS_SUPPORT_EMAIL, 'dealerjzad@gmail.com')
  })

  test('states the unused-balance refund, processing fee, and enforcement rules', () => {
    for (const messages of [english, chinese]) {
      assert.match(messages['legal.terms.refunds.p2'], /1%–3%/)
      assert.ok(messages['legal.terms.refunds.p3'].trim())
      assert.match(messages['legal.refund.eligible.b2'], /1%–3%/)
      assert.ok(messages['legal.refund.nonRefundable.b3'].trim())
    }

    assert.match(english['legal.terms.refunds.p2'], /unused purchased balance/)
    assert.match(chinese['legal.terms.refunds.p2'], /未使用的已购余额/)
    assert.match(english['legal.terms.refunds.p3'], /suspended or terminated/)
    assert.match(chinese['legal.terms.refunds.p3'], /封停或终止/)
  })
})
