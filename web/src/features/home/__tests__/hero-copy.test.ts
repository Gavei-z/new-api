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
import { UNIROUTERS_HOME_HERO_COPY } from '../constants'

describe('uniRouters home hero copy', () => {
  test('keeps the English hero copy polished and unchanged by Chinese localization', () => {
    assert.equal(
      englishMessages.translation[UNIROUTERS_HOME_HERO_COPY.taglineKey],
      'The Foundation for AI Agents'
    )
    assert.equal(
      englishMessages.translation[UNIROUTERS_HOME_HERO_COPY.headlineKey],
      'One Unified Gateway for Multi Models.'
    )
  })

  test('uses the requested Chinese hero copy', () => {
    assert.equal(
      chineseMessages.translation[UNIROUTERS_HOME_HERO_COPY.taglineKey],
      '为AI智能体提供基座'
    )
    assert.equal(
      chineseMessages.translation[UNIROUTERS_HOME_HERO_COPY.headlineKey],
      '一个网关key，使用多种主流模型。'
    )
  })
})
