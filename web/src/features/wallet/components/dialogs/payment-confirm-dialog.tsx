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
import { Loader2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Skeleton } from '@/components/ui/skeleton'
import { formatLocalCurrencyAmount } from '@/lib/currency'

import { DEFAULT_DISCOUNT_RATE, STRIPE_CHECKOUT_METHODS } from '../../constants'
import {
  formatCurrency,
  formatCnyAmount,
  formatUsdAmount,
  getPaymentIcon,
  isStripePayment,
} from '../../lib'
import type {
  PaymentMethod,
  StripeCheckoutMethod,
  StripeFxQuote,
} from '../../types'
import { StripeFxQuoteDetails } from '../stripe-fx-quote-details'

interface PaymentConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  topupAmount: number
  paymentAmount: number
  paymentMethod: Readonly<PaymentMethod> | undefined
  calculating: boolean
  processing: boolean
  discountRate?: number
  usdExchangeRate?: number
  currencyCode?: 'USD' | 'CNY'
  stripeCheckoutMethod?: StripeCheckoutMethod
  stripeFxQuote?: Readonly<StripeFxQuote>
}

export function PaymentConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  topupAmount,
  paymentAmount,
  paymentMethod,
  calculating,
  processing,
  discountRate = DEFAULT_DISCOUNT_RATE,
  usdExchangeRate = 1,
  currencyCode,
  stripeCheckoutMethod,
  stripeFxQuote,
}: PaymentConfirmDialogProps) {
  const { t } = useTranslation()
  const hasDiscount = discountRate > 0 && discountRate < 1 && paymentAmount > 0
  const originalAmount = hasDiscount ? paymentAmount / discountRate : 0
  const discountAmount = hasDiscount ? originalAmount - paymentAmount : 0
  const showPaymentProvider = !isStripePayment(paymentMethod?.type ?? '')
  const isStripeCheckout = isStripePayment(paymentMethod?.type ?? '')
  const isWeChatPay =
    stripeCheckoutMethod === STRIPE_CHECKOUT_METHODS.WECHAT_PAY
  let creditAmountText = formatLocalCurrencyAmount(
    topupAmount * usdExchangeRate,
    {
      digitsLarge: 2,
      digitsSmall: 2,
      abbreviate: false,
    }
  )
  if (isStripeCheckout) {
    creditAmountText = `${formatUsdAmount(topupAmount)} USD`
  }

  let paymentAmountText = formatCurrency(paymentAmount)
  let originalAmountText = formatCurrency(originalAmount)
  let discountAmountText = formatCurrency(discountAmount)
  if (currencyCode === 'USD') {
    paymentAmountText = `${formatUsdAmount(paymentAmount)} USD`
    originalAmountText = `${formatUsdAmount(originalAmount)} USD`
    discountAmountText = `${formatUsdAmount(discountAmount)} USD`
  } else if (currencyCode === 'CNY') {
    const cnyPaymentAmount = stripeFxQuote
      ? stripeFxQuote.pay_amount_minor / 100
      : paymentAmount
    paymentAmountText = `${formatCnyAmount(cnyPaymentAmount)} CNY`
    originalAmountText = `${formatCnyAmount(originalAmount)} CNY`
    discountAmountText = `${formatCnyAmount(discountAmount)} CNY`
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-md'>
        <AlertDialogHeader>
          <AlertDialogTitle className='text-xl font-semibold'>
            {t('Confirm Payment')}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t('Review your payment details')}
          </AlertDialogDescription>
        </AlertDialogHeader>

        <div className='space-y-3 py-3 sm:space-y-4 sm:py-4'>
          <div className='flex items-center justify-between'>
            <span className='text-muted-foreground text-sm'>
              {isStripeCheckout ? t('USD credit amount') : t('Topup Amount')}
            </span>
            <span className='text-lg font-semibold'>{creditAmountText}</span>
          </div>

          <div
            data-slot='payment-confirm-quote'
            aria-live='polite'
            className='flex items-center justify-between'
          >
            <span className='text-muted-foreground text-sm'>
              {t('You Pay')}
            </span>
            {calculating ? (
              <Skeleton className='h-6 w-24' />
            ) : (
              <div className='flex items-baseline gap-2'>
                <span className='text-2xl font-semibold'>
                  {paymentAmountText}
                </span>
                {hasDiscount && (
                  <span className='text-muted-foreground text-sm line-through'>
                    {originalAmountText}
                  </span>
                )}
              </div>
            )}
          </div>

          {hasDiscount && !calculating && (
            <div className='bg-muted/50 rounded-lg p-3'>
              <div className='flex items-center justify-between text-sm'>
                <span className='text-muted-foreground'>{t('You save')}</span>
                <span className='font-semibold text-green-600'>
                  {discountAmountText}
                </span>
              </div>
            </div>
          )}

          {showPaymentProvider && (
            <div className='border-t pt-4'>
              <div className='flex items-center justify-between'>
                <span className='text-muted-foreground text-sm'>
                  {t('Payment Method')}
                </span>
                <div className='flex items-center gap-2'>
                  {getPaymentIcon(
                    paymentMethod?.type,
                    'h-4 w-4',
                    paymentMethod?.icon,
                    paymentMethod?.name
                  )}
                  <span className='font-medium'>{paymentMethod?.name}</span>
                </div>
              </div>
            </div>
          )}

          {isWeChatPay && (
            <div className='border-t pt-4'>
              <div className='flex items-center justify-between gap-4'>
                <span className='text-muted-foreground text-sm'>
                  {t('Payment Method')}
                </span>
                <div className='flex items-center gap-2 font-medium text-[#067a3b] dark:text-[#41d17c]'>
                  {getPaymentIcon('wxpay', 'h-4 w-4')}
                  <span>{t('WeChat Pay')}</span>
                </div>
              </div>
              <p className='text-muted-foreground mt-2 text-xs leading-5'>
                {t(
                  'WeChat checkout displays the final amount in CNY; your wallet is credited with the selected USD amount.'
                )}
              </p>
              {stripeFxQuote && (
                <StripeFxQuoteDetails
                  quote={stripeFxQuote}
                  className='mt-3 rounded-lg bg-[#07C160]/5 p-3'
                />
              )}
            </div>
          )}
        </div>

        <AlertDialogFooter className='grid grid-cols-2 gap-2 sm:flex'>
          <AlertDialogCancel disabled={processing}>
            {t('Cancel')}
          </AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm} disabled={processing}>
            {processing && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
            {t('Confirm Payment')}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
