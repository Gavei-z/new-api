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
import { useQuery } from '@tanstack/react-query'
import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Card, CardContent } from '@/components/ui/card'
import { PAYMENT_RETURN_REFRESH_DELAYS_MS } from '@/features/wallet/constants'
import { useTopupInfo } from '@/features/wallet/hooks'

import { getCurrentTeam } from '../api'
import { TeamDetail } from './team-detail'

interface TeamManagerConsoleProps {
  initialStripeStatus?: 'success' | 'cancel'
}

export function TeamManagerConsole(props: TeamManagerConsoleProps) {
  const { t } = useTranslation()
  const { topupInfo } = useTopupInfo()
  const handledStripeStatusRef = useRef(false)
  const teamQuery = useQuery({
    queryKey: ['current-team'],
    queryFn: getCurrentTeam,
  })
  const refetchTeam = teamQuery.refetch

  useEffect(() => {
    if (!props.initialStripeStatus || handledStripeStatusRef.current) return
    handledStripeStatusRef.current = true

    const refreshTimers: number[] = []

    if (props.initialStripeStatus === 'success') {
      toast.success(
        t(
          'Returned from Stripe. We are confirming the team payment and refreshing the shared balance.'
        )
      )
      PAYMENT_RETURN_REFRESH_DELAYS_MS.forEach((delay) => {
        refreshTimers.push(
          window.setTimeout(() => {
            void refetchTeam()
          }, delay)
        )
      })
    } else {
      toast.info(t('Team Stripe top-up was cancelled. No charge was made.'))
    }
    window.history.replaceState({}, '', window.location.pathname)

    return () => {
      refreshTimers.forEach((timer) => window.clearTimeout(timer))
      handledStripeStatusRef.current = false
    }
  }, [props.initialStripeStatus, refetchTeam, t])

  if (teamQuery.isLoading) {
    return (
      <Card>
        <CardContent className='text-muted-foreground flex min-h-64 items-center justify-center'>
          {t('Loading...')}
        </CardContent>
      </Card>
    )
  }

  if (!teamQuery.data) {
    return (
      <Card>
        <CardContent className='text-muted-foreground flex min-h-64 items-center justify-center text-center'>
          {t('Your account is not an active team administrator.')}
        </CardContent>
      </Card>
    )
  }

  return (
    <TeamDetail
      mode='manager'
      team={teamQuery.data.team}
      summary={teamQuery.data}
      topupInfo={topupInfo}
    />
  )
}
