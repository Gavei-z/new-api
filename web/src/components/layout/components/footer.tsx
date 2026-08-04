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
import { Link } from '@tanstack/react-router'
import { Fragment } from 'react'
import { useTranslation } from 'react-i18next'

import { useSystemConfig } from '@/hooks/use-system-config'
import {
  DEFAULT_LOGO,
  DEFAULT_SYSTEM_NAME,
  UNIROUTERS_SUPPORT_EMAIL,
} from '@/lib/constants'
import { cn } from '@/lib/utils'

import {
  PUBLIC_FOOTER_COLUMNS,
  type FooterColumnProps,
  type FooterLink,
} from './footer-links'

interface FooterProps {
  logo?: string
  name?: string
  columns?: readonly FooterColumnProps[]
  copyright?: string
  className?: string
}

function FooterLinkItem(props: { link: FooterLink }) {
  const { t } = useTranslation()
  const isExternalUrl = props.link.href.startsWith('http')
  const usesAnchor = isExternalUrl || props.link.href.startsWith('mailto:')
  const label = t(props.link.text)

  if (usesAnchor) {
    return (
      <a
        href={props.link.href}
        target={isExternalUrl ? '_blank' : undefined}
        rel={isExternalUrl ? 'noopener noreferrer' : undefined}
        className='text-muted-foreground hover:text-foreground text-sm transition-colors duration-200'
      >
        {label}
      </a>
    )
  }

  return (
    <Link
      to={props.link.href}
      hash={props.link.hash}
      className='text-muted-foreground hover:text-foreground text-sm transition-colors duration-200'
    >
      {label}
    </Link>
  )
}

function LegalLinks() {
  const { t } = useTranslation()
  const items = [
    {
      key: 'terms',
      label: t('Terms of Service'),
      href: '/terms',
    },
    {
      key: 'privacy',
      label: t('Privacy Policy'),
      href: '/privacy',
    },
    {
      key: 'refund-policy',
      label: t('Refund Policy'),
      href: '/refund-policy',
    },
  ]

  return (
    <>
      {items.map((item, index) => (
        <Fragment key={item.key}>
          {index > 0 && (
            <span aria-hidden='true' className='text-muted-foreground/30'>
              ·
            </span>
          )}
          <Link
            to={item.href}
            className='hover:text-foreground transition-colors duration-200'
          >
            {item.label}
          </Link>
        </Fragment>
      ))}
    </>
  )
}

export function Footer(props: FooterProps) {
  const { t } = useTranslation()
  const { footerHtml } = useSystemConfig()

  const displayLogo = props.logo ?? DEFAULT_LOGO
  const displayName = props.name ?? DEFAULT_SYSTEM_NAME
  const currentYear = new Date().getFullYear()
  const displayColumns: readonly FooterColumnProps[] =
    props.columns ?? PUBLIC_FOOTER_COLUMNS

  if (footerHtml) {
    return (
      <footer
        className={cn(
          'border-border/40 relative z-10 border-t',
          props.className
        )}
      >
        <div className='mx-auto w-full max-w-6xl px-6 py-5'>
          <div className='bg-muted/20 border-border/50 flex flex-col items-center justify-between gap-4 rounded-2xl border px-4 py-4 backdrop-blur-sm sm:flex-row sm:px-5'>
            <div
              className='custom-footer text-muted-foreground min-w-0 text-center text-sm sm:text-left'
              dangerouslySetInnerHTML={{ __html: footerHtml }}
            />
            <div className='border-border/60 text-muted-foreground/60 flex w-full flex-wrap items-center justify-center gap-x-3 gap-y-1 border-t pt-4 text-xs sm:w-auto sm:justify-end sm:border-t-0 sm:border-l sm:pt-0 sm:pl-5'>
              <LegalLinks />
            </div>
          </div>
        </div>
      </footer>
    )
  }

  return (
    <footer
      className={cn('border-border/40 relative z-10 border-t', props.className)}
    >
      <div className='mx-auto max-w-7xl px-6 py-12 md:py-16'>
        <div className='grid gap-12 lg:grid-cols-[minmax(220px,1fr)_minmax(0,2.4fr)] lg:gap-16'>
          <div>
            <Link to='/' className='group flex items-center gap-2.5'>
              <img
                src={displayLogo}
                alt={displayName}
                className='size-8 rounded-lg object-contain'
              />
              <span className='text-sm font-semibold tracking-tight'>
                {displayName}
              </span>
            </Link>
            <p className='text-muted-foreground mt-4 max-w-[240px] text-sm leading-6'>
              {t('footer.brand.tagline')}
            </p>
            <a
              href={`mailto:${UNIROUTERS_SUPPORT_EMAIL}`}
              className='text-muted-foreground hover:text-foreground mt-4 inline-flex text-sm transition-colors'
            >
              {UNIROUTERS_SUPPORT_EMAIL}
            </a>
          </div>

          <nav
            aria-label={t('Footer navigation')}
            className='grid grid-cols-2 gap-x-8 gap-y-10 sm:grid-cols-4'
          >
            {displayColumns.map((column) => (
              <div key={column.title}>
                <p className='text-foreground mb-4 text-sm font-semibold'>
                  {t(column.title)}
                </p>
                <ul className='space-y-3'>
                  {column.links.map((link) => (
                    <li key={`${link.href}-${link.hash ?? ''}-${link.text}`}>
                      <FooterLinkItem link={link} />
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </nav>
        </div>

        <div className='border-border/40 text-muted-foreground mt-12 flex flex-col gap-4 border-t pt-6 text-xs sm:flex-row sm:items-center sm:justify-between'>
          <span>
            &copy; {currentYear} {displayName}.{' '}
            {props.copyright ?? t('footer.defaultCopyright')}
          </span>
          <div className='flex flex-wrap items-center gap-x-3 gap-y-1'>
            <LegalLinks />
          </div>
        </div>
      </div>
    </footer>
  )
}
