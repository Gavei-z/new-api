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
import i18next from 'i18next'
import { useState, useCallback, useRef } from 'react'
import { toast } from 'sonner'

import {
  calculateAmount,
  calculateStripeAmount,
  calculateWaffoAmount,
  calculateWaffoPancakeAmount,
  requestPayment,
  requestStripePayment,
  isApiSuccess,
} from '../api'
import { STRIPE_CHECKOUT_METHODS } from '../constants'
import {
  isStripePayment,
  isIntegerTopupAmount,
  isWaffoPayment,
  isWaffoPancakePayment,
  submitPaymentForm,
  createStripeTopupPayload,
  parsePaymentAmountQuote,
} from '../lib'
import type {
  AmountRequest,
  AmountResponse,
  PaymentAmountQuote,
  StripeCheckoutMethod,
} from '../types'

// ============================================================================
// Payment Hook
// ============================================================================

type AmountCalculator = (request: AmountRequest) => Promise<AmountResponse>

export interface PaymentAmountCalculators {
  regular: AmountCalculator
  stripe: AmountCalculator
  waffo: AmountCalculator
  waffoPancake: AmountCalculator
}

const defaultPaymentAmountCalculators: PaymentAmountCalculators = {
  regular: calculateAmount,
  stripe: calculateStripeAmount,
  waffo: calculateWaffoAmount,
  waffoPancake: calculateWaffoPancakeAmount,
}

export type CheckoutQuoteResult =
  | Readonly<{ status: 'success'; quote: PaymentAmountQuote }>
  | Readonly<{ status: 'invalid' }>
  | Readonly<{ status: 'superseded' }>

async function requestPaymentAmountResponse(
  topupAmount: number,
  paymentType: string,
  stripeCheckoutMethod: StripeCheckoutMethod,
  calculators: PaymentAmountCalculators
): Promise<AmountResponse | null> {
  if (!isIntegerTopupAmount(topupAmount)) return null

  let calculator = calculators.regular
  if (isStripePayment(paymentType)) {
    calculator = calculators.stripe
  } else if (isWaffoPayment(paymentType)) {
    calculator = calculators.waffo
  } else if (isWaffoPancakePayment(paymentType)) {
    calculator = calculators.waffoPancake
  }

  const request: AmountRequest = { amount: topupAmount }
  if (isStripePayment(paymentType)) {
    request.checkout_method = stripeCheckoutMethod
  }

  const response = await calculator(request)
  return isApiSuccess(response) ? response : null
}

export async function requestPaymentAmount(
  topupAmount: number,
  paymentType: string,
  stripeCheckoutMethod: StripeCheckoutMethod = STRIPE_CHECKOUT_METHODS.STANDARD,
  calculators: PaymentAmountCalculators = defaultPaymentAmountCalculators
): Promise<number> {
  const response = await requestPaymentAmountResponse(
    topupAmount,
    paymentType,
    stripeCheckoutMethod,
    calculators
  )
  if (!response || typeof response.data !== 'string') return 0

  const parsedAmount = Number(response.data)
  return Number.isFinite(parsedAmount) && parsedAmount > 0 ? parsedAmount : 0
}

export async function requestCheckoutQuote(
  topupAmount: number,
  paymentType: string,
  stripeCheckoutMethod: StripeCheckoutMethod = STRIPE_CHECKOUT_METHODS.STANDARD,
  calculators: PaymentAmountCalculators = defaultPaymentAmountCalculators
): Promise<PaymentAmountQuote | null> {
  const response = await requestPaymentAmountResponse(
    topupAmount,
    paymentType,
    stripeCheckoutMethod,
    calculators
  )
  if (!response) return null

  return parsePaymentAmountQuote(response, stripeCheckoutMethod)
}

export function usePayment(
  calculators: PaymentAmountCalculators = defaultPaymentAmountCalculators
) {
  const [amount, setAmount] = useState<number>(0)
  const [calculating, setCalculating] = useState(false)
  const [processing, setProcessing] = useState(false)
  const amountRequestIdRef = useRef(0)

  // Calculate payment amount
  const calculatePaymentAmount = useCallback(
    async (
      topupAmount: number,
      paymentType: string,
      stripeCheckoutMethod: StripeCheckoutMethod = STRIPE_CHECKOUT_METHODS.STANDARD
    ) => {
      const requestId = ++amountRequestIdRef.current
      try {
        setCalculating(true)
        const calculatedAmount = await requestPaymentAmount(
          topupAmount,
          paymentType,
          stripeCheckoutMethod,
          calculators
        )
        if (requestId === amountRequestIdRef.current) {
          setAmount(calculatedAmount)
        }
        return calculatedAmount
      } catch {
        if (requestId === amountRequestIdRef.current) {
          setAmount(0)
        }
        return 0
      } finally {
        if (requestId === amountRequestIdRef.current) {
          setCalculating(false)
        }
      }
    },
    [calculators]
  )

  // Checkout quotes are immutable confirmation inputs. They invalidate any
  // older background preview but never write CNY or method-specific values
  // into the form's shared preview amount.
  const quotePaymentAmount = useCallback(
    async (
      topupAmount: number,
      paymentType: string,
      stripeCheckoutMethod: StripeCheckoutMethod = STRIPE_CHECKOUT_METHODS.STANDARD
    ) => {
      const requestId = ++amountRequestIdRef.current
      setCalculating(false)
      try {
        const quote = await requestCheckoutQuote(
          topupAmount,
          paymentType,
          stripeCheckoutMethod,
          calculators
        )
        if (requestId !== amountRequestIdRef.current) {
          return { status: 'superseded' } as const
        }
        if (!quote) return { status: 'invalid' } as const
        return { status: 'success', quote } as const
      } catch {
        if (requestId !== amountRequestIdRef.current) {
          return { status: 'superseded' } as const
        }
        return { status: 'invalid' } as const
      }
    },
    [calculators]
  )

  const replacePaymentAmount = useCallback((nextAmount: number) => {
    amountRequestIdRef.current += 1
    setCalculating(false)
    setAmount(nextAmount)
  }, [])

  // Process payment
  const processPayment = useCallback(
    async (
      topupAmount: number,
      paymentType: string,
      stripeCheckoutMethod?: StripeCheckoutMethod,
      quoteVersion?: string
    ) => {
      try {
        setProcessing(true)

        if (!isIntegerTopupAmount(topupAmount)) {
          toast.error(i18next.t('Amount must be a whole number'))
          return false
        }

        const isStripe = isStripePayment(paymentType)

        const response = isStripe
          ? await requestStripePayment(
              createStripeTopupPayload(
                topupAmount,
                stripeCheckoutMethod ?? 'standard',
                quoteVersion
              )
            )
          : await requestPayment({
              amount: topupAmount,
              payment_method: paymentType,
            })

        if (!isApiSuccess(response)) {
          if (response.code === 'stripe_fx_quote_expired') {
            toast.error(
              i18next.t(
                'The exchange rate quote has expired. Please close this dialog and choose WeChat Pay again.'
              )
            )
          } else if (response.code === 'stripe_fx_rate_unavailable') {
            toast.error(
              i18next.t(
                'The exchange rate is temporarily unavailable. Please try again later.'
              )
            )
          } else {
            toast.error(response.message || i18next.t('Payment request failed'))
          }
          return false
        }

        // Handle Stripe payment
        if (isStripe && response.data?.pay_link) {
          window.location.assign(response.data.pay_link as string)
          toast.success(i18next.t('Redirecting to payment page...'))
          return true
        }

        // Handle non-Stripe payment
        if (!isStripe && response.data) {
          const url = (response as unknown as { url?: string }).url
          if (url) {
            submitPaymentForm(url, response.data)
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
        }

        return false
      } catch {
        toast.error(i18next.t('Payment request failed'))
        return false
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  return {
    amount,
    calculating,
    processing,
    calculatePaymentAmount,
    quotePaymentAmount,
    processPayment,
    setAmount: replacePaymentAmount,
  }
}
