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
  UNIROUTERS_HOME_DOCS_PATH,
  UNIROUTERS_HOME_NAV_LINKS,
} from '../constants'

describe('uniRouters home navigation contract', () => {
  test('keeps every home documentation entry on the branded docs route', () => {
    const docsLink = UNIROUTERS_HOME_NAV_LINKS.find(
      (link) => link.title === 'Docs'
    )

    assert.equal(UNIROUTERS_HOME_DOCS_PATH, '/docs')
    assert.equal(docsLink?.href, '/docs')
    assert.notEqual(
      docsLink && 'external' in docsLink ? docsLink.external : false,
      true
    )
  })
})
