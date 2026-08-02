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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Building2, Plus, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { formatQuota, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import {
  adminCreateTeam,
  adminListEnterpriseInquiries,
  adminListTeams,
  adminUpdateEnterpriseInquiryStatus,
} from '../api'
import { getErrorMessage, teamStatusLabel } from '../lib'
import type { CreateTeamInput } from '../types'
import { TeamDetail } from './team-detail'
import { CreateTeamDialog } from './team-dialogs'

function inquiryStatusLabel(
  t: ReturnType<typeof useTranslation>['t'],
  status: number
) {
  if (status === 2) return t('Contacted')
  if (status === 3) return t('Closed')
  return t('Pending')
}

export function RootTeamConsole() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [selectedTeamId, setSelectedTeamId] = useState<number | null>(null)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [inquiryChangingId, setInquiryChangingId] = useState<number | null>(
    null
  )
  const teamsQuery = useQuery({
    queryKey: ['admin-teams'],
    queryFn: adminListTeams,
  })
  const inquiriesQuery = useQuery({
    queryKey: ['enterprise-inquiries'],
    queryFn: adminListEnterpriseInquiries,
  })

  const teams = teamsQuery.data ?? []
  const selectedTeam =
    teams.find((team) => team.id === selectedTeamId) ?? teams[0]

  useEffect(() => {
    if (selectedTeam && selectedTeam.id !== selectedTeamId) {
      setSelectedTeamId(selectedTeam.id)
    }
  }, [selectedTeam, selectedTeamId])

  const createTeam = async (values: CreateTeamInput) => {
    await adminCreateTeam(values)
    await queryClient.invalidateQueries({ queryKey: ['admin-teams'] })
  }

  const updateInquiryStatus = async (inquiryId: number, status: number) => {
    setInquiryChangingId(inquiryId)
    try {
      await adminUpdateEnterpriseInquiryStatus(inquiryId, status)
      toast.success(t('Inquiry status updated'))
      await queryClient.invalidateQueries({
        queryKey: ['enterprise-inquiries'],
      })
    } catch (error) {
      toast.error(getErrorMessage(error, t('Failed to update inquiry status')))
    } finally {
      setInquiryChangingId(null)
    }
  }

  return (
    <>
      <Tabs defaultValue='teams' className='space-y-3'>
        <div className='flex flex-wrap items-center justify-between gap-3'>
          <TabsList variant='line'>
            <TabsTrigger value='teams'>{t('Enterprise Teams')}</TabsTrigger>
            <TabsTrigger value='inquiries'>
              {t('Enterprise Inquiries')}
            </TabsTrigger>
          </TabsList>
          <Button onClick={() => setCreateDialogOpen(true)}>
            <Plus />
            {t('Create Team')}
          </Button>
        </div>

        <TabsContent value='teams' className='space-y-4'>
          <div className='grid gap-4 xl:grid-cols-[17rem_minmax(0,1fr)]'>
            <Card className='h-fit'>
              <CardHeader>
                <CardTitle>{t('All Teams')}</CardTitle>
                <CardDescription>
                  {t('{{count}} isolated tenants', { count: teams.length })}
                </CardDescription>
              </CardHeader>
              <CardContent className='space-y-2'>
                {teams.map((team) => (
                  <button
                    key={team.id}
                    type='button'
                    className={cn(
                      'border-border hover:bg-muted/60 w-full rounded-lg border p-3 text-left transition-colors',
                      selectedTeam?.id === team.id &&
                        'border-primary bg-primary/5'
                    )}
                    onClick={() => setSelectedTeamId(team.id)}
                  >
                    <div className='flex items-start justify-between gap-2'>
                      <div className='min-w-0'>
                        <div className='truncate font-medium'>{team.name}</div>
                        <div className='text-muted-foreground truncate text-xs'>
                          {team.slug}
                        </div>
                      </div>
                      <Badge
                        variant={
                          team.status === 1 ? 'secondary' : 'destructive'
                        }
                      >
                        {teamStatusLabel(t, team.status)}
                      </Badge>
                    </div>
                    <div className='text-muted-foreground mt-3 flex items-center justify-between text-xs'>
                      <span>{t('Balance')}</span>
                      <span className='text-foreground font-medium'>
                        {formatQuota(team.quota)}
                      </span>
                    </div>
                  </button>
                ))}
                {teamsQuery.isLoading && (
                  <p className='text-muted-foreground py-8 text-center text-sm'>
                    {t('Loading...')}
                  </p>
                )}
                {!teamsQuery.isLoading && teams.length === 0 && (
                  <div className='text-muted-foreground py-8 text-center text-sm'>
                    <Building2 className='mx-auto mb-2 size-8 opacity-50' />
                    {t('No enterprise teams yet')}
                  </div>
                )}
              </CardContent>
            </Card>

            {selectedTeam ? (
              <TeamDetail
                key={selectedTeam.id}
                mode='root'
                team={selectedTeam}
              />
            ) : (
              <Card className='min-h-64'>
                <CardContent className='text-muted-foreground flex flex-1 flex-col items-center justify-center text-center'>
                  <Users className='mb-3 size-10 opacity-50' />
                  <p>{t('Create a team to begin enterprise management.')}</p>
                </CardContent>
              </Card>
            )}
          </div>
        </TabsContent>

        <TabsContent value='inquiries'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Enterprise Quote Inquiries')}</CardTitle>
              <CardDescription>
                {t(
                  'Contact details submitted through the public enterprise service page.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='px-0'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className='pl-4'>{t('Company')}</TableHead>
                    <TableHead>{t('Contact')}</TableHead>
                    <TableHead>{t('Team Size')}</TableHead>
                    <TableHead>{t('Expected Monthly Usage')}</TableHead>
                    <TableHead>{t('Submitted')}</TableHead>
                    <TableHead className='pr-4'>{t('Status')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {(inquiriesQuery.data?.items ?? []).map((inquiry) => (
                    <TableRow key={inquiry.id}>
                      <TableCell className='pl-4'>
                        <div className='font-medium'>
                          {inquiry.company_name}
                        </div>
                        <div
                          className='text-muted-foreground max-w-64 truncate text-xs'
                          title={inquiry.message}
                        >
                          {inquiry.message || '-'}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div>{inquiry.contact_name}</div>
                        <div className='text-muted-foreground text-xs'>
                          {[inquiry.email, inquiry.phone, inquiry.wechat]
                            .filter(Boolean)
                            .join(' · ')}
                        </div>
                      </TableCell>
                      <TableCell>{inquiry.team_size || '-'}</TableCell>
                      <TableCell>
                        {inquiry.expected_monthly_usage || '-'}
                      </TableCell>
                      <TableCell>
                        {formatTimestampToDate(inquiry.created_at)}
                      </TableCell>
                      <TableCell className='pr-4'>
                        <NativeSelect
                          size='sm'
                          value={String(inquiry.status)}
                          disabled={inquiryChangingId === inquiry.id}
                          aria-label={t('Inquiry Status')}
                          onChange={(event) =>
                            updateInquiryStatus(
                              inquiry.id,
                              Number(event.target.value)
                            )
                          }
                        >
                          <NativeSelectOption value='1'>
                            {inquiryStatusLabel(t, 1)}
                          </NativeSelectOption>
                          <NativeSelectOption value='2'>
                            {inquiryStatusLabel(t, 2)}
                          </NativeSelectOption>
                          <NativeSelectOption value='3'>
                            {inquiryStatusLabel(t, 3)}
                          </NativeSelectOption>
                        </NativeSelect>
                      </TableCell>
                    </TableRow>
                  ))}
                  {!inquiriesQuery.isLoading &&
                    (inquiriesQuery.data?.items.length ?? 0) === 0 && (
                      <TableRow>
                        <TableCell
                          colSpan={6}
                          className='text-muted-foreground h-28 text-center'
                        >
                          {t('No enterprise inquiries')}
                        </TableCell>
                      </TableRow>
                    )}
                  {inquiriesQuery.isLoading && (
                    <TableRow>
                      <TableCell
                        colSpan={6}
                        className='text-muted-foreground h-28 text-center'
                      >
                        {t('Loading...')}
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <CreateTeamDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        onSubmit={createTeam}
      />
    </>
  )
}
