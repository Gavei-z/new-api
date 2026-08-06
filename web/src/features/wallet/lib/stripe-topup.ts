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
import type { TopupInfo } from '../types'

export const DEFAULT_STRIPE_TOPUP_MIN = 2
export const DEFAULT_STRIPE_TOPUP_MAX = 50000
export const DEFAULT_STRIPE_TOPUP_PRESETS = [2, 5, 10, 50, 200, 500]

export type StripeTopupValidationError =
  | 'required'
  | 'integer'
  | 'minimum'
  | 'maximum'

export interface StripeTopupBounds {
  min: number
  max: number
}

export function getStripeTopupBounds(
  topupInfo: TopupInfo | null
): StripeTopupBounds {
  const configuredMin = Number(topupInfo?.stripe_min_topup)
  const configuredMax = Number(topupInfo?.stripe_max_topup)
  const min =
    Number.isSafeInteger(configuredMin) && configuredMin > 0
      ? Math.min(
          DEFAULT_STRIPE_TOPUP_MAX,
          Math.max(DEFAULT_STRIPE_TOPUP_MIN, configuredMin)
        )
      : DEFAULT_STRIPE_TOPUP_MIN
  const max =
    Number.isSafeInteger(configuredMax) && configuredMax >= min
      ? Math.min(configuredMax, DEFAULT_STRIPE_TOPUP_MAX)
      : DEFAULT_STRIPE_TOPUP_MAX

  return { min, max }
}

export function validateStripeTopupAmount(
  amount: number,
  bounds: StripeTopupBounds
): StripeTopupValidationError | null {
  if (!Number.isFinite(amount) || amount <= 0) {
    return 'required'
  }
  if (!Number.isSafeInteger(amount)) {
    return 'integer'
  }
  if (amount < bounds.min) {
    return 'minimum'
  }
  if (amount > bounds.max) {
    return 'maximum'
  }
  return null
}

export function getStripeTopupPresets(topupInfo: TopupInfo | null): number[] {
  const bounds = getStripeTopupBounds(topupInfo)
  const configuredPresets =
    topupInfo?.stripe_amount_options ?? topupInfo?.amount_options ?? []
  const candidates =
    configuredPresets.length > 0
      ? configuredPresets
      : DEFAULT_STRIPE_TOPUP_PRESETS

  return candidates.filter(
    (amount) => validateStripeTopupAmount(amount, bounds) === null
  )
}

export function formatUsdAmount(amount: number): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(amount)
}
