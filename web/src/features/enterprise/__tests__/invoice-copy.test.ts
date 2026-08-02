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
import {
  ENTERPRISE_INVOICE_COPY_KEY,
  ENTERPRISE_QUOTE_BENEFITS,
} from '../constants'

describe('enterprise invoice copy', () => {
  test('shows corporate invoicing support in English and Chinese', () => {
    assert.ok(ENTERPRISE_QUOTE_BENEFITS.includes(ENTERPRISE_INVOICE_COPY_KEY))
    assert.equal(
      englishMessages.translation[ENTERPRISE_INVOICE_COPY_KEY],
      'Corporate invoicing supported'
    )
    assert.equal(
      chineseMessages.translation[ENTERPRISE_INVOICE_COPY_KEY],
      '支持对公开票'
    )
  })
})
