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
import { Gift, ExternalLink, Loader2, Receipt, WalletCards } from 'lucide-react'
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import { IconBadge } from '@/components/ui/icon-badge'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Skeleton } from '@/components/ui/skeleton'
import { TitledCard } from '@/components/ui/titled-card'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  formatCurrency,
  formatUsdAmount,
  getDiscountLabel,
  getPaymentIcon,
  getMinTopupAmount,
  getStripeTopupBounds,
  isIntegerTopupAmount,
  calculatePresetPricing,
  validateStripeTopupAmount,
} from '../lib'
import type {
  PaymentMethod,
  PresetAmount,
  TopupInfo,
  CreemProduct,
  WaffoPayMethod,
} from '../types'
import { CreemProductsSection } from './creem-products-section'

interface RechargeFormCardProps {
  topupInfo: TopupInfo | null
  presetAmounts: PresetAmount[]
  selectedPreset: number | null
  onSelectPreset: (preset: PresetAmount) => void
  topupAmount: number
  onTopupAmountChange: (amount: number) => void
  paymentAmount: number
  calculating: boolean
  onPaymentMethodSelect: (method: PaymentMethod) => void
  paymentLoading: string | null
  redemptionCode: string
  onRedemptionCodeChange: (code: string) => void
  onRedeem: () => void
  redeeming: boolean
  topupLink?: string
  loading?: boolean
  priceRatio?: number
  usdExchangeRate?: number
  onOpenBilling?: () => void
  creemProducts?: CreemProduct[]
  enableCreemTopup?: boolean
  onCreemProductSelect?: (product: CreemProduct) => void
  enableWaffoTopup?: boolean
  waffoPayMethods?: WaffoPayMethod[]
  waffoMinTopup?: number
  onWaffoMethodSelect?: (method: WaffoPayMethod, index: number) => void
  enableWaffoPancakeTopup?: boolean
}

export function RechargeFormCard({
  topupInfo,
  presetAmounts,
  selectedPreset,
  onSelectPreset,
  topupAmount,
  onTopupAmountChange,
  paymentAmount,
  calculating,
  onPaymentMethodSelect,
  paymentLoading,
  redemptionCode,
  onRedemptionCodeChange,
  onRedeem,
  redeeming,
  topupLink,
  loading,
  priceRatio = 1,
  usdExchangeRate = 1,
  onOpenBilling,
  creemProducts,
  enableCreemTopup,
  onCreemProductSelect,
  enableWaffoTopup,
  waffoPayMethods,
  waffoMinTopup,
  onWaffoMethodSelect,
  enableWaffoPancakeTopup,
}: RechargeFormCardProps) {
  const { t } = useTranslation()
  const [localAmount, setLocalAmount] = useState(topupAmount.toString())

  useEffect(() => {
    setLocalAmount(topupAmount.toString())
  }, [topupAmount])

  const handleAmountChange = (value: string) => {
    setLocalAmount(value)
    const numericValue = Number(value)
    if (value === '' || !Number.isFinite(numericValue)) {
      onTopupAmountChange(0)
      return
    }
    if (numericValue >= 0) {
      onTopupAmountChange(numericValue)
    }
  }

  const hasConfigurableTopup =
    topupInfo?.enable_online_topup ||
    topupInfo?.enable_stripe_topup ||
    enableWaffoTopup ||
    enableWaffoPancakeTopup
  const hasAnyTopup = hasConfigurableTopup || enableCreemTopup
  const hasStandardPaymentMethods =
    Array.isArray(topupInfo?.pay_methods) && topupInfo.pay_methods.length > 0
  const hasWaffoPaymentMethods =
    Array.isArray(waffoPayMethods) && waffoPayMethods.length > 0
  const minTopup = getMinTopupAmount(topupInfo)
  const stripeBounds = getStripeTopupBounds(topupInfo)
  const stripeValidation = validateStripeTopupAmount(topupAmount, stripeBounds)
  const hasUsdStripe = topupInfo?.enable_stripe_topup === true
  const parsedLocalAmount = Number(localAmount)
  const hasFractionalAmount =
    localAmount !== '' &&
    Number.isFinite(parsedLocalAmount) &&
    !isIntegerTopupAmount(parsedLocalAmount)
  const redemptionEnabled = topupInfo?.enable_redemption !== false
  let singleStripePaymentMethod: PaymentMethod | null = null
  if (
    hasUsdStripe &&
    topupInfo?.pay_methods?.length === 1 &&
    topupInfo.pay_methods[0]?.type === 'stripe'
  ) {
    singleStripePaymentMethod = topupInfo.pay_methods[0]
  }

  let stripeValidationMessage = ''
  if (stripeValidation === 'integer') {
    stripeValidationMessage = t('Enter a whole USD amount.')
  } else if (stripeValidation === 'minimum') {
    stripeValidationMessage = t('Minimum top-up is ${{amount}} USD.', {
      amount: stripeBounds.min,
    })
  } else if (stripeValidation === 'maximum') {
    stripeValidationMessage = t('Maximum top-up is ${{amount}} USD.', {
      amount: stripeBounds.max,
    })
  } else if (stripeValidation === 'required' && localAmount !== '') {
    stripeValidationMessage = t('Enter a valid USD amount.')
  }
  let amountValidationMessage = ''
  if (hasFractionalAmount) {
    amountValidationMessage = t('Amount must be a whole number')
  } else if (hasUsdStripe) {
    amountValidationMessage = stripeValidationMessage
  }
  let displayedPaymentAmount = formatCurrency(paymentAmount)
  if (hasFractionalAmount) {
    displayedPaymentAmount = '—'
  } else if (hasUsdStripe) {
    displayedPaymentAmount =
      stripeValidation === null ? `${formatUsdAmount(paymentAmount)} USD` : '—'
  }
  let singlePaymentDisabledReason = ''
  if (
    singleStripePaymentMethod &&
    (hasFractionalAmount || stripeValidation !== null)
  ) {
    singlePaymentDisabledReason =
      amountValidationMessage || t('Enter a valid USD amount.')
  } else if (
    singleStripePaymentMethod?.min_topup &&
    singleStripePaymentMethod.min_topup > topupAmount
  ) {
    singlePaymentDisabledReason = t('Minimum topup amount: {{amount}}', {
      amount: singleStripePaymentMethod.min_topup,
    })
  }

  if (loading) {
    return (
      <Card data-card-hover='false' className='gap-0 overflow-hidden py-0'>
        <CardHeader className='border-b p-3 !pb-3 sm:p-5 sm:!pb-5'>
          <Skeleton className='h-6 w-32' />
          <Skeleton className='mt-2 h-4 w-48' />
        </CardHeader>
        <CardContent className='space-y-4 p-3 sm:space-y-6 sm:p-5'>
          <div className='space-y-4 sm:space-y-6'>
            {/* Preset Amounts Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-16' />
              <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
                {Array.from({ length: 6 }, (_, index) => `preset-${index}`).map(
                  (key) => (
                    <Skeleton key={key} className='h-[72px] rounded-lg' />
                  )
                )}
              </div>
            </div>

            {/* Custom Amount Input Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-28' />
              <Skeleton className='h-[42px] w-full' />
            </div>

            {/* Payment Methods Skeleton */}
            <div className='space-y-3'>
              <Skeleton className='h-3 w-32' />
              <div className='flex flex-wrap gap-3'>
                {['primary', 'secondary', 'tertiary'].map((key) => (
                  <Skeleton key={key} className='h-10 w-24 rounded-lg' />
                ))}
              </div>
            </div>
          </div>

          {/* Redemption Code Section Skeleton */}
          <div className='space-y-3 border-t pt-8'>
            <Skeleton className='h-3 w-24' />
            <div className='flex gap-2'>
              <Skeleton className='h-10 flex-1' />
              <Skeleton className='h-10 w-20' />
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <TitledCard
      title={t('Add Funds')}
      description={
        singleStripePaymentMethod
          ? undefined
          : t('Choose an amount and payment method')
      }
      icon={<WalletCards className='h-4 w-4' />}
      iconTone='success'
      disableHoverEffect
      action={
        onOpenBilling ? (
          <Button
            variant='outline'
            size='sm'
            onClick={onOpenBilling}
            className='w-full gap-2 sm:w-auto'
          >
            <Receipt className='h-4 w-4' />
            {t('Order History')}
          </Button>
        ) : null
      }
      contentClassName='space-y-4 sm:space-y-6'
    >
      {/* Online Topup Section */}
      {hasAnyTopup ? (
        <div className='space-y-4 sm:space-y-6'>
          {hasConfigurableTopup && (
            <>
              {presetAmounts.length > 0 && (
                <div className='space-y-2.5 sm:space-y-3'>
                  <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                    {t('Amount')}
                  </Label>
                  <div className='grid grid-cols-2 gap-1.5 sm:grid-cols-3 sm:gap-3'>
                    {presetAmounts.map((preset) => {
                      const discount =
                        preset.discount ||
                        topupInfo?.discount?.[preset.value] ||
                        1.0
                      const {
                        displayValue,
                        actualPrice,
                        savedAmount,
                        hasDiscount,
                      } = calculatePresetPricing(
                        preset.value,
                        priceRatio,
                        discount,
                        usdExchangeRate
                      )
                      const displayedActualPrice = hasUsdStripe
                        ? preset.value
                        : actualPrice
                      const showDiscount = !hasUsdStripe && hasDiscount
                      return (
                        <Button
                          key={preset.value}
                          variant='outline'
                          className={cn(
                            'flex min-h-16 flex-col items-start rounded-lg px-3 py-2.5 text-left whitespace-normal sm:min-h-[72px] sm:p-4',
                            selectedPreset === preset.value
                              ? 'border-foreground bg-foreground/5 dark:border-foreground dark:bg-foreground/10'
                              : 'border-muted'
                          )}
                          onClick={() => onSelectPreset(preset)}
                        >
                          <div className='flex w-full items-center justify-between'>
                            <div className='text-base font-semibold sm:text-lg'>
                              {hasUsdStripe
                                ? formatUsdAmount(preset.value)
                                : formatNumber(displayValue)}
                            </div>
                            {showDiscount && (
                              <div className='text-xs font-medium text-green-600'>
                                {getDiscountLabel(discount)}
                              </div>
                            )}
                          </div>
                          <div className='text-muted-foreground mt-1.5 w-full text-xs sm:mt-2'>
                            {hasUsdStripe
                              ? t('Pay {{amount}} USD', {
                                  amount: formatUsdAmount(displayedActualPrice),
                                })
                              : `Pay ${formatCurrency(displayedActualPrice)}`}
                            {showDiscount && savedAmount > 0 && (
                              <span className='text-green-600'>
                                {' '}
                                •{' '}
                                {hasUsdStripe
                                  ? t('Save {{amount}} USD', {
                                      amount: formatUsdAmount(savedAmount),
                                    })
                                  : `Save ${formatCurrency(savedAmount)}`}
                              </span>
                            )}
                          </div>
                        </Button>
                      )
                    })}
                  </div>
                </div>
              )}

              <div
                data-slot='wallet-amount-entry'
                className='grid max-w-2xl grid-cols-1 gap-3 sm:grid-cols-[minmax(0,18rem)_minmax(0,16rem)] sm:items-start'
              >
                <div
                  data-slot='wallet-custom-amount-field'
                  className='space-y-2.5'
                >
                  <Label
                    htmlFor='topup-amount'
                    className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
                  >
                    {hasUsdStripe
                      ? t('Custom Amount (USD)')
                      : t('Custom Amount')}
                  </Label>
                  <div>
                    <Input
                      id='topup-amount'
                      type='number'
                      inputMode='numeric'
                      value={localAmount}
                      onChange={(e) => handleAmountChange(e.target.value)}
                      min={hasUsdStripe ? stripeBounds.min : minTopup}
                      max={hasUsdStripe ? stripeBounds.max : undefined}
                      step={1}
                      aria-invalid={
                        amountValidationMessage ? 'true' : undefined
                      }
                      aria-describedby={
                        hasUsdStripe || hasFractionalAmount
                          ? 'topup-amount-guidance'
                          : undefined
                      }
                      placeholder={
                        hasUsdStripe
                          ? t('${{min}}–${{max}} USD', {
                              min: stripeBounds.min,
                              max: stripeBounds.max,
                            })
                          : `Minimum ${minTopup}`
                      }
                      className='h-10 text-base sm:text-lg'
                    />
                    {(hasUsdStripe || hasFractionalAmount) && (
                      <p
                        id='topup-amount-guidance'
                        aria-live='polite'
                        className={cn(
                          'mt-1.5 text-xs',
                          amountValidationMessage
                            ? 'text-destructive'
                            : 'text-muted-foreground'
                        )}
                      >
                        {amountValidationMessage ||
                          t('Whole USD amounts from ${{min}} to ${{max}}.', {
                            min: stripeBounds.min,
                            max: stripeBounds.max,
                          })}
                      </p>
                    )}
                  </div>
                </div>
                <div data-slot='wallet-payment-total' className='space-y-2.5'>
                  <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                    {t('Amount to pay:')}
                  </p>
                  <div
                    data-slot='wallet-payment-total-display'
                    className='bg-muted/30 flex h-10 items-center justify-end rounded-lg border px-3'
                  >
                    {calculating ? (
                      <Skeleton className='h-5 w-16' />
                    ) : (
                      <span className='text-base font-semibold'>
                        {displayedPaymentAmount}
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {singleStripePaymentMethod ? (
                <Button
                  data-slot='wallet-primary-payment-button'
                  onClick={() =>
                    onPaymentMethodSelect(singleStripePaymentMethod)
                  }
                  disabled={!!singlePaymentDisabledReason || !!paymentLoading}
                  title={singlePaymentDisabledReason || undefined}
                  aria-label={
                    singlePaymentDisabledReason
                      ? `${t('Pay')}. ${singlePaymentDisabledReason}`
                      : t('Pay')
                  }
                  className='h-10 w-full sm:w-auto sm:min-w-56'
                >
                  {paymentLoading === singleStripePaymentMethod.type && (
                    <Loader2 className='h-4 w-4 animate-spin' />
                  )}
                  {t('Pay')}
                </Button>
              ) : (
                <div className='space-y-2.5 sm:space-y-3'>
                  <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                    {t('Payment Method')}
                  </Label>
                  {hasStandardPaymentMethods ? (
                    <div className='grid grid-cols-2 gap-1.5 sm:gap-3 lg:grid-cols-3'>
                      {topupInfo?.pay_methods?.map((method) => {
                        const minTopup = method.min_topup || 0
                        const belowMethodMinimum = minTopup > topupAmount
                        const invalidStripeAmount =
                          !hasFractionalAmount &&
                          method.type === 'stripe' &&
                          stripeValidation !== null
                        const disabled =
                          hasFractionalAmount ||
                          belowMethodMinimum ||
                          invalidStripeAmount
                        let disabledReason: string | undefined
                        if (hasFractionalAmount) {
                          disabledReason = t('Amount must be a whole number')
                        } else if (invalidStripeAmount) {
                          disabledReason =
                            stripeValidationMessage ||
                            t('Enter a valid USD amount.')
                        } else if (belowMethodMinimum) {
                          disabledReason = t(
                            'Minimum topup amount: {{amount}}',
                            {
                              amount: minTopup,
                            }
                          )
                        }
                        let disabledLabel: string | undefined
                        if (hasFractionalAmount) {
                          disabledLabel = t('Amount must be a whole number')
                        } else if (disabled && method.type === 'stripe') {
                          disabledLabel = t('${{min}}–${{max}} USD', {
                            min: stripeBounds.min,
                            max: stripeBounds.max,
                          })
                        } else if (disabled) {
                          disabledLabel = `${t('Minimum:')} ${minTopup}`
                        }

                        const button = (
                          <Button
                            key={method.type}
                            variant='outline'
                            onClick={() => onPaymentMethodSelect(method)}
                            disabled={disabled || !!paymentLoading}
                            title={disabledReason}
                            aria-label={
                              disabledReason
                                ? `${method.name}. ${disabledReason}`
                                : method.name
                            }
                            className='min-h-14 min-w-0 justify-start gap-2 rounded-lg px-3 py-2 text-left'
                          >
                            {paymentLoading === method.type ? (
                              <Loader2 className='h-4 w-4 animate-spin' />
                            ) : (
                              getPaymentIcon(
                                method.type,
                                'h-4 w-4',
                                method.icon,
                                method.name
                              )
                            )}
                            <span className='flex min-w-0 flex-col items-start gap-0.5'>
                              <span className='max-w-full truncate'>
                                {method.name}
                              </span>
                              {disabledLabel && (
                                <span className='text-muted-foreground max-w-full truncate text-[11px] leading-4 font-normal'>
                                  {disabledLabel}
                                </span>
                              )}
                            </span>
                          </Button>
                        )

                        return disabled ? (
                          <TooltipProvider key={method.type}>
                            <Tooltip>
                              <TooltipTrigger render={button} />
                              <TooltipContent>{disabledReason}</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        ) : (
                          button
                        )
                      })}
                    </div>
                  ) : null}
                  {!hasStandardPaymentMethods && !hasWaffoPaymentMethods && (
                    <Alert>
                      <AlertDescription>
                        {t(
                          'No payment methods available. Please contact administrator.'
                        )}
                      </AlertDescription>
                    </Alert>
                  )}
                </div>
              )}

              {enableWaffoTopup &&
                hasWaffoPaymentMethods &&
                onWaffoMethodSelect && (
                  <div className='space-y-2.5 sm:space-y-3'>
                    <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                      {t('Waffo Payment')}
                    </Label>
                    <div className='grid grid-cols-2 gap-1.5 sm:gap-3 lg:grid-cols-3'>
                      {waffoPayMethods?.map((method, index) => {
                        const loadingKey = `waffo-${index}`
                        const methodKey = `${method.payMethodType ?? 'unknown'}-${method.payMethodName ?? method.name}`
                        const waffoMin = waffoMinTopup || 0
                        const belowMin = waffoMin > topupAmount
                        let disabledReason: string | undefined
                        let disabledLabel: string | undefined
                        if (hasFractionalAmount) {
                          disabledReason = t('Amount must be a whole number')
                          disabledLabel = disabledReason
                        } else if (belowMin) {
                          disabledReason = t(
                            'Minimum topup amount: {{amount}}',
                            {
                              amount: waffoMin,
                            }
                          )
                          disabledLabel = `${t('Minimum:')} ${waffoMin}`
                        }
                        const disabled = hasFractionalAmount || belowMin

                        let methodIcon = getPaymentIcon('waffo')
                        if (paymentLoading === loadingKey) {
                          methodIcon = (
                            <Loader2 className='h-4 w-4 animate-spin' />
                          )
                        } else if (method.icon) {
                          methodIcon = (
                            <img
                              src={method.icon}
                              alt={method.name}
                              className='h-4 w-4 object-contain'
                            />
                          )
                        }

                        const button = (
                          <Button
                            key={methodKey}
                            variant='outline'
                            onClick={() => onWaffoMethodSelect(method, index)}
                            disabled={disabled || !!paymentLoading}
                            title={disabledReason}
                            aria-label={
                              disabledReason
                                ? `${method.name}. ${disabledReason}`
                                : method.name
                            }
                            className='min-h-14 min-w-0 justify-start gap-2 rounded-lg px-3 py-2 text-left'
                          >
                            {methodIcon}
                            <span className='flex min-w-0 flex-col items-start gap-0.5'>
                              <span className='max-w-full truncate'>
                                {method.name}
                              </span>
                              {disabledLabel && (
                                <span className='text-muted-foreground max-w-full truncate text-[11px] leading-4 font-normal'>
                                  {disabledLabel}
                                </span>
                              )}
                            </span>
                          </Button>
                        )

                        return disabled ? (
                          <TooltipProvider key={methodKey}>
                            <Tooltip>
                              <TooltipTrigger render={button} />
                              <TooltipContent>{disabledReason}</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        ) : (
                          button
                        )
                      })}
                    </div>
                  </div>
                )}
            </>
          )}
        </div>
      ) : (
        <Alert>
          <AlertDescription>
            {t(
              'Online topup is not enabled. Please use redemption code or contact administrator.'
            )}
          </AlertDescription>
        </Alert>
      )}

      {/* Creem Products Section */}
      {enableCreemTopup &&
        Array.isArray(creemProducts) &&
        creemProducts.length > 0 &&
        onCreemProductSelect && (
          <div className='space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
            <Label className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
              {t('Creem Payment')}
            </Label>
            <CreemProductsSection
              products={creemProducts}
              onProductSelect={onCreemProductSelect}
            />
          </div>
        )}

      {/* Redemption Code Section */}
      {redemptionEnabled ? (
        <div className='space-y-2.5 border-t pt-4 sm:space-y-3 sm:pt-6'>
          <div className='flex items-center gap-2'>
            <IconBadge tone='warning' size='xs'>
              <Gift />
            </IconBadge>
            <Label
              htmlFor='redemption-code'
              className='text-muted-foreground text-xs font-medium tracking-wider uppercase'
            >
              {t('Have a Code?')}
            </Label>
          </div>
          <div className='grid grid-cols-[minmax(0,1fr)_auto] gap-2'>
            <Input
              id='redemption-code'
              value={redemptionCode}
              onChange={(e) => onRedemptionCodeChange(e.target.value)}
              placeholder={t('Enter your redemption code')}
              className='h-9 min-w-0'
            />
            <Button
              onClick={onRedeem}
              disabled={redeeming}
              variant='outline'
              className='h-9 px-4'
            >
              {redeeming && <Loader2 className='mr-2 h-4 w-4 animate-spin' />}
              {t('Redeem')}
            </Button>
          </div>
          {topupLink && (
            <p className='text-muted-foreground text-xs'>
              {t('Need a redemption code?')}{' '}
              <a
                href={topupLink}
                target='_blank'
                rel='noopener noreferrer'
                className='inline-flex items-center gap-1 underline-offset-4 hover:underline'
              >
                {t('Get one here')}
                <ExternalLink className='h-3 w-3' />
              </a>
            </p>
          )}
        </div>
      ) : (
        <Alert className='border-t'>
          <AlertDescription>
            {t(
              'Redemption codes are disabled until the administrator confirms compliance terms.'
            )}
          </AlertDescription>
        </Alert>
      )}
    </TitledCard>
  )
}
