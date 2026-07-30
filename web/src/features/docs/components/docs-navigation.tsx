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
import { ChevronDown } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { DOCS_SECTIONS, UNIROUTERS_API_BASE_URL } from '../constants'

function DocsNavigationLinks() {
  const { t } = useTranslation()

  return (
    <ul className='space-y-1'>
      {DOCS_SECTIONS.map((section) => (
        <li key={section.id}>
          <a
            href={`#${section.id}`}
            className='text-muted-foreground hover:bg-muted/60 hover:text-foreground block rounded-lg px-3 py-2 text-sm transition-colors'
          >
            {t(section.labelKey)}
          </a>
        </li>
      ))}
    </ul>
  )
}

export function DesktopDocsNavigation() {
  const { t } = useTranslation()

  return (
    <aside className='sticky top-24 hidden max-h-[calc(100svh-7rem)] self-start overflow-y-auto lg:block'>
      <nav aria-label={t('docs.navigation.label')}>
        <p className='text-foreground mb-3 px-3 text-xs font-semibold tracking-wider uppercase'>
          {t('docs.navigation.title')}
        </p>
        <DocsNavigationLinks />
      </nav>
      <div className='border-border/60 bg-muted/25 mt-6 rounded-xl border p-3.5'>
        <p className='text-muted-foreground text-[11px] font-medium tracking-wider uppercase'>
          {t('docs.navigation.baseUrl')}
        </p>
        <code className='mt-2 block text-xs font-medium break-all'>
          {UNIROUTERS_API_BASE_URL}
        </code>
      </div>
    </aside>
  )
}

export function MobileDocsNavigation() {
  const { t } = useTranslation()

  return (
    <details className='border-border/60 bg-card/70 group mb-6 rounded-xl border lg:hidden'>
      <summary className='flex cursor-pointer list-none items-center justify-between px-4 py-3 text-sm font-medium'>
        {t('docs.navigation.mobileTitle')}
        <ChevronDown
          aria-hidden='true'
          className='text-muted-foreground size-4 transition-transform group-open:rotate-180'
        />
      </summary>
      <nav
        aria-label={t('docs.navigation.label')}
        className='border-border/50 border-t p-2'
      >
        <DocsNavigationLinks />
      </nav>
    </details>
  )
}

export function OnThisPageNavigation() {
  const { t } = useTranslation()

  return (
    <aside className='sticky top-24 hidden self-start xl:block'>
      <nav aria-label={t('docs.navigation.onThisPage')}>
        <p className='text-foreground mb-3 text-xs font-semibold tracking-wider uppercase'>
          {t('docs.navigation.onThisPage')}
        </p>
        <ul className='border-border/60 space-y-2 border-l pl-3'>
          {DOCS_SECTIONS.slice(1).map((section) => (
            <li key={section.id}>
              <a
                href={`#${section.id}`}
                className='text-muted-foreground hover:text-foreground block text-xs leading-5 transition-colors'
              >
                {t(section.labelKey)}
              </a>
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  )
}
