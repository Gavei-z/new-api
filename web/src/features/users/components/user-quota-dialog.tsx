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
import { useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { getCurrencyDisplay, getCurrencyLabel } from '@/lib/currency'
import { formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import { adjustUserQuota } from '../api'
import {
  createQuotaAdjustmentIdempotencyKey,
  getQuotaAdjustmentValue,
} from '../lib/quota-adjustment'
import type { QuotaAdjustMode } from '../types'

interface UserQuotaDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  userId: number
  currentQuota: number
  onSuccess: () => void
}

export function UserQuotaDialog(props: UserQuotaDialogProps) {
  const { t } = useTranslation()
  const [mode, setMode] = useState<QuotaAdjustMode>('add')
  const [amount, setAmount] = useState('')
  const [loading, setLoading] = useState(false)
  const idempotencyKeyRef = useRef<string | null>(null)

  const { meta: currencyMeta } = getCurrencyDisplay()
  const currencyLabel = getCurrencyLabel()
  const tokensOnly = currencyMeta.kind === 'tokens'

  const parsedQuotaValue = getQuotaAdjustmentValue(amount, mode)
  const quotaValue = parsedQuotaValue ?? 0
  const amountIsValid = parsedQuotaValue !== null

  const getPreviewText = () => {
    const current = props.currentQuota
    const val = quotaValue
    switch (mode) {
      case 'add':
        return `${t('Current quota')}: ${formatQuota(current)}  +${formatQuota(val)} = ${formatQuota(current + val)}`
      case 'subtract':
        return `${t('Current quota')}: ${formatQuota(current)}  -${formatQuota(val)} = ${formatQuota(current - val)}`
      case 'override': {
        return `${t('Current quota')}: ${formatQuota(current)} → ${formatQuota(quotaValue)}`
      }
      default:
        return ''
    }
  }

  const handleConfirm = async () => {
    if (loading) return
    if (!amountIsValid || parsedQuotaValue === null) return

    const idempotencyKey =
      idempotencyKeyRef.current ?? createQuotaAdjustmentIdempotencyKey()
    idempotencyKeyRef.current = idempotencyKey
    setLoading(true)
    try {
      const result = await adjustUserQuota({
        id: props.userId,
        action: 'add_quota',
        mode,
        value: parsedQuotaValue,
        idempotency_key: idempotencyKey,
      })
      if (result.success) {
        if (result.data?.cache_invalidation_failed) {
          toast.warning(
            t('Quota was adjusted, but cache synchronization is still pending')
          )
        } else {
          toast.success(t('Quota adjusted successfully'))
        }
        idempotencyKeyRef.current = null
        setAmount('')
        setMode('add')
        props.onOpenChange(false)
        props.onSuccess()
      } else {
        toast.error(result.message || t('Failed to adjust quota'))
      }
    } catch (e: unknown) {
      toast.error(e instanceof Error ? e.message : t('Failed to adjust quota'))
    } finally {
      setLoading(false)
    }
  }

  const handleCancel = () => {
    if (loading) return
    idempotencyKeyRef.current = null
    setAmount('')
    setMode('add')
    props.onOpenChange(false)
  }

  const placeholder = tokensOnly
    ? t('Enter amount in tokens')
    : t('Enter amount in {{currency}}', { currency: currencyLabel })
  const modeLabels: Record<QuotaAdjustMode, string> = {
    add: t('Add'),
    subtract: t('Subtract'),
    override: t('Override'),
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={(open) => {
        if (!open && loading) return
        if (!open) idempotencyKeyRef.current = null
        props.onOpenChange(open)
      }}
      title={t('Adjust Quota')}
      description={t('Select an operation mode and enter the amount')}
      showCloseButton={!loading}
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        <>
          <Button variant='outline' onClick={handleCancel} disabled={loading}>
            {t('Cancel')}
          </Button>
          <Button onClick={handleConfirm} disabled={loading || !amountIsValid}>
            {loading ? t('Processing...') : t('Confirm')}
          </Button>
        </>
      }
    >
      <div className='space-y-4'>
        <div className='text-muted-foreground text-sm'>{getPreviewText()}</div>

        <div className='space-y-2'>
          <Label>{t('Mode')}</Label>
          <div className='flex gap-1'>
            {(['add', 'subtract', 'override'] as const).map((m) => (
              <Button
                key={m}
                type='button'
                variant='outline'
                size='sm'
                disabled={loading}
                className={cn(
                  mode === m &&
                    'bg-primary text-primary-foreground hover:bg-primary/90 hover:text-primary-foreground'
                )}
                onClick={() => {
                  idempotencyKeyRef.current = null
                  setMode(m)
                  setAmount('')
                }}
              >
                {modeLabels[m]}
              </Button>
            ))}
          </div>
        </div>

        <div className='space-y-2'>
          <Label>
            {t('Amount')} ({currencyLabel})
          </Label>
          <Input
            type='number'
            step={tokensOnly ? 1 : 0.000001}
            min={0}
            disabled={loading}
            placeholder={placeholder}
            value={amount}
            onChange={(e) => {
              idempotencyKeyRef.current = null
              setAmount(e.target.value)
            }}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handleConfirm()
            }}
          />
        </div>
      </div>
    </Dialog>
  )
}
