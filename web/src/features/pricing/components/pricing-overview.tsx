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
import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  Building2,
  Check,
  CircleDollarSign,
  Sparkles,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

const payAsYouGoBenefits = [
  'Discounted rates on every enabled model',
  'Text and image generation models',
  'Full OpenAI-compatible API',
  'No minimum spend or subscription',
  'Real-time usage logs and spend controls',
] as const

const enterpriseBenefits = [
  'Everything in Pay as you go',
  'Volume discounts and custom pricing',
  'High or dedicated rate limits',
  'Priority support and onboarding',
  'Corporate invoicing and bank transfer',
] as const

const comparisonRows = [
  {
    feature: 'Platform fee',
    personal: 'None',
    enterprise: 'None',
  },
  {
    feature: 'Billing',
    personal: 'Pay only for what you use',
    enterprise: 'Custom commercial terms',
  },
  {
    feature: 'Minimum spend',
    personal: 'None',
    enterprise: 'Custom',
  },
  {
    feature: 'Models',
    personal: 'All enabled text and image models',
    enterprise: 'Everything in Pay as you go',
  },
  {
    feature: 'API access',
    personal: 'Full OpenAI-compatible API',
    enterprise: 'Full OpenAI-compatible API',
  },
  {
    feature: 'Rate limits',
    personal: 'Standard shared limits',
    enterprise: 'High or dedicated rate limits',
  },
  {
    feature: 'Usage logs and spend controls',
    personal: 'Included',
    enterprise: 'Included',
  },
  {
    feature: 'Support',
    personal: 'Standard support',
    enterprise: 'Priority support',
  },
  {
    feature: 'Invoicing',
    personal: 'Available for eligible spending',
    enterprise: 'Corporate invoicing and bank transfer',
  },
] as const

function BenefitList(props: { items: readonly string[] }) {
  const { t } = useTranslation()

  return (
    <ul className='grid gap-3'>
      {props.items.map((item) => (
        <li key={item} className='flex items-start gap-2.5 text-sm'>
          <span className='mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-emerald-500/10 text-emerald-600 dark:text-emerald-400'>
            <Check className='size-3.5' aria-hidden='true' />
          </span>
          <span className='text-muted-foreground'>{t(item)}</span>
        </li>
      ))}
    </ul>
  )
}

export function PricingOverview() {
  const { t } = useTranslation()

  return (
    <section
      aria-labelledby='pricing-overview-title'
      className='mx-auto max-w-6xl'
    >
      <header className='mx-auto max-w-3xl text-center'>
        <div className='border-border bg-card/70 text-muted-foreground mx-auto mb-5 inline-flex items-center gap-2 rounded-full border px-3.5 py-1.5 text-xs font-medium shadow-sm backdrop-blur'>
          <Sparkles className='size-3.5 text-blue-500' aria-hidden='true' />
          {t('Pricing')}
        </div>
        <h1
          id='pricing-overview-title'
          className='text-[clamp(2.35rem,6vw,4.5rem)] leading-[1.04] font-black tracking-[-0.05em]'
        >
          {t('Simple, transparent pricing.')}
        </h1>
        <p className='text-muted-foreground mx-auto mt-5 max-w-2xl text-sm leading-7 sm:text-lg'>
          {t(
            'No platform fees, subscriptions, or minimum spend. Top up your wallet and pay only for successful usage.'
          )}
        </p>
        <p
          data-pricing-currency
          className='text-muted-foreground/80 mt-3 text-xs font-medium tracking-wide'
        >
          {t('All prices are displayed in USD unless stated otherwise.')}
        </p>
      </header>

      <div className='mt-10 grid gap-5 md:grid-cols-2' data-pricing-plan-grid>
        <Card className='border-blue-500/40 bg-gradient-to-b from-blue-500/[0.08] to-transparent shadow-xl shadow-blue-950/5'>
          <CardHeader className='gap-3 px-6 pt-4 sm:px-8 sm:pt-6'>
            <div className='flex items-start justify-between gap-4'>
              <div className='flex size-11 shrink-0 items-center justify-center rounded-xl bg-blue-500/10 text-blue-600 dark:text-blue-400'>
                <CircleDollarSign className='size-6' aria-hidden='true' />
              </div>
              <span
                data-pricing-popular-badge
                className='shrink-0 rounded-full bg-blue-600 px-3 py-1 text-[11px] font-semibold tracking-wide text-white uppercase shadow-sm'
              >
                {t('Most popular')}
              </span>
            </div>
            <div>
              <CardTitle className='text-xl'>{t('Pay as you go')}</CardTitle>
              <CardDescription className='mt-1.5 leading-6'>
                {t(
                  'Top up any amount and start using models immediately. Your balance is deducted according to the live discounted rates below.'
                )}
              </CardDescription>
            </div>
            <div className='flex items-end gap-2 pt-1'>
              <strong className='text-4xl tracking-tight'>$0</strong>
              <span className='text-muted-foreground pb-1 text-sm'>
                {t('platform fee')}
              </span>
            </div>
          </CardHeader>
          <CardContent className='flex flex-1 flex-col gap-6 px-6 pb-3 sm:px-8 sm:pb-5'>
            <BenefitList items={payAsYouGoBenefits} />
            <Button
              size='lg'
              className='mt-auto h-11 w-full'
              render={<Link to='/wallet' />}
            >
              {t('Get started')}
              <ArrowRight aria-hidden='true' />
            </Button>
          </CardContent>
        </Card>

        <Card className='border-violet-500/25 bg-gradient-to-b from-violet-500/[0.06] to-transparent'>
          <CardHeader className='gap-3 px-6 pt-4 sm:px-8 sm:pt-6'>
            <div className='flex size-11 items-center justify-center rounded-xl bg-violet-500/10 text-violet-600 dark:text-violet-400'>
              <Building2 className='size-6' aria-hidden='true' />
            </div>
            <div>
              <CardTitle className='text-xl'>{t('Enterprise')}</CardTitle>
              <CardDescription className='mt-1.5 leading-6'>
                {t(
                  'Built for teams with higher volume, dedicated capacity, tailored commercial terms, and hands-on support.'
                )}
              </CardDescription>
            </div>
            <strong className='pt-1 text-4xl tracking-tight'>
              {t('Custom')}
            </strong>
          </CardHeader>
          <CardContent className='flex flex-1 flex-col gap-6 px-6 pb-3 sm:px-8 sm:pb-5'>
            <BenefitList items={enterpriseBenefits} />
            <Button
              size='lg'
              variant='outline'
              className='mt-auto h-11 w-full bg-transparent'
              render={<Link to='/enterprise' />}
            >
              {t('Get enterprise pricing')}
              <ArrowRight aria-hidden='true' />
            </Button>
          </CardContent>
        </Card>
      </div>

      <section aria-labelledby='compare-plans-title' className='mt-14 sm:mt-18'>
        <div className='mb-5'>
          <h2
            id='compare-plans-title'
            className='text-2xl font-bold tracking-tight sm:text-3xl'
          >
            {t('Compare plans')}
          </h2>
          <p className='text-muted-foreground mt-2 text-sm sm:text-base'>
            {t(
              'Start with flexible pay-as-you-go pricing, then move to a tailored enterprise plan when your team needs more.'
            )}
          </p>
        </div>
        <div className='border-border bg-card/70 overflow-x-auto rounded-2xl border shadow-sm backdrop-blur'>
          <table className='w-full min-w-[720px] border-collapse text-left text-sm'>
            <thead>
              <tr className='border-border bg-muted/35 border-b'>
                <th scope='col' className='w-[34%] px-5 py-4 font-semibold'>
                  {t('Feature')}
                </th>
                <th scope='col' className='px-5 py-4 font-semibold'>
                  {t('Pay as you go')}
                </th>
                <th scope='col' className='px-5 py-4 font-semibold'>
                  {t('Enterprise')}
                </th>
              </tr>
            </thead>
            <tbody>
              {comparisonRows.map((row) => (
                <tr
                  key={row.feature}
                  className='border-border/70 border-b last:border-b-0'
                >
                  <th
                    scope='row'
                    className='text-foreground px-5 py-4 font-medium'
                  >
                    {t(row.feature)}
                  </th>
                  <td className='text-muted-foreground px-5 py-4'>
                    {t(row.personal)}
                  </td>
                  <td className='text-muted-foreground px-5 py-4'>
                    {t(row.enterprise)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </section>
  )
}
