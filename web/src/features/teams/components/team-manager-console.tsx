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
import { useTranslation } from 'react-i18next'

import { Card, CardContent } from '@/components/ui/card'

import { getCurrentTeam } from '../api'
import { TeamDetail } from './team-detail'

export function TeamManagerConsole() {
  const { t } = useTranslation()
  const teamQuery = useQuery({
    queryKey: ['current-team'],
    queryFn: getCurrentTeam,
  })

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
    />
  )
}
