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
/**
 * Home page constants
 * All hardcoded data for home page sections
 */
import type { TFunction } from 'i18next'

import { UNIROUTERS_DOCS_PATH } from '@/features/docs/constants'

// Layout - Main base classes
export const MAIN_BASE_CLASSES = 'bg-background text-foreground w-full'

export const UNIROUTERS_HOME_DOCS_PATH = UNIROUTERS_DOCS_PATH
export const UNIROUTERS_HOME_NAV_LINKS = [
  { title: 'Models', href: '/models', requiresAuth: true },
  { title: 'Pricing', href: '/pricing' },
  { title: 'Docs', href: UNIROUTERS_HOME_DOCS_PATH },
]

export const UNIROUTERS_HOME_HERO_COPY = {
  taglineKey: 'One integration for every leading AI model',
  headlineKey: 'One Unified Gateway for Multi Models.',
} as const

export const UNIROUTERS_HOME_FAQS = [
  {
    questionKey: 'How do I use UniRouters?',
    answerKey:
      'After installing Node.js and the official Claude Code or Codex CLI, simply configure your API key and API base URL in the terminal. You can also call UniRouters directly from Python, JavaScript, or curl.',
  },
  {
    questionKey: 'Why is it so cheap?',
    answerKey:
      'We buy at wholesale rates and operate at scale, keeping costs low without hidden tricks.',
  },
  {
    questionKey: 'Does UniRouters support cache reads?',
    answerKey:
      'Yes. Cache reads are supported. When usage is calculated, input tokens served from cache are multiplied by a discount factor, so they consume less credit.',
  },
  {
    questionKey: 'How can I request an invoice?',
    answerKey:
      'If your cumulative personal spending exceeds USD 40, or your enterprise service spending exceeds USD 70, we can issue an invoice. Standard invoices are available, and bank transfers are supported.',
  },
  {
    questionKey: 'What if I do not use all of my credit?',
    answerKey:
      'You can share API access with friends or team members by creating separate API keys under one account. UniRouters supports multiple API keys per account.',
  },
] as const

// Hero section - AI Applications (Left side)
export const AI_APPLICATIONS = [
  'LobeHub.Color',
  'Dify.Color',
  'OpenWebUI',
  'Cline',
] as const

// Hero section - AI Models (Right side)
export const AI_MODELS = [
  'Qwen.Color',
  'DeepSeek.Color',
  'Doubao.Color',
  'OpenAI',
  'Claude.Color',
  'Gemini.Color',
] as const

// Hero section - Gateway Features
export const GATEWAY_FEATURES = [
  'Cost Tracking',
  'Model Access',
  'Guardrails',
  'Observability',
  'Budgets',
  'Load Balancing',
  'Rate Limiting',
  'Token Mgmt',
  'Prompt Caching',
  'Pass-Through',
] as const

// Stats section - Default statistics
export const DEFAULT_STATS = [
  {
    value: '50',
    suffix: '+',
    description: 'upstream services integrated',
  },
  {
    value: '100',
    suffix: '+',
    description: 'model billing support',
  },
  {
    value: '50',
    suffix: '+',
    description: 'compatible API routes',
  },
  {
    value: '10',
    suffix: '+',
    description: 'scheduling controls',
  },
] as const

// Features section - Default features
export const DEFAULT_FEATURES = [
  {
    title: 'Lightning Fast',
    description:
      'Optimized network architecture ensures millisecond response times',
    iconName: 'Zap',
  },
  {
    title: 'Secure & Reliable',
    description:
      'Enterprise-grade security with comprehensive permission management',
    iconName: 'Shield',
  },
  {
    title: 'Global Coverage',
    description: 'Multi-region deployment for stable global access',
    iconName: 'Globe',
  },
  {
    title: 'Developer Friendly',
    description: 'Compatible API routes for common AI application workflows',
    iconName: 'Code',
  },
  {
    title: 'High Performance',
    description: 'Support for high concurrency with automatic load balancing',
    iconName: 'Gauge',
  },
  {
    title: 'Transparent Billing',
    description: 'Pay-as-you-go with real-time usage monitoring',
    iconName: 'DollarSign',
  },
  {
    title: 'Team Collaboration',
    description: 'Multi-user management with flexible permission allocation',
    iconName: 'Users',
  },
  {
    title: 'Open Source',
    description: 'Community driven, self-hosted, and extensible',
    iconName: 'HeartHandshake',
  },
] as const

export function getGatewayFeatures(t: TFunction) {
  return GATEWAY_FEATURES.map((feature) => t(feature))
}

export function getDefaultStats(t: TFunction) {
  return DEFAULT_STATS.map((stat) => ({
    ...stat,
    description: stat.description ? t(stat.description) : undefined,
  }))
}

export function getDefaultFeatures(t: TFunction) {
  return DEFAULT_FEATURES.map((feature) => ({
    ...feature,
    title: t(feature.title),
    description: t(feature.description),
  }))
}
