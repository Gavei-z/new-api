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

import { SectionPageLayout } from '@/components/layout'
import { ROLE } from '@/lib/roles'
import { useAuthStore } from '@/stores/auth-store'

import { RootTeamConsole } from './components/root-team-console'
import { TeamManagerConsole } from './components/team-manager-console'

export function Teams() {
  const { t } = useTranslation()
  const isRoot = useAuthStore(
    (state) => state.auth.user?.role === ROLE.SUPER_ADMIN
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Team Management')}</SectionPageLayout.Title>
      <SectionPageLayout.Content>
        {isRoot ? <RootTeamConsole /> : <TeamManagerConsole />}
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}
