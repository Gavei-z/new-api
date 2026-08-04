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
export type PublicLegalSection = Readonly<{
  id: string
  titleKey: string
  paragraphKeys: readonly string[]
  bulletKeys?: readonly string[]
}>

export type PublicLegalRoute = '/terms' | '/privacy' | '/refund-policy'

export type PublicLegalLink = Readonly<{
  labelKey: string
  href: PublicLegalRoute
}>

export const TERMS_SECTIONS = [
  {
    id: 'service',
    titleKey: 'legal.terms.service.title',
    paragraphKeys: ['legal.terms.service.p1', 'legal.terms.service.p2'],
  },
  {
    id: 'eligibility',
    titleKey: 'legal.terms.eligibility.title',
    paragraphKeys: ['legal.terms.eligibility.p1', 'legal.terms.eligibility.p2'],
  },
  {
    id: 'acceptable-use',
    titleKey: 'legal.terms.acceptableUse.title',
    paragraphKeys: ['legal.terms.acceptableUse.p1'],
    bulletKeys: [
      'legal.terms.acceptableUse.b1',
      'legal.terms.acceptableUse.b2',
      'legal.terms.acceptableUse.b3',
      'legal.terms.acceptableUse.b4',
    ],
  },
  {
    id: 'api-content',
    titleKey: 'legal.terms.apiContent.title',
    paragraphKeys: ['legal.terms.apiContent.p1', 'legal.terms.apiContent.p2'],
  },
  {
    id: 'pricing-and-billing',
    titleKey: 'legal.terms.billing.title',
    paragraphKeys: [
      'legal.terms.billing.p1',
      'legal.terms.billing.p2',
      'legal.terms.billing.p3',
    ],
  },
  {
    id: 'delivery-refunds-cancellation',
    titleKey: 'legal.terms.refunds.title',
    paragraphKeys: [
      'legal.terms.refunds.p1',
      'legal.terms.refunds.p2',
      'legal.terms.refunds.p3',
      'legal.terms.refunds.p4',
    ],
  },
  {
    id: 'availability',
    titleKey: 'legal.terms.availability.title',
    paragraphKeys: [
      'legal.terms.availability.p1',
      'legal.terms.availability.p2',
    ],
  },
  {
    id: 'suspension',
    titleKey: 'legal.terms.suspension.title',
    paragraphKeys: ['legal.terms.suspension.p1', 'legal.terms.suspension.p2'],
  },
  {
    id: 'intellectual-property',
    titleKey: 'legal.terms.ip.title',
    paragraphKeys: ['legal.terms.ip.p1', 'legal.terms.ip.p2'],
  },
  {
    id: 'disclaimers',
    titleKey: 'legal.terms.disclaimers.title',
    paragraphKeys: ['legal.terms.disclaimers.p1', 'legal.terms.disclaimers.p2'],
  },
  {
    id: 'liability',
    titleKey: 'legal.terms.liability.title',
    paragraphKeys: ['legal.terms.liability.p1'],
  },
  {
    id: 'changes',
    titleKey: 'legal.terms.changes.title',
    paragraphKeys: ['legal.terms.changes.p1'],
  },
  {
    id: 'governing-terms',
    titleKey: 'legal.terms.governing.title',
    paragraphKeys: ['legal.terms.governing.p1'],
  },
] as const satisfies readonly PublicLegalSection[]

export const PRIVACY_SECTIONS = [
  {
    id: 'scope',
    titleKey: 'legal.privacy.scope.title',
    paragraphKeys: ['legal.privacy.scope.p1'],
  },
  {
    id: 'information-collected',
    titleKey: 'legal.privacy.collection.title',
    paragraphKeys: ['legal.privacy.collection.p1'],
    bulletKeys: [
      'legal.privacy.collection.b1',
      'legal.privacy.collection.b2',
      'legal.privacy.collection.b3',
      'legal.privacy.collection.b4',
      'legal.privacy.collection.b5',
    ],
  },
  {
    id: 'api-content',
    titleKey: 'legal.privacy.apiContent.title',
    paragraphKeys: [
      'legal.privacy.apiContent.p1',
      'legal.privacy.apiContent.p2',
    ],
  },
  {
    id: 'use-of-information',
    titleKey: 'legal.privacy.use.title',
    paragraphKeys: ['legal.privacy.use.p1'],
    bulletKeys: [
      'legal.privacy.use.b1',
      'legal.privacy.use.b2',
      'legal.privacy.use.b3',
      'legal.privacy.use.b4',
    ],
  },
  {
    id: 'sharing',
    titleKey: 'legal.privacy.sharing.title',
    paragraphKeys: ['legal.privacy.sharing.p1'],
    bulletKeys: [
      'legal.privacy.sharing.b1',
      'legal.privacy.sharing.b2',
      'legal.privacy.sharing.b3',
      'legal.privacy.sharing.b4',
    ],
  },
  {
    id: 'payments',
    titleKey: 'legal.privacy.payments.title',
    paragraphKeys: ['legal.privacy.payments.p1', 'legal.privacy.payments.p2'],
  },
  {
    id: 'cookies',
    titleKey: 'legal.privacy.cookies.title',
    paragraphKeys: ['legal.privacy.cookies.p1'],
  },
  {
    id: 'retention',
    titleKey: 'legal.privacy.retention.title',
    paragraphKeys: ['legal.privacy.retention.p1'],
  },
  {
    id: 'security',
    titleKey: 'legal.privacy.security.title',
    paragraphKeys: ['legal.privacy.security.p1'],
  },
  {
    id: 'international-transfers',
    titleKey: 'legal.privacy.transfers.title',
    paragraphKeys: ['legal.privacy.transfers.p1'],
  },
  {
    id: 'your-rights',
    titleKey: 'legal.privacy.rights.title',
    paragraphKeys: ['legal.privacy.rights.p1', 'legal.privacy.rights.p2'],
  },
  {
    id: 'children',
    titleKey: 'legal.privacy.children.title',
    paragraphKeys: ['legal.privacy.children.p1'],
  },
  {
    id: 'changes',
    titleKey: 'legal.privacy.changes.title',
    paragraphKeys: ['legal.privacy.changes.p1'],
  },
] as const satisfies readonly PublicLegalSection[]

export const REFUND_SECTIONS = [
  {
    id: 'digital-delivery',
    titleKey: 'legal.refund.delivery.title',
    paragraphKeys: ['legal.refund.delivery.p1', 'legal.refund.delivery.p2'],
  },
  {
    id: 'eligible-refunds',
    titleKey: 'legal.refund.eligible.title',
    paragraphKeys: ['legal.refund.eligible.p1'],
    bulletKeys: [
      'legal.refund.eligible.b1',
      'legal.refund.eligible.b2',
      'legal.refund.eligible.b3',
    ],
  },
  {
    id: 'non-refundable',
    titleKey: 'legal.refund.nonRefundable.title',
    paragraphKeys: ['legal.refund.nonRefundable.p1'],
    bulletKeys: [
      'legal.refund.nonRefundable.b1',
      'legal.refund.nonRefundable.b2',
      'legal.refund.nonRefundable.b3',
      'legal.refund.nonRefundable.b4',
    ],
  },
  {
    id: 'request-process',
    titleKey: 'legal.refund.request.title',
    paragraphKeys: ['legal.refund.request.p1', 'legal.refund.request.p2'],
  },
  {
    id: 'review-and-timing',
    titleKey: 'legal.refund.timing.title',
    paragraphKeys: ['legal.refund.timing.p1', 'legal.refund.timing.p2'],
  },
  {
    id: 'cancellation',
    titleKey: 'legal.refund.cancellation.title',
    paragraphKeys: [
      'legal.refund.cancellation.p1',
      'legal.refund.cancellation.p2',
    ],
  },
  {
    id: 'payment-disputes',
    titleKey: 'legal.refund.disputes.title',
    paragraphKeys: ['legal.refund.disputes.p1'],
  },
] as const satisfies readonly PublicLegalSection[]

export const TERMS_RELATED_LINKS = [
  { labelKey: 'Privacy Policy', href: '/privacy' },
  { labelKey: 'Refund Policy', href: '/refund-policy' },
] as const satisfies readonly PublicLegalLink[]

export const PRIVACY_RELATED_LINKS = [
  { labelKey: 'Terms of Service', href: '/terms' },
  { labelKey: 'Refund Policy', href: '/refund-policy' },
] as const satisfies readonly PublicLegalLink[]

export const REFUND_RELATED_LINKS = [
  { labelKey: 'Terms of Service', href: '/terms' },
  { labelKey: 'Privacy Policy', href: '/privacy' },
] as const satisfies readonly PublicLegalLink[]
