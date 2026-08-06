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
  formatUsdAmount,
  getStripeTopupBounds,
  getStripeTopupPresets,
  validateStripeTopupAmount,
} from '@/features/wallet/lib'
import type { TopupInfo } from '@/features/wallet/types'
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

  const amountQuery = useQuery({
    queryKey: ['team-stripe-amount', amount],
    queryFn: () => calculateCurrentTeamStripeAmount(amount),
    enabled: props.open && validation === null,
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

  const quotedAmount = Number(amountQuery.data)
  const hasValidQuote =
    amountQuery.isSuccess && Number.isFinite(quotedAmount) && quotedAmount > 0
  const canSubmit =
    validation === null &&
    hasValidQuote &&
    !checkoutMutation.isPending &&
    props.topupInfo?.enable_stripe_topup === true

  const submit = () => {
    if (!canSubmit) return
    checkoutMutation.mutate(amount)
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Add Team Funds')}</DialogTitle>
          <DialogDescription>
            {t(
              'Pay with Stripe and add USD credits directly to the shared team balance.'
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

          <div className='bg-muted/40 flex items-center justify-between rounded-lg border p-3'>
            <div>
              <p className='text-sm font-medium'>{t('Stripe Checkout')}</p>
              <p className='text-muted-foreground text-xs'>
                {t('The authenticated team is credited after payment.')}
              </p>
            </div>
            <div className='flex items-center gap-2 text-right'>
              {amountQuery.isFetching ? (
                <Loader2 className='text-muted-foreground size-4 animate-spin' />
              ) : (
                <CreditCard className='text-muted-foreground size-4' />
              )}
              <span className='font-semibold'>
                {hasValidQuote ? `${formatUsdAmount(quotedAmount)} USD` : '—'}
              </span>
            </div>
          </div>

          {amountQuery.isError && validation === null && (
            <p className='text-destructive text-sm' role='alert'>
              {t('Unable to verify this amount. Please try again.')}
            </p>
          )}
        </div>

        <DialogFooter>
          <Button
            type='button'
            variant='outline'
            disabled={checkoutMutation.isPending}
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button type='button' disabled={!canSubmit} onClick={submit}>
            {checkoutMutation.isPending && (
              <Loader2 className='size-4 animate-spin' />
            )}
            {t('Continue to Stripe')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
