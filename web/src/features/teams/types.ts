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
export type Team = {
  id: number
  name: string
  slug: string
  status: number
  quota: number
  total_quota?: number
  reserve_quota?: number
  total_available_quota?: number
  used_quota: number
  created_by: number
  created_at: number
  updated_at: number
}

export type TeamSummary = {
  team: Team
  member_count: number
  role: number
}

export type TeamMember = {
  user_id: number
  username: string
  display_name: string
  email: string
  user_status: number
  role: number
  member_status: number
  used_quota: number
  request_count: number
  created_at: number
}

export type TeamUsage = {
  user_id: number
  username: string
  model_name: string
  request_count: number
  quota: number
  prompt_tokens: number
  completion_tokens: number
}

export type TeamQuotaTransaction = {
  id: number | string
  source?: string
  team_id: number
  user_id: number
  actor_user_id: number
  type: string
  quota_delta: number
  active_quota_delta?: number
  reserve_quota_delta?: number
  used_quota_delta: number
  balance_after: number
  active_balance_after?: number
  reserve_balance_after?: number
  used_quota_after: number
  note: string
  created_at: number
}

export type EnterpriseInquiry = {
  id: number
  company_name: string
  contact_name: string
  email: string
  phone: string
  wechat: string
  team_size: string
  expected_monthly_usage: string
  message: string
  status: number
  created_at: number
}

export type PageResult<T> = {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type CreateTeamInput = {
  name: string
  slug: string
  owner: {
    username: string
    password: string
    display_name: string
    email: string
  }
}

export type CreateTeamMemberInput = {
  username: string
  password: string
  display_name: string
  email: string
}

export type EnterpriseInquiryInput = {
  company_name: string
  contact_name: string
  email: string
  phone: string
  wechat: string
  team_size: string
  message: string
  privacy_accepted: boolean
  website: string
}

export type TeamStripeTopupPayload = {
  amount: number
}

export type TeamStripeCheckout = {
  pay_link: string
}
