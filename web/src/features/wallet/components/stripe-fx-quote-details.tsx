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
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import type { StripeFxQuote } from '../types'

interface StripeFxQuoteDetailsProps {
  quote: Readonly<StripeFxQuote>
  className?: string
}

export function StripeFxQuoteDetails(props: StripeFxQuoteDetailsProps) {
  const { t } = useTranslation()
  const normalizedSource = props.quote.source.toLowerCase().replaceAll('_', ' ')
  const sourceLabel =
    normalizedSource === 'boc spot selling' ||
    normalizedSource.includes('bank of china')
      ? t('Bank of China spot exchange selling rate')
      : props.quote.source

  return (
    <div
      data-slot='stripe-fx-quote-details'
      className={cn('text-muted-foreground space-y-1 text-xs', props.className)}
    >
      <p className='text-foreground font-medium'>
        {t('1 USD = ¥{{rate}} CNY', {
          rate: props.quote.exchange_rate,
        })}
      </p>
      <p>
        {t('Exchange rate source: {{source}}', {
          source: sourceLabel,
        })}
      </p>
      <p>
        {t('Pricing date: {{date}}', {
          date: props.quote.pricing_date,
        })}
      </p>
    </div>
  )
}
