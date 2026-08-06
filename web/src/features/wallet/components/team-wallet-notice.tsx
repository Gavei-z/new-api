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
import { CreditCard, Receipt, UsersRound } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { TitledCard } from '@/components/ui/titled-card'

interface TeamWalletNoticeProps {
  onOpenTeam: () => void
  onOpenHistory: () => void
}

export function TeamWalletNotice(props: TeamWalletNoticeProps) {
  const { t } = useTranslation()

  return (
    <TitledCard
      title={t('Team Funding')}
      description={t('Your API usage is funded by the shared team balance.')}
      icon={<UsersRound className='size-4' />}
      iconTone='info'
      disableHoverEffect
      action={
        <Button
          type='button'
          variant='outline'
          size='sm'
          onClick={props.onOpenHistory}
        >
          <Receipt className='size-4' />
          {t('Personal Order History')}
        </Button>
      }
    >
      <div className='bg-muted/30 flex flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between'>
        <p className='text-muted-foreground max-w-2xl text-sm'>
          {t(
            'Personal top-up is disabled for team identities. Team administrators can fund the shared balance directly with Stripe.'
          )}
        </p>
        <Button type='button' className='shrink-0' onClick={props.onOpenTeam}>
          <CreditCard className='size-4' />
          {t('Open Team Funding')}
        </Button>
      </div>
    </TitledCard>
  )
}
