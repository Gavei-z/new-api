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
  convertDetectedLanguage,
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
  toIntlLocale,
} from '../languages'

describe('interface language availability', () => {
  test('offers only Simplified Chinese and English', () => {
    assert.deepEqual(INTERFACE_LANGUAGE_OPTIONS, [
      { code: 'zhCN', label: '简体中文' },
      { code: 'en', label: 'English' },
    ])
  })

  test('maps every Chinese locale variant to Simplified Chinese', () => {
    for (const language of ['zh', 'zh-CN', 'zh_TW', 'zh-Hant-HK']) {
      assert.equal(normalizeInterfaceLanguage(language), 'zhCN')
      assert.equal(convertDetectedLanguage(language), 'zhCN')
    }
    assert.equal(toIntlLocale('zhCN'), 'zh-CN')
  })

  test('falls back to English for hidden or unknown interface languages', () => {
    for (const language of ['fr', 'ja', 'ru', 'vi', '', undefined]) {
      assert.equal(normalizeInterfaceLanguage(language), 'en')
    }
    assert.equal(toIntlLocale('en'), 'en')
  })
})
