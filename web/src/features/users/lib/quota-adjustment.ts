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
import { parseQuotaFromDollars } from '@/lib/format'

import type { QuotaAdjustMode } from '../types'

export function createQuotaAdjustmentIdempotencyKey(): string {
  if (
    typeof globalThis.crypto !== 'undefined' &&
    typeof globalThis.crypto.randomUUID === 'function'
  ) {
    return globalThis.crypto.randomUUID()
  }
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function isQuotaAdjustmentAmountValid(
  amount: string,
  mode: QuotaAdjustMode
): boolean {
  return getQuotaAdjustmentValue(amount, mode) !== null
}

export function getQuotaAdjustmentValue(
  amount: string,
  mode: QuotaAdjustMode
): number | null {
  if (amount.trim() === '') return null
  const amountValue = Number(amount)
  if (!Number.isFinite(amountValue) || amountValue < 0) return null

  const quotaValue = parseQuotaFromDollars(amountValue)
  if (!Number.isSafeInteger(quotaValue) || quotaValue < 0) return null
  if (amountValue > 0 && quotaValue === 0) return null
  if (mode !== 'override' && quotaValue <= 0) return null
  return quotaValue
}
