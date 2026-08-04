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
import { UNIROUTERS_SUPPORT_EMAIL } from '@/lib/constants'

export interface FooterLink {
  text: string
  href: string
  hash?: string
}

export interface FooterColumnProps {
  title: string
  links: readonly FooterLink[]
}

export const PUBLIC_FOOTER_COLUMNS: readonly FooterColumnProps[] = [
  {
    title: 'footer.columns.product.title',
    links: [
      {
        text: 'footer.columns.product.links.pricing',
        href: '/pricing',
      },
      {
        text: 'footer.columns.product.links.imageGeneration',
        href: '/pricing/gpt-image-2',
      },
      {
        text: 'footer.columns.product.links.enterprise',
        href: '/enterprise',
      },
    ],
  },
  {
    title: 'footer.columns.resources.title',
    links: [
      {
        text: 'footer.columns.resources.links.docs',
        href: '/docs',
      },
      {
        text: 'footer.columns.resources.links.quickStart',
        href: '/docs',
        hash: 'quick-start',
      },
      {
        text: 'footer.columns.resources.links.apiReference',
        href: '/docs',
        hash: 'api-reference',
      },
      {
        text: 'footer.columns.resources.links.faq',
        href: '/',
        hash: 'faqs',
      },
    ],
  },
  {
    title: 'footer.columns.support.title',
    links: [
      {
        text: 'footer.columns.support.links.contact',
        href: `mailto:${UNIROUTERS_SUPPORT_EMAIL}?subject=UniRouters%20Support`,
      },
      {
        text: 'footer.columns.support.links.invoice',
        href: `mailto:${UNIROUTERS_SUPPORT_EMAIL}?subject=UniRouters%20Invoice%20Request`,
      },
      {
        text: 'footer.columns.support.links.enterpriseQuote',
        href: '/enterprise',
        hash: 'enterprise-quote',
      },
    ],
  },
  {
    title: 'footer.columns.legal.title',
    links: [
      {
        text: 'Terms of Service',
        href: '/terms',
      },
      {
        text: 'Privacy Policy',
        href: '/privacy',
      },
      {
        text: 'Refund Policy',
        href: '/refund-policy',
      },
    ],
  },
]
