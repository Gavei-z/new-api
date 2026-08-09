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
import type { StripeCheckoutMethod } from '@/features/wallet/types'
import { api } from '@/lib/api'

import {
  createTeamStripeAmountPayload,
  createTeamStripeTopupPayload,
} from './lib'
import type {
  CreateTeamInput,
  CreateTeamMemberInput,
  EnterpriseInquiry,
  EnterpriseInquiryInput,
  PageResult,
  Team,
  TeamMember,
  TeamQuotaTransaction,
  TeamSummary,
  TeamStripeCheckout,
  TeamStripeCheckoutRequest,
  TeamUsage,
} from './types'

function unwrap<T>(response: {
  data: { success?: boolean; data: T; message?: string }
}): T {
  const succeeded =
    response.data.success === true || response.data.message === 'success'
  if (!succeeded) {
    const dataMessage =
      typeof response.data.data === 'string' ? response.data.data : ''
    throw new Error(dataMessage || response.data.message || 'Request failed')
  }
  return response.data.data
}

export async function getCurrentTeam(): Promise<TeamSummary> {
  return unwrap(await api.get('/api/team'))
}

export async function getCurrentTeamMembers(): Promise<TeamMember[]> {
  return unwrap(await api.get('/api/team/members'))
}

export async function createCurrentTeamMember(
  input: CreateTeamMemberInput
): Promise<{ user_id: number; username: string }> {
  return unwrap(await api.post('/api/team/members', input))
}

export async function updateCurrentTeamMemberStatus(
  userId: number,
  status: number
): Promise<void> {
  unwrap(await api.patch(`/api/team/members/${userId}/status`, { status }))
}

export async function resetCurrentTeamMemberPassword(
  userId: number,
  password: string
): Promise<void> {
  unwrap(
    await api.post(`/api/team/members/${userId}/reset-password`, { password })
  )
}

export async function getCurrentTeamUsage(
  startTimestamp?: number
): Promise<TeamUsage[]> {
  return unwrap(
    await api.get('/api/team/usage', {
      params: { start_timestamp: startTimestamp },
    })
  )
}

export async function getCurrentTeamTransactions(): Promise<
  PageResult<TeamQuotaTransaction>
> {
  return unwrap(await api.get('/api/team/transactions?page=1&page_size=100'))
}

export async function fundCurrentTeam(
  amount: number,
  idempotencyKey: string
): Promise<Team> {
  return unwrap(
    await api.post('/api/team/fund', {
      amount,
      idempotency_key: idempotencyKey,
    })
  )
}

export async function calculateCurrentTeamStripeAmount(
  amount: number,
  checkoutMethod: StripeCheckoutMethod = 'standard'
): Promise<string> {
  return unwrap(
    await api.post(
      '/api/team/stripe/amount',
      createTeamStripeAmountPayload(amount, checkoutMethod),
      {
        skipBusinessError: true,
      } as Record<string, unknown>
    )
  )
}

export async function requestCurrentTeamStripePayment(
  request: TeamStripeCheckoutRequest
): Promise<TeamStripeCheckout> {
  return unwrap(
    await api.post(
      '/api/team/stripe/pay',
      createTeamStripeTopupPayload(request.amount, request.checkoutMethod),
      {
        skipBusinessError: true,
      } as Record<string, unknown>
    )
  )
}

export async function adminListTeams(): Promise<Team[]> {
  return unwrap(await api.get('/api/team/admin'))
}

export async function adminCreateTeam(input: CreateTeamInput): Promise<void> {
  unwrap(await api.post('/api/team/admin', input))
}

export async function adminGetTeamMembers(
  teamId: number
): Promise<TeamMember[]> {
  return unwrap(await api.get(`/api/team/admin/${teamId}/members`))
}

export async function adminCreateTeamMember(
  teamId: number,
  input: CreateTeamMemberInput
): Promise<{ user_id: number; username: string }> {
  return unwrap(await api.post(`/api/team/admin/${teamId}/members`, input))
}

export async function adminUpdateTeamMemberStatus(
  teamId: number,
  userId: number,
  status: number
): Promise<void> {
  unwrap(
    await api.patch(`/api/team/admin/${teamId}/members/${userId}/status`, {
      status,
    })
  )
}

export async function adminResetTeamMemberPassword(
  teamId: number,
  userId: number,
  password: string
): Promise<void> {
  unwrap(
    await api.post(
      `/api/team/admin/${teamId}/members/${userId}/reset-password`,
      { password }
    )
  )
}

export async function adminGetTeamUsage(
  teamId: number,
  startTimestamp?: number
): Promise<TeamUsage[]> {
  return unwrap(
    await api.get(`/api/team/admin/${teamId}/usage`, {
      params: { start_timestamp: startTimestamp },
    })
  )
}

export async function adminGetTeamTransactions(
  teamId: number
): Promise<PageResult<TeamQuotaTransaction>> {
  return unwrap(
    await api.get(`/api/team/admin/${teamId}/transactions?page=1&page_size=100`)
  )
}

export async function adminAdjustTeamQuota(
  teamId: number,
  operation: 'add' | 'subtract',
  amount: number,
  note: string,
  idempotencyKey: string
): Promise<Team> {
  return unwrap(
    await api.post(`/api/team/admin/${teamId}/quota`, {
      operation,
      amount,
      note,
      idempotency_key: idempotencyKey,
    })
  )
}

export async function adminUpdateTeamStatus(
  teamId: number,
  status: number
): Promise<void> {
  unwrap(await api.patch(`/api/team/admin/${teamId}/status`, { status }))
}

export async function adminListEnterpriseInquiries(): Promise<
  PageResult<EnterpriseInquiry>
> {
  return unwrap(await api.get('/api/enterprise/inquiries?page=1&page_size=100'))
}

export async function adminUpdateEnterpriseInquiryStatus(
  inquiryId: number,
  status: number
): Promise<void> {
  unwrap(
    await api.patch(`/api/enterprise/inquiries/${inquiryId}/status`, {
      status,
    })
  )
}

export async function createEnterpriseInquiry(
  input: EnterpriseInquiryInput
): Promise<{ inquiry_id: number }> {
  return unwrap(await api.post('/api/enterprise/inquiries', input))
}
