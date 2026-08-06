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
import {
  BanknoteArrowUp,
  CircleDollarSign,
  CreditCard,
  KeyRound,
  Plus,
  ShieldCheck,
  UserRoundCheck,
  Users,
} from 'lucide-react'
import { useState } from 'react'
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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import type { TopupInfo } from '@/features/wallet/types'
import { formatNumber, formatQuota, formatTimestampToDate } from '@/lib/format'
import { useAuthStore } from '@/stores/auth-store'

import {
  adminAdjustTeamQuota,
  adminCreateTeamMember,
  adminGetTeamMembers,
  adminGetTeamTransactions,
  adminGetTeamUsage,
  adminResetTeamMemberPassword,
  adminUpdateTeamMemberStatus,
  adminUpdateTeamStatus,
  createCurrentTeamMember,
  fundCurrentTeam,
  getCurrentTeamMembers,
  getCurrentTeamTransactions,
  getCurrentTeamUsage,
  resetCurrentTeamMemberPassword,
  updateCurrentTeamMemberStatus,
} from '../api'
import {
  createIdempotencyKey,
  getErrorMessage,
  teamRoleLabel,
  teamStatusLabel,
  transactionTypeLabel,
} from '../lib'
import type {
  CreateTeamMemberInput,
  Team,
  TeamMember,
  TeamSummary,
} from '../types'
import {
  CreateMemberDialog,
  QuotaDialog,
  ResetPasswordDialog,
} from './team-dialogs'
import { TeamStripeTopupDialog } from './team-stripe-topup-dialog'

type TeamDetailProps =
  | {
      mode: 'root'
      team: Team
      summary?: never
    }
  | {
      mode: 'manager'
      team: Team
      summary: TeamSummary
      topupInfo: TopupInfo | null
    }

function signedQuota(value: number): string {
  if (value === 0) return formatQuota(0)
  return `${value > 0 ? '+' : '-'}${formatQuota(Math.abs(value))}`
}

function EmptyTableRow(props: { columns: number; text: string }) {
  return (
    <TableRow>
      <TableCell
        colSpan={props.columns}
        className='text-muted-foreground h-24 text-center'
      >
        {props.text}
      </TableCell>
    </TableRow>
  )
}

export function TeamDetail(props: TeamDetailProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const currentUserId = useAuthStore((state) => state.auth.user?.id)
  const [memberDialogOpen, setMemberDialogOpen] = useState(false)
  const [quotaDialogOpen, setQuotaDialogOpen] = useState(false)
  const [stripeDialogOpen, setStripeDialogOpen] = useState(false)
  const [resetTarget, setResetTarget] = useState<TeamMember | null>(null)
  const [statusChangingId, setStatusChangingId] = useState<number | null>(null)
  const [usageStartTimestamp] = useState(() =>
    Math.floor(Date.now() / 1000 - 30 * 24 * 60 * 60)
  )
  const isRoot = props.mode === 'root'
  const queryScope = isRoot ? `root-${props.team.id}` : 'current'

  const membersQuery = useQuery({
    queryKey: ['team-members', queryScope],
    queryFn: () =>
      isRoot ? adminGetTeamMembers(props.team.id) : getCurrentTeamMembers(),
  })
  const usageQuery = useQuery({
    queryKey: ['team-usage', queryScope, usageStartTimestamp],
    queryFn: () =>
      isRoot
        ? adminGetTeamUsage(props.team.id, usageStartTimestamp)
        : getCurrentTeamUsage(usageStartTimestamp),
  })
  const transactionsQuery = useQuery({
    queryKey: ['team-transactions', queryScope],
    queryFn: () =>
      isRoot
        ? adminGetTeamTransactions(props.team.id)
        : getCurrentTeamTransactions(),
  })

  const invalidateTeamData = async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: ['team-members', queryScope],
      }),
      queryClient.invalidateQueries({
        queryKey: ['team-usage', queryScope],
      }),
      queryClient.invalidateQueries({
        queryKey: ['team-transactions', queryScope],
      }),
      queryClient.invalidateQueries({ queryKey: ['current-team'] }),
      queryClient.invalidateQueries({ queryKey: ['admin-teams'] }),
    ])
  }

  const createMember = async (values: CreateTeamMemberInput) => {
    if (isRoot) {
      await adminCreateTeamMember(props.team.id, values)
    } else {
      await createCurrentTeamMember(values)
    }
    await invalidateTeamData()
  }

  const updateMemberStatus = async (member: TeamMember) => {
    const nextStatus = member.member_status === 1 ? 2 : 1
    if (
      nextStatus === 2 &&
      !window.confirm(
        t('Disable {{username}} and revoke all active sessions?', {
          username: member.username,
        })
      )
    ) {
      return
    }
    setStatusChangingId(member.user_id)
    try {
      if (isRoot) {
        await adminUpdateTeamMemberStatus(
          props.team.id,
          member.user_id,
          nextStatus
        )
      } else {
        await updateCurrentTeamMemberStatus(member.user_id, nextStatus)
      }
      toast.success(t('Member status updated'))
      await invalidateTeamData()
    } catch (error) {
      toast.error(getErrorMessage(error, t('Failed to update member status')))
    } finally {
      setStatusChangingId(null)
    }
  }

  const resetPassword = async (password: string) => {
    if (!resetTarget) return
    if (isRoot) {
      await adminResetTeamMemberPassword(
        props.team.id,
        resetTarget.user_id,
        password
      )
    } else {
      await resetCurrentTeamMemberPassword(resetTarget.user_id, password)
    }
  }

  const updateQuota = async (values: {
    amount: number
    operation: 'add' | 'subtract'
    note: string
  }) => {
    if (isRoot) {
      await adminAdjustTeamQuota(
        props.team.id,
        values.operation,
        values.amount,
        values.note,
        createIdempotencyKey()
      )
    } else {
      await fundCurrentTeam(values.amount, createIdempotencyKey())
    }
    await invalidateTeamData()
  }

  const toggleTeamStatus = async () => {
    if (!isRoot) return
    const nextStatus = props.team.status === 1 ? 2 : 1
    if (
      nextStatus === 2 &&
      !window.confirm(
        t(
          'Disable this team? New API usage and team management actions will be blocked.'
        )
      )
    ) {
      return
    }
    try {
      await adminUpdateTeamStatus(props.team.id, nextStatus)
      toast.success(t('Team status updated'))
      await invalidateTeamData()
    } catch (error) {
      toast.error(getErrorMessage(error, t('Failed to update team status')))
    }
  }

  const actorRole =
    props.mode === 'manager' ? props.summary.role : Number.POSITIVE_INFINITY
  const canManageMember = (member: TeamMember) =>
    isRoot || (member.user_id !== currentUserId && member.role < actorRole)

  const members = membersQuery.data ?? []
  const usage = usageQuery.data ?? []
  const transactions = transactionsQuery.data?.items ?? []
  const availableTeamQuota =
    props.team.total_available_quota ??
    props.team.total_quota ??
    props.team.quota

  return (
    <div className='space-y-4'>
      <div className='flex flex-wrap items-start justify-between gap-3'>
        <div>
          <div className='flex flex-wrap items-center gap-2'>
            <h3 className='text-xl font-semibold'>{props.team.name}</h3>
            <Badge
              variant={props.team.status === 1 ? 'secondary' : 'destructive'}
            >
              {teamStatusLabel(t, props.team.status)}
            </Badge>
          </div>
          <p className='text-muted-foreground mt-1 text-sm'>
            {t('Tenant identifier')}: {props.team.slug}
          </p>
        </div>
        <div className='flex flex-wrap gap-2'>
          <Button variant='outline' onClick={() => setMemberDialogOpen(true)}>
            <Plus />
            {t('Create Member')}
          </Button>
          {!isRoot && (
            <Button
              onClick={() => setStripeDialogOpen(true)}
              disabled={props.topupInfo?.enable_stripe_topup !== true}
              title={
                props.topupInfo?.enable_stripe_topup === true
                  ? undefined
                  : t('Stripe top-up is not available.')
              }
            >
              <CreditCard />
              {t('Add Team Funds')}
            </Button>
          )}
          <Button
            variant={isRoot ? 'default' : 'outline'}
            onClick={() => setQuotaDialogOpen(true)}
          >
            {isRoot ? <CircleDollarSign /> : <BanknoteArrowUp />}
            {isRoot
              ? t('Adjust Balance')
              : t('Transfer Existing Personal Balance')}
          </Button>
          {isRoot && (
            <Button
              variant={props.team.status === 1 ? 'destructive' : 'outline'}
              onClick={toggleTeamStatus}
            >
              <ShieldCheck />
              {props.team.status === 1 ? t('Disable Team') : t('Enable Team')}
            </Button>
          )}
        </div>
      </div>

      <div className='grid gap-3 sm:grid-cols-2 xl:grid-cols-4'>
        <Card size='sm'>
          <CardHeader>
            <CardDescription>{t('Available Team Balance')}</CardDescription>
            <CardTitle className='text-2xl'>
              {formatQuota(availableTeamQuota)}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card size='sm'>
          <CardHeader>
            <CardDescription>{t('Total Team Usage')}</CardDescription>
            <CardTitle className='text-2xl'>
              {formatQuota(props.team.used_quota)}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card size='sm'>
          <CardHeader>
            <CardDescription>{t('Team Members')}</CardDescription>
            <CardTitle className='flex items-center gap-2 text-2xl'>
              <Users className='text-muted-foreground size-5' />
              {formatNumber(members.length)}
            </CardTitle>
          </CardHeader>
        </Card>
        <Card size='sm'>
          <CardHeader>
            <CardDescription>{t('Your Team Role')}</CardDescription>
            <CardTitle className='flex items-center gap-2 text-base'>
              <UserRoundCheck className='text-muted-foreground size-5' />
              {isRoot ? t('Super Admin') : teamRoleLabel(t, actorRole)}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      <Tabs defaultValue='members'>
        <TabsList variant='line'>
          <TabsTrigger value='members'>{t('Team Members')}</TabsTrigger>
          <TabsTrigger value='usage'>{t('Team Usage')}</TabsTrigger>
          <TabsTrigger value='ledger'>{t('Balance Ledger')}</TabsTrigger>
        </TabsList>

        <TabsContent value='members'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Team Members')}</CardTitle>
              <CardDescription>
                {t(
                  'Member accounts remain ordinary users globally and cannot access channels or system settings.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='px-0'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className='pl-4'>{t('User')}</TableHead>
                    <TableHead>{t('Role')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                    <TableHead>{t('Requests')}</TableHead>
                    <TableHead>{t('Recorded User Usage')}</TableHead>
                    <TableHead className='pr-4 text-right'>
                      {t('Actions')}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {members.map((member) => (
                    <TableRow key={member.user_id}>
                      <TableCell className='pl-4'>
                        <div className='font-medium'>{member.username}</div>
                        <div className='text-muted-foreground text-xs'>
                          {member.display_name ||
                            member.email ||
                            `#${member.user_id}`}
                        </div>
                      </TableCell>
                      <TableCell>{teamRoleLabel(t, member.role)}</TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            member.member_status === 1
                              ? 'secondary'
                              : 'destructive'
                          }
                        >
                          {teamStatusLabel(t, member.member_status)}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {formatNumber(member.request_count)}
                      </TableCell>
                      <TableCell>{formatQuota(member.used_quota)}</TableCell>
                      <TableCell className='pr-4'>
                        <div className='flex justify-end gap-1'>
                          <Button
                            size='sm'
                            variant='ghost'
                            disabled={!canManageMember(member)}
                            onClick={() => setResetTarget(member)}
                          >
                            <KeyRound />
                            {t('Reset Password')}
                          </Button>
                          <Button
                            size='sm'
                            variant={
                              member.member_status === 1
                                ? 'destructive'
                                : 'outline'
                            }
                            disabled={
                              !canManageMember(member) ||
                              statusChangingId === member.user_id
                            }
                            onClick={() => updateMemberStatus(member)}
                          >
                            {member.member_status === 1
                              ? t('Disable')
                              : t('Enable')}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                  {!membersQuery.isLoading && members.length === 0 && (
                    <EmptyTableRow
                      columns={6}
                      text={t('No team members found')}
                    />
                  )}
                  {membersQuery.isLoading && (
                    <EmptyTableRow columns={6} text={t('Loading...')} />
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value='usage'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Team Usage')}</CardTitle>
              <CardDescription>
                {t(
                  'Consumption in the last 30 days, grouped by member and model from usage logs.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='px-0'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className='pl-4'>{t('Member')}</TableHead>
                    <TableHead>{t('Model')}</TableHead>
                    <TableHead>{t('Requests')}</TableHead>
                    <TableHead>{t('Prompt Tokens')}</TableHead>
                    <TableHead>{t('Completion Tokens')}</TableHead>
                    <TableHead className='pr-4'>{t('Usage')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {usage.map((row) => (
                    <TableRow key={`${row.user_id}-${row.model_name}`}>
                      <TableCell className='pl-4 font-medium'>
                        {row.username || `#${row.user_id}`}
                      </TableCell>
                      <TableCell>{row.model_name || '-'}</TableCell>
                      <TableCell>{formatNumber(row.request_count)}</TableCell>
                      <TableCell>{formatNumber(row.prompt_tokens)}</TableCell>
                      <TableCell>
                        {formatNumber(row.completion_tokens)}
                      </TableCell>
                      <TableCell className='pr-4'>
                        {formatQuota(row.quota)}
                      </TableCell>
                    </TableRow>
                  ))}
                  {!usageQuery.isLoading && usage.length === 0 && (
                    <EmptyTableRow columns={6} text={t('No usage recorded')} />
                  )}
                  {usageQuery.isLoading && (
                    <EmptyTableRow columns={6} text={t('Loading...')} />
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value='ledger'>
          <Card>
            <CardHeader>
              <CardTitle>{t('Balance Ledger')}</CardTitle>
              <CardDescription>
                {t(
                  'Immutable balance changes provide an audit trail for grants, usage, settlement, and refunds.'
                )}
              </CardDescription>
            </CardHeader>
            <CardContent className='px-0'>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className='pl-4'>{t('Time')}</TableHead>
                    <TableHead>{t('Type')}</TableHead>
                    <TableHead>{t('User ID')}</TableHead>
                    <TableHead>{t('Balance Change')}</TableHead>
                    <TableHead>{t('Balance After')}</TableHead>
                    <TableHead className='pr-4'>{t('Audit Note')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {transactions.map((transaction) => (
                    <TableRow key={transaction.id}>
                      <TableCell className='pl-4'>
                        {formatTimestampToDate(transaction.created_at)}
                      </TableCell>
                      <TableCell>
                        <div>{transactionTypeLabel(t, transaction.type)}</div>
                        {transaction.source === 'stripe' && (
                          <div className='text-muted-foreground text-xs'>
                            Stripe
                          </div>
                        )}
                      </TableCell>
                      <TableCell>
                        {transaction.user_id > 0 ? transaction.user_id : '-'}
                      </TableCell>
                      <TableCell
                        className={
                          transaction.quota_delta >= 0
                            ? 'text-emerald-600'
                            : 'text-destructive'
                        }
                      >
                        {signedQuota(transaction.quota_delta)}
                      </TableCell>
                      <TableCell>
                        {formatQuota(transaction.balance_after)}
                      </TableCell>
                      <TableCell className='text-muted-foreground max-w-60 truncate pr-4'>
                        {transaction.note || '-'}
                      </TableCell>
                    </TableRow>
                  ))}
                  {!transactionsQuery.isLoading &&
                    transactions.length === 0 && (
                      <EmptyTableRow
                        columns={6}
                        text={t('No balance transactions')}
                      />
                    )}
                  {transactionsQuery.isLoading && (
                    <EmptyTableRow columns={6} text={t('Loading...')} />
                  )}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <CreateMemberDialog
        open={memberDialogOpen}
        onOpenChange={setMemberDialogOpen}
        onSubmit={createMember}
      />
      <QuotaDialog
        kind={isRoot ? 'adjust' : 'fund'}
        open={quotaDialogOpen}
        onOpenChange={setQuotaDialogOpen}
        onSubmit={updateQuota}
      />
      {!isRoot && (
        <TeamStripeTopupDialog
          open={stripeDialogOpen}
          onOpenChange={setStripeDialogOpen}
          topupInfo={props.topupInfo}
        />
      )}
      <ResetPasswordDialog
        open={resetTarget !== null}
        onOpenChange={(open) => !open && setResetTarget(null)}
        username={resetTarget?.username ?? ''}
        onSubmit={resetPassword}
      />
    </div>
  )
}
