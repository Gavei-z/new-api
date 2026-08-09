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
import { STRIPE_CHECKOUT_METHODS } from '../constants'
import type {
  AmountResponse,
  PaymentAmountQuote,
  StripeCheckoutMethod,
  StripeFxQuote,
} from '../types'

const DECIMAL_PATTERN = /^\d+(?:\.\d+)?$/
const PRICING_DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/

function asRecord(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return null
  return value as Record<string, unknown>
}

function asPositiveSafeInteger(value: unknown): number | undefined {
  const parsed = typeof value === 'string' ? Number(value) : value
  if (
    typeof parsed !== 'number' ||
    !Number.isSafeInteger(parsed) ||
    parsed <= 0
  ) {
    return undefined
  }
  return parsed
}

function asPositiveDecimalString(value: unknown): string | undefined {
  if (typeof value !== 'string' && typeof value !== 'number') return undefined
  const normalized = String(value).trim()
  if (!DECIMAL_PATTERN.test(normalized)) return undefined
  const parsed = Number(normalized)
  if (!Number.isFinite(parsed) || parsed <= 0) return undefined
  return normalized
}

export function normalizeStripeFxQuote(value: unknown): StripeFxQuote | null {
  const quote = asRecord(value)
  if (!quote) return null

  const payAmountMinor = asPositiveSafeInteger(quote.pay_amount_minor)
  const currency =
    typeof quote.currency === 'string' ? quote.currency.toUpperCase() : ''
  const exchangeRate = asPositiveDecimalString(quote.exchange_rate)
  const source = typeof quote.source === 'string' ? quote.source.trim() : ''
  const pricingDate =
    typeof quote.pricing_date === 'string' ? quote.pricing_date.trim() : ''
  const quoteVersion =
    typeof quote.quote_version === 'string' ? quote.quote_version.trim() : ''
  const sourcePublishedAt =
    quote.source_published_at === undefined ||
    quote.source_published_at === null
      ? undefined
      : asPositiveSafeInteger(quote.source_published_at)

  if (
    payAmountMinor === undefined ||
    currency !== 'CNY' ||
    exchangeRate === undefined ||
    source === '' ||
    !PRICING_DATE_PATTERN.test(pricingDate) ||
    quoteVersion === '' ||
    quoteVersion.length > 512 ||
    (quote.source_published_at !== undefined && sourcePublishedAt === undefined)
  ) {
    return null
  }

  return Object.freeze({
    pay_amount_minor: payAmountMinor,
    currency: 'CNY',
    exchange_rate: exchangeRate,
    source,
    pricing_date: pricingDate,
    source_published_at: sourcePublishedAt,
    quote_version: quoteVersion,
  })
}

export function parsePaymentAmountQuote(
  response: AmountResponse,
  checkoutMethod: StripeCheckoutMethod
): PaymentAmountQuote | null {
  if (typeof response.data !== 'string' || response.data.trim() === '') {
    return null
  }

  const parsedAmount = Number(response.data)
  if (!Number.isFinite(parsedAmount) || parsedAmount <= 0) return null

  if (checkoutMethod !== STRIPE_CHECKOUT_METHODS.WECHAT_PAY) {
    return Object.freeze({ amount: parsedAmount })
  }

  const stripeFxQuote = normalizeStripeFxQuote(response.quote)
  if (!stripeFxQuote) return null

  const quotedMinorAmount = Math.round(parsedAmount * 100)
  if (quotedMinorAmount !== stripeFxQuote.pay_amount_minor) return null

  return Object.freeze({
    amount: stripeFxQuote.pay_amount_minor / 100,
    stripeFxQuote,
  })
}
