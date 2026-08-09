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
import { useNavigate } from '@tanstack/react-router'
import { useState, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { SectionPageLayout } from '@/components/layout'
import { useStatus } from '@/hooks/use-status'
import { useSystemConfig } from '@/hooks/use-system-config'
import { getSelf } from '@/lib/api'
import { useAuthStore } from '@/stores/auth-store'

import { AffiliateRewardsCard } from './components/affiliate-rewards-card'
import { BillingHistoryDialog } from './components/dialogs/billing-history-dialog'
import { CreemConfirmDialog } from './components/dialogs/creem-confirm-dialog'
import { PaymentConfirmDialog } from './components/dialogs/payment-confirm-dialog'
import { TransferDialog } from './components/dialogs/transfer-dialog'
import { RechargeFormCard } from './components/recharge-form-card'
import { SubscriptionPlansCard } from './components/subscription-plans-card'
import { TeamWalletNotice } from './components/team-wallet-notice'
import { WalletStatsCard } from './components/wallet-stats-card'
import {
  DEFAULT_DISCOUNT_RATE,
  PAYMENT_RETURN_REFRESH_DELAYS_MS,
  PAYMENT_TYPES,
} from './constants'
import {
  useTopupInfo,
  usePayment,
  useAffiliate,
  useRedemption,
  useCreemPayment,
  useWaffoPayment,
  useWaffoPancakePayment,
} from './hooks'
import {
  getDefaultPaymentType,
  getMinTopupAmount,
  getStripeTopupBounds,
  isIntegerTopupAmount,
  isStripePayment,
  dispatchSelectedPayment,
  canTopUpPersonalWallet,
  validateStripeTopupAmount,
  getPaymentLoadingKey,
} from './lib'
import type {
  UserWalletData,
  PaymentMethod,
  PresetAmount,
  CreemProduct,
  WaffoPayMethod,
} from './types'

interface WalletProps {
  initialShowHistory?: boolean
  initialStripeStatus?: 'success' | 'cancel'
}

export function Wallet(props: WalletProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const personalTopupAllowed = useAuthStore((state) =>
    canTopUpPersonalWallet(state.auth.user)
  )
  const [user, setUser] = useState<UserWalletData | null>(null)
  const [userLoading, setUserLoading] = useState(true)
  const [topupAmount, setTopupAmount] = useState(0)
  const [selectedPreset, setSelectedPreset] = useState<number | null>(null)
  const [selectedPaymentMethod, setSelectedPaymentMethod] =
    useState<PaymentMethod>()
  const [selectedWaffoMethodIndex, setSelectedWaffoMethodIndex] = useState<
    number | null
  >(null)
  const [paymentLoading, setPaymentLoading] = useState<string | null>(null)
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false)
  const [transferDialogOpen, setTransferDialogOpen] = useState(false)
  const [billingDialogOpen, setBillingDialogOpen] = useState(false)
  const [redemptionCode, setRedemptionCode] = useState('')
  const [creemDialogOpen, setCreemDialogOpen] = useState(false)
  const [selectedCreemProduct, setSelectedCreemProduct] =
    useState<CreemProduct | null>(null)
  const [showSubscriptionPanel, setShowSubscriptionPanel] = useState(true)

  const { status } = useStatus()
  const { currency } = useSystemConfig()
  const { topupInfo, presetAmounts, loading: topupLoading } = useTopupInfo()

  // Calculate effective exchange rate - when display type is USD, use rate of 1
  const effectiveUsdExchangeRate = useMemo(() => {
    return currency?.quotaDisplayType === 'USD'
      ? 1
      : currency?.usdExchangeRate || 1
  }, [currency?.quotaDisplayType, currency?.usdExchangeRate])
  const {
    amount: paymentAmount,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
    setAmount: setPaymentAmount,
  } = usePayment()
  const {
    affiliateLink,
    loading: affiliateLoading,
    transferQuota,
    transferring,
  } = useAffiliate()
  const { redeeming, redeemCode } = useRedemption()
  const { processing: creemProcessing, processCreemPayment } = useCreemPayment()
  const { processing: waffoProcessing, processWaffoPayment } = useWaffoPayment()
  const { processing: pancakeProcessing, processWaffoPancakePayment } =
    useWaffoPancakePayment()

  // Fetch and refresh user data
  const fetchUser = useCallback(async (showLoading = true) => {
    try {
      if (showLoading) setUserLoading(true)
      const response = await getSelf()
      if (response.success && response.data) {
        setUser(response.data as UserWalletData)
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error('Failed to fetch user data:', error)
    } finally {
      if (showLoading) setUserLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  useEffect(() => {
    if (props.initialShowHistory) {
      setBillingDialogOpen(true)
      window.history.replaceState({}, '', window.location.pathname)
    }
  }, [props.initialShowHistory])

  useEffect(() => {
    if (!props.initialStripeStatus) return

    const refreshTimers: number[] = []

    if (props.initialStripeStatus === 'success') {
      toast.success(
        t(
          'Returned from Stripe. We are confirming your payment and refreshing the balance.'
        )
      )
      PAYMENT_RETURN_REFRESH_DELAYS_MS.forEach((delay) => {
        refreshTimers.push(
          window.setTimeout(() => {
            void fetchUser(false)
          }, delay)
        )
      })
    } else {
      toast.info(t('Stripe top-up was cancelled. No charge was made.'))
    }
    window.history.replaceState({}, '', window.location.pathname)

    return () => {
      refreshTimers.forEach((timer) => window.clearTimeout(timer))
    }
  }, [fetchUser, props.initialStripeStatus, t])

  // Initialize topup amount when topup info is loaded
  useEffect(() => {
    if (topupInfo && topupAmount === 0) {
      const defaultPaymentType = getDefaultPaymentType(topupInfo)
      const minTopup = isStripePayment(defaultPaymentType)
        ? getStripeTopupBounds(topupInfo).min
        : getMinTopupAmount(topupInfo)
      setTopupAmount(minTopup)

      // Calculate initial payment amount with default payment type
      calculatePaymentAmount(minTopup, defaultPaymentType)
    }
  }, [topupInfo, topupAmount, calculatePaymentAmount])

  // Get current payment type (selected or default)
  const getCurrentPaymentType = useCallback(() => {
    return selectedPaymentMethod?.type || getDefaultPaymentType(topupInfo)
  }, [selectedPaymentMethod, topupInfo])

  // Handle preset selection
  const handleSelectPreset = (preset: PresetAmount) => {
    setTopupAmount(preset.value)
    setSelectedPreset(preset.value)
    calculatePaymentAmount(preset.value, getCurrentPaymentType())
  }

  // Handle topup amount change
  const handleTopupAmountChange = (amount: number) => {
    setTopupAmount(amount)
    setSelectedPreset(null)
    if (!isIntegerTopupAmount(amount) || amount <= 0) {
      setPaymentAmount(0)
      return
    }
    const paymentType = getCurrentPaymentType()
    const stripeValidation = validateStripeTopupAmount(
      amount,
      getStripeTopupBounds(topupInfo)
    )
    if (isStripePayment(paymentType) && stripeValidation !== null) {
      setPaymentAmount(0)
      return
    }
    calculatePaymentAmount(amount, paymentType)
  }

  // Handle payment method selection
  const handlePaymentMethodSelect = async (method: PaymentMethod) => {
    if (!isIntegerTopupAmount(topupAmount)) {
      toast.error(t('Amount must be a whole number'))
      return
    }

    setSelectedPaymentMethod(method)
    setSelectedWaffoMethodIndex(null)
    setPaymentLoading(getPaymentLoadingKey(method))

    try {
      // Validate minimum topup
      const minTopup = isStripePayment(method.type)
        ? getStripeTopupBounds(topupInfo).min
        : method.min_topup || getMinTopupAmount(topupInfo)
      if (topupAmount < minTopup) {
        return
      }
      if (
        isStripePayment(method.type) &&
        validateStripeTopupAmount(
          topupAmount,
          getStripeTopupBounds(topupInfo)
        ) !== null
      ) {
        toast.error(t('Enter a whole USD amount within the allowed range.'))
        return
      }

      // Calculate payment amount and show confirmation dialog
      await calculatePaymentAmount(topupAmount, method.type)
      setConfirmDialogOpen(true)
    } finally {
      setPaymentLoading(null)
    }
  }

  // Handle payment confirmation
  const handlePaymentConfirm = async () => {
    if (!selectedPaymentMethod) return
    if (!isIntegerTopupAmount(topupAmount)) {
      toast.error(t('Amount must be a whole number'))
      return
    }
    if (
      isStripePayment(selectedPaymentMethod.type) &&
      validateStripeTopupAmount(
        topupAmount,
        getStripeTopupBounds(topupInfo)
      ) !== null
    ) {
      toast.error(t('Enter a whole USD amount within the allowed range.'))
      return
    }

    const success = await dispatchSelectedPayment(
      selectedPaymentMethod,
      topupAmount,
      selectedWaffoMethodIndex,
      {
        regular: processPayment,
        waffo: processWaffoPayment,
        waffoPancake: processWaffoPancakePayment,
      }
    )

    if (success) {
      setConfirmDialogOpen(false)
      await fetchUser()
    }
  }

  // Handle redemption
  const handleRedeem = async () => {
    if (!redemptionCode) return

    const success = await redeemCode(redemptionCode)
    if (success) {
      setRedemptionCode('')
      await fetchUser()
    }
  }

  // Handle transfer
  const handleTransfer = async (amount: number) => {
    const success = await transferQuota(amount)
    if (success) {
      await fetchUser()
    }
    return success
  }

  // Handle Creem product selection
  const handleCreemProductSelect = (product: CreemProduct) => {
    setSelectedCreemProduct(product)
    setCreemDialogOpen(true)
  }

  // Handle Creem payment confirmation
  const handleCreemConfirm = async () => {
    if (!selectedCreemProduct) return

    const success = await processCreemPayment(selectedCreemProduct.productId)
    if (success) {
      setCreemDialogOpen(false)
      setSelectedCreemProduct(null)
      await fetchUser()
    }
  }

  const handleWaffoMethodSelect = async (
    method: WaffoPayMethod,
    index: number
  ) => {
    if (!isIntegerTopupAmount(topupAmount)) {
      toast.error(t('Amount must be a whole number'))
      return
    }

    const loadingKey = `waffo-${index}`
    setSelectedPaymentMethod({
      name: method.name,
      type: PAYMENT_TYPES.WAFFO,
      icon: method.icon,
    })
    setSelectedWaffoMethodIndex(index)
    setPaymentLoading(loadingKey)

    try {
      await calculatePaymentAmount(topupAmount, PAYMENT_TYPES.WAFFO)
      setConfirmDialogOpen(true)
    } finally {
      setPaymentLoading(null)
    }
  }

  // Get discount rate for current topup amount
  const getDiscountRate = useCallback(() => {
    if (selectedPaymentMethod && isStripePayment(selectedPaymentMethod.type)) {
      return DEFAULT_DISCOUNT_RATE
    }
    return topupInfo?.discount?.[topupAmount] || DEFAULT_DISCOUNT_RATE
  }, [selectedPaymentMethod, topupInfo, topupAmount])

  const handleSubscriptionAvailabilityChange = useCallback(
    (available: boolean) => {
      setShowSubscriptionPanel(available)
    },
    []
  )
  const availableUserQuota =
    user?.total_available_quota ?? user?.total_quota ?? user?.quota

  return (
    <>
      <SectionPageLayout>
        <SectionPageLayout.Title>{t('Wallet')}</SectionPageLayout.Title>
        <SectionPageLayout.Content>
          <div className='mx-auto flex w-full max-w-7xl flex-col gap-4 sm:gap-5'>
            <WalletStatsCard user={user} loading={userLoading} />

            {!personalTopupAllowed ? (
              <TeamWalletNotice
                onOpenTeam={() => navigate({ to: '/team-management' })}
                onOpenHistory={() => setBillingDialogOpen(true)}
              />
            ) : (
              <>
                <div
                  className={
                    showSubscriptionPanel
                      ? 'grid gap-4 xl:grid-cols-[minmax(0,1.05fr)_minmax(360px,0.95fr)] xl:items-start'
                      : 'grid gap-4'
                  }
                >
                  <div id='wallet-add-funds' className='scroll-mt-4'>
                    <RechargeFormCard
                      topupInfo={topupInfo}
                      presetAmounts={presetAmounts}
                      selectedPreset={selectedPreset}
                      onSelectPreset={handleSelectPreset}
                      topupAmount={topupAmount}
                      onTopupAmountChange={handleTopupAmountChange}
                      paymentAmount={paymentAmount}
                      calculating={calculating}
                      onPaymentMethodSelect={handlePaymentMethodSelect}
                      paymentLoading={paymentLoading}
                      redemptionCode={redemptionCode}
                      onRedemptionCodeChange={setRedemptionCode}
                      onRedeem={handleRedeem}
                      redeeming={redeeming}
                      topupLink={topupInfo?.topup_link}
                      loading={topupLoading}
                      priceRatio={(status?.price as number) || 1}
                      usdExchangeRate={effectiveUsdExchangeRate}
                      onOpenBilling={() => setBillingDialogOpen(true)}
                      creemProducts={topupInfo?.creem_products}
                      enableCreemTopup={topupInfo?.enable_creem_topup}
                      onCreemProductSelect={handleCreemProductSelect}
                      enableWaffoTopup={topupInfo?.enable_waffo_topup}
                      waffoPayMethods={topupInfo?.waffo_pay_methods}
                      waffoMinTopup={topupInfo?.waffo_min_topup}
                      onWaffoMethodSelect={handleWaffoMethodSelect}
                      enableWaffoPancakeTopup={
                        topupInfo?.enable_waffo_pancake_topup
                      }
                    />
                  </div>

                  <SubscriptionPlansCard
                    topupInfo={topupInfo}
                    onAvailabilityChange={handleSubscriptionAvailabilityChange}
                    userQuota={availableUserQuota}
                    onPurchaseSuccess={fetchUser}
                  />
                </div>

                <AffiliateRewardsCard
                  user={user}
                  affiliateLink={affiliateLink}
                  onTransfer={() => setTransferDialogOpen(true)}
                  complianceConfirmed={
                    topupInfo?.payment_compliance_confirmed !== false
                  }
                  loading={affiliateLoading}
                />
              </>
            )}
          </div>
        </SectionPageLayout.Content>
      </SectionPageLayout>

      <PaymentConfirmDialog
        open={confirmDialogOpen}
        onOpenChange={setConfirmDialogOpen}
        onConfirm={handlePaymentConfirm}
        topupAmount={topupAmount}
        paymentAmount={paymentAmount}
        paymentMethod={selectedPaymentMethod}
        calculating={calculating}
        processing={processing || waffoProcessing || pancakeProcessing}
        discountRate={getDiscountRate()}
        usdExchangeRate={effectiveUsdExchangeRate}
        currencyCode={
          selectedPaymentMethod && isStripePayment(selectedPaymentMethod.type)
            ? 'USD'
            : undefined
        }
        stripeCheckoutMethod={selectedPaymentMethod?.checkout_method}
      />

      <TransferDialog
        open={transferDialogOpen}
        onOpenChange={setTransferDialogOpen}
        onConfirm={handleTransfer}
        availableQuota={user?.aff_quota ?? 0}
        transferring={transferring}
      />

      <BillingHistoryDialog
        open={billingDialogOpen}
        onOpenChange={setBillingDialogOpen}
      />

      <CreemConfirmDialog
        open={creemDialogOpen}
        onOpenChange={setCreemDialogOpen}
        onConfirm={handleCreemConfirm}
        product={selectedCreemProduct}
        processing={creemProcessing}
      />
    </>
  )
}
