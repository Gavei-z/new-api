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
import type { TFunction } from 'i18next'

import {
  PAYMENT_TYPES,
  STRIPE_CHECKOUT_METHODS,
} from '@/features/wallet/constants'
import type { StripeCheckoutMethod } from '@/features/wallet/types'

import type { TeamStripeAmountPayload, TeamStripeTopupPayload } from './types'

export function getErrorMessage(error: unknown, fallback: string): string {
  return error instanceof Error && error.message ? error.message : fallback
}

export function getErrorCode(error: unknown): string | undefined {
  if (!error || typeof error !== 'object' || !('code' in error)) {
    return undefined
  }
  return typeof error.code === 'string' ? error.code : undefined
}

export function createIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `team-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function teamRoleLabel(t: TFunction, role: number): string {
  if (role >= 100) return t('Team Owner')
  if (role >= 10) return t('Team Administrator')
  return t('Team Member')
}

export function teamStatusLabel(t: TFunction, status: number): string {
  return status === 1 ? t('Enabled') : t('Disabled')
}

export function transactionTypeLabel(t: TFunction, type: string): string {
  const labels: Record<string, string> = {
    admin_grant: t('Administrator grant'),
    admin_deduction: t('Administrator deduction'),
    recharge: t('Wallet transfer'),
    stripe_topup: t('Stripe top-up'),
    stripe_recharge: t('Stripe top-up'),
    credit: t('Stripe top-up'),
    stripe_refund: t('Stripe refund'),
    reversal: t('Stripe refund or dispute reversal'),
    stripe_reversal: t('Stripe refund or dispute reversal'),
    stripe_dispute: t('Stripe dispute reversal'),
    restoration: t('Stripe dispute restoration'),
    stripe_restoration: t('Stripe dispute restoration'),
    stripe_dispute_restoration: t('Stripe dispute restoration'),
    release: t('Prepaid reserve release'),
    consume: t('Usage pre-charge'),
    settlement: t('Usage settlement'),
    refund: t('Usage refund'),
    task_adjustment: t('Task adjustment'),
    manual_correction: t('Manual correction'),
  }
  return labels[type] ?? type
}

/**
 * The authenticated team is resolved on the server. Keeping the request body
 * Neither checkout variant lets the browser select or override a team target.
 */
export function createTeamStripeTopupPayload(
  amount: number,
  checkoutMethod: StripeCheckoutMethod,
  quoteVersion?: string
): TeamStripeTopupPayload {
  const payload: TeamStripeTopupPayload = {
    amount,
    payment_method: PAYMENT_TYPES.STRIPE,
    checkout_method: checkoutMethod,
  }
  if (checkoutMethod === STRIPE_CHECKOUT_METHODS.WECHAT_PAY && quoteVersion) {
    payload.quote_version = quoteVersion
  }
  return payload
}

export function createTeamStripeAmountPayload(
  amount: number,
  checkoutMethod: StripeCheckoutMethod
): TeamStripeAmountPayload {
  return {
    amount,
    checkout_method: checkoutMethod,
  }
}
