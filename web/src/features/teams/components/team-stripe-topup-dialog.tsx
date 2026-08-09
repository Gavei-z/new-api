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
import { useMutation, useQuery } from '@tanstack/react-query'
import { CreditCard, Loader2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  PAYMENT_TYPES,
  STRIPE_CHECKOUT_METHODS,
} from '@/features/wallet/constants'
import {
  formatCnyAmount,
  formatUsdAmount,
  getPaymentIcon,
  getStripeTopupBounds,
  getStripeTopupPresets,
  validateStripeTopupAmount,
} from '@/features/wallet/lib'
import type { StripeCheckoutMethod, TopupInfo } from '@/features/wallet/types'
import { cn } from '@/lib/utils'

import {
  calculateCurrentTeamStripeAmount,
  requestCurrentTeamStripePayment,
} from '../api'
import { getErrorMessage } from '../lib'

interface TeamStripeTopupDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  topupInfo: TopupInfo | null
}

export function TeamStripeTopupDialog(props: TeamStripeTopupDialogProps) {
  const { t } = useTranslation()
  const bounds = useMemo(
    () => getStripeTopupBounds(props.topupInfo),
    [props.topupInfo]
  )
  const presets = useMemo(
    () => getStripeTopupPresets(props.topupInfo),
    [props.topupInfo]
  )
  const initialAmount = presets[0] ?? bounds.min
  const [amountInput, setAmountInput] = useState(String(initialAmount))
  const amount = Number(amountInput)
  const validation = validateStripeTopupAmount(amount, bounds)

  useEffect(() => {
    if (!props.open) {
      setAmountInput(String(initialAmount))
    }
  }, [initialAmount, props.open])

  const standardAmountQuery = useQuery({
    queryKey: ['team-stripe-amount', amount, STRIPE_CHECKOUT_METHODS.STANDARD],
    queryFn: () =>
      calculateCurrentTeamStripeAmount(
        amount,
        STRIPE_CHECKOUT_METHODS.STANDARD
      ),
    enabled: props.open && validation === null,
    retry: false,
  })
  const weChatAmountQuery = useQuery({
    queryKey: [
      'team-stripe-amount',
      amount,
      STRIPE_CHECKOUT_METHODS.WECHAT_PAY,
    ],
    queryFn: () =>
      calculateCurrentTeamStripeAmount(
        amount,
        STRIPE_CHECKOUT_METHODS.WECHAT_PAY
      ),
    enabled:
      props.open &&
      validation === null &&
      props.topupInfo?.enable_stripe_wechat_pay === true,
    retry: false,
  })

  const checkoutMutation = useMutation({
    mutationFn: requestCurrentTeamStripePayment,
    onSuccess: (checkout) => {
      if (!checkout.pay_link) {
        toast.error(t('Payment request failed'))
        return
      }
      window.location.assign(checkout.pay_link)
    },
    onError: (error) => {
      toast.error(getErrorMessage(error, t('Payment request failed')))
    },
  })

  let validationMessage = ''
  if (validation === 'integer') {
    validationMessage = t('Enter a whole USD amount.')
  } else if (validation === 'minimum') {
    validationMessage = t('Minimum top-up is ${{amount}} USD.', {
      amount: bounds.min,
    })
  } else if (validation === 'maximum') {
    validationMessage = t('Maximum top-up is ${{amount}} USD.', {
      amount: bounds.max,
    })
  } else if (validation === 'required') {
    validationMessage = t('Enter a valid USD amount.')
  }

  const standardQuotedAmount = Number(standardAmountQuery.data)
  const weChatQuotedAmount = Number(weChatAmountQuery.data)
  const hasValidStandardQuote =
    standardAmountQuery.isSuccess &&
    !standardAmountQuery.isFetching &&
    Number.isFinite(standardQuotedAmount) &&
    standardQuotedAmount > 0
  const hasValidWeChatQuote =
    weChatAmountQuery.isSuccess &&
    !weChatAmountQuery.isFetching &&
    Number.isFinite(weChatQuotedAmount) &&
    weChatQuotedAmount > 0
  const canSubmitBase =
    validation === null &&
    !checkoutMutation.isPending &&
    props.topupInfo?.enable_stripe_topup === true
  const canSubmitStandard = canSubmitBase && hasValidStandardQuote
  const canSubmitWeChat =
    canSubmitBase &&
    props.topupInfo?.enable_stripe_wechat_pay === true &&
    hasValidWeChatQuote

  const submit = (checkoutMethod: StripeCheckoutMethod) => {
    const canSubmit =
      checkoutMethod === STRIPE_CHECKOUT_METHODS.WECHAT_PAY
        ? canSubmitWeChat
        : canSubmitStandard
    if (!canSubmit) return
    checkoutMutation.mutate({ amount, checkoutMethod })
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Add Team Funds')}</DialogTitle>
          <DialogDescription>
            {t(
              'Choose a USD amount. Credits are added to the shared team balance after payment.'
            )}
          </DialogDescription>
        </DialogHeader>

        <div className='space-y-5'>
          <div className='space-y-2'>
            <Label>{t('Amount (USD)')}</Label>
            <div className='grid grid-cols-3 gap-2 sm:grid-cols-6'>
              {presets.map((preset) => (
                <Button
                  key={preset}
                  type='button'
                  variant='outline'
                  className={cn(
                    'h-11 px-2',
                    amount === preset &&
                      'border-foreground bg-foreground/5 dark:bg-foreground/10'
                  )}
                  aria-pressed={amount === preset}
                  onClick={() => setAmountInput(String(preset))}
                >
                  ${preset}
                </Button>
              ))}
            </div>
          </div>

          <div className='space-y-2'>
            <Label htmlFor='team-stripe-amount'>
              {t('Custom Amount (USD)')}
            </Label>
            <Input
              id='team-stripe-amount'
              type='number'
              inputMode='numeric'
              min={bounds.min}
              max={bounds.max}
              step={1}
              value={amountInput}
              onChange={(event) => setAmountInput(event.target.value)}
              aria-invalid={validationMessage ? 'true' : undefined}
              aria-describedby='team-stripe-amount-guidance'
            />
            <p
              id='team-stripe-amount-guidance'
              className={cn(
                'text-xs',
                validationMessage ? 'text-destructive' : 'text-muted-foreground'
              )}
            >
              {validationMessage ||
                t('Whole USD amounts from ${{min}} to ${{max}}.', {
                  min: bounds.min,
                  max: bounds.max,
                })}
            </p>
          </div>

          <div
            data-slot='team-stripe-quote-summary'
            className='grid gap-3 sm:grid-cols-2'
          >
            <div className='bg-muted/40 rounded-lg border p-3 text-left'>
              <div className='flex items-center gap-2'>
                <CreditCard className='text-muted-foreground size-4' />
                <p className='text-sm font-medium'>{t('USD credit amount')}</p>
              </div>
              <p className='mt-2 text-lg font-semibold'>
                {validation === null ? `${formatUsdAmount(amount)} USD` : '—'}
              </p>
              <p className='text-muted-foreground mt-1 text-xs'>
                {t('The authenticated team is credited after payment.')}
              </p>
            </div>

            <div className='space-y-2'>
              <div
                data-slot='team-standard-payment-quote'
                aria-live='polite'
                className='bg-muted/40 flex min-h-14 items-center justify-between rounded-lg border p-3'
              >
                <span className='text-muted-foreground text-sm'>
                  {t('You Pay')}
                </span>
                <div className='flex items-center gap-2'>
                  {standardAmountQuery.isFetching && (
                    <Loader2 className='text-muted-foreground size-4 animate-spin' />
                  )}
                  <span className='font-semibold'>
                    {hasValidStandardQuote
                      ? `${formatUsdAmount(standardQuotedAmount)} USD`
                      : '—'}
                  </span>
                </div>
              </div>

              {props.topupInfo?.enable_stripe_wechat_pay && (
                <div
                  data-slot='team-wechat-payment-quote'
                  aria-live='polite'
                  className='flex min-h-14 items-center justify-between rounded-lg border border-[#07C160]/30 bg-[#07C160]/5 p-3'
                >
                  <div className='flex items-center gap-2 text-[#067a3b] dark:text-[#41d17c]'>
                    {getPaymentIcon(PAYMENT_TYPES.WECHAT, 'size-4')}
                    <span className='text-sm font-medium'>
                      {t('WeChat Pay')}
                    </span>
                  </div>
                  <div className='flex items-center gap-2'>
                    {weChatAmountQuery.isFetching && (
                      <Loader2 className='text-muted-foreground size-4 animate-spin' />
                    )}
                    <span className='font-semibold'>
                      {hasValidWeChatQuote
                        ? `${formatCnyAmount(weChatQuotedAmount)} CNY`
                        : '—'}
                    </span>
                  </div>
                </div>
              )}
            </div>
          </div>

          {props.topupInfo?.enable_stripe_wechat_pay && (
            <p className='text-muted-foreground text-xs leading-5'>
              {t(
                'WeChat checkout displays the final amount in CNY; your wallet is credited with the selected USD amount.'
              )}
            </p>
          )}

          {standardAmountQuery.isError && validation === null && (
            <p className='text-destructive text-sm' role='alert'>
              {t('Unable to verify this amount. Please try again.')}
            </p>
          )}
          {weChatAmountQuery.isError && validation === null && (
            <p className='text-destructive text-sm' role='alert'>
              {t('Unable to verify the WeChat Pay amount. Please try again.')}
            </p>
          )}
        </div>

        <DialogFooter
          className={cn(
            'grid gap-2 sm:space-x-0',
            props.topupInfo?.enable_stripe_wechat_pay
              ? 'sm:grid-cols-[auto_1fr_1fr]'
              : 'sm:grid-cols-[auto_1fr]'
          )}
        >
          <Button
            type='button'
            variant='outline'
            disabled={checkoutMutation.isPending}
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            disabled={!canSubmitStandard}
            onClick={() => submit(STRIPE_CHECKOUT_METHODS.STANDARD)}
          >
            {checkoutMutation.isPending &&
              checkoutMutation.variables?.checkoutMethod ===
                STRIPE_CHECKOUT_METHODS.STANDARD && (
                <Loader2 className='size-4 animate-spin' />
              )}
            {t('Pay')}
          </Button>
          {props.topupInfo?.enable_stripe_wechat_pay && (
            <Button
              type='button'
              variant='outline'
              disabled={!canSubmitWeChat}
              onClick={() => submit(STRIPE_CHECKOUT_METHODS.WECHAT_PAY)}
              className='border-[#07C160]/50 text-[#067a3b] hover:border-[#07C160] hover:bg-[#07C160]/5 hover:text-[#067a3b] dark:text-[#41d17c] dark:hover:text-[#5fe494]'
            >
              {checkoutMutation.isPending &&
              checkoutMutation.variables?.checkoutMethod ===
                STRIPE_CHECKOUT_METHODS.WECHAT_PAY ? (
                <Loader2 className='size-4 animate-spin' />
              ) : (
                getPaymentIcon(PAYMENT_TYPES.WECHAT, 'size-4')
              )}
              {t('WeChat Pay')}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
