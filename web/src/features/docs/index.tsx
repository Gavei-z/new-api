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
import { ArrowRight, Braces, KeyRound, Network, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'

import {
  ApiReferenceSection,
  CodingAgentsSection,
  TroubleshootingSection,
} from './components/coding-reference-sections'
import {
  DesktopDocsNavigation,
  MobileDocsNavigation,
  OnThisPageNavigation,
} from './components/docs-navigation'
import {
  AuthenticationSection,
  OpenAISdksSection,
  QuickStartSection,
} from './components/quick-start-sections'
import {
  UNIROUTERS_API_BASE_URL,
  UNIROUTERS_API_KEY_PLACEHOLDER,
} from './constants'

const CAPABILITIES = [
  {
    icon: Network,
    titleKey: 'docs.overview.capabilityEndpoint.title',
    descriptionKey: 'docs.overview.capabilityEndpoint.description',
  },
  {
    icon: KeyRound,
    titleKey: 'docs.overview.capabilityKey.title',
    descriptionKey: 'docs.overview.capabilityKey.description',
  },
  {
    icon: Braces,
    titleKey: 'docs.overview.capabilityProtocol.title',
    descriptionKey: 'docs.overview.capabilityProtocol.description',
  },
] as const

function DocsHero() {
  const { t } = useTranslation()

  return (
    <header id='overview' className='scroll-mt-24 pt-8 sm:pt-12'>
      <div className='border-primary/20 bg-primary/5 text-primary inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs font-medium'>
        <span className='bg-primary size-1.5 rounded-full' />
        {t('docs.hero.eyebrow')}
      </div>
      <h1 className='mt-5 max-w-3xl text-[clamp(2.25rem,6vw,4rem)] leading-[1.08] font-semibold tracking-[-0.035em]'>
        {t('docs.hero.title')}
      </h1>
      <p className='text-muted-foreground mt-5 max-w-2xl text-base leading-7 sm:text-lg sm:leading-8'>
        {t('docs.hero.description')}
      </p>
      <a
        href='#quick-start'
        className='bg-primary text-primary-foreground hover:bg-primary/90 mt-7 inline-flex h-10 items-center gap-2 rounded-lg px-4 text-sm font-medium transition-colors'
      >
        {t('docs.hero.start')}
        <ArrowRight aria-hidden='true' className='size-4' />
      </a>

      <div className='mt-9 grid gap-3 sm:grid-cols-3'>
        {CAPABILITIES.map((capability) => {
          const Icon = capability.icon
          return (
            <div
              key={capability.titleKey}
              className='border-border/60 bg-card/60 rounded-xl border p-4 backdrop-blur-sm'
            >
              <span className='bg-primary/10 text-primary flex size-9 items-center justify-center rounded-lg'>
                <Icon aria-hidden='true' className='size-4' />
              </span>
              <h2 className='mt-3 text-sm font-semibold'>
                {t(capability.titleKey)}
              </h2>
              <p className='text-muted-foreground mt-1 text-xs leading-5'>
                {t(capability.descriptionKey)}
              </p>
            </div>
          )
        })}
      </div>
    </header>
  )
}

function DocsFooter() {
  const { t } = useTranslation()

  return (
    <footer className='border-border/60 mt-20 border-t py-8'>
      <div className='flex flex-col justify-between gap-3 text-xs sm:flex-row sm:items-center'>
        <div>
          <span className='font-semibold'>uniRouters</span>
          <span className='text-muted-foreground'>
            {' '}
            · {t('docs.footer.tagline')}
          </span>
        </div>
        <p className='text-muted-foreground'>
          {t('docs.footer.builtOn')}{' '}
          <a
            href='https://github.com/QuantumNous/new-api'
            target='_blank'
            rel='noopener noreferrer'
            className='text-foreground hover:underline'
          >
            New API
          </a>
        </p>
      </div>
    </footer>
  )
}

export function Docs() {
  const { t } = useTranslation()

  return (
    <>
      <title>{t('docs.meta.title')}</title>
      <meta name='description' content={t('docs.meta.description')} />
      <PublicLayout
        showMainContainer={false}
        siteName='uniRouters'
        logo={
          <span className='flex size-8 items-center justify-center rounded-lg bg-gradient-to-br from-blue-500 via-violet-500 to-cyan-500 text-white shadow-lg shadow-blue-500/25'>
            <Zap aria-hidden='true' className='size-4 fill-current' />
          </span>
        }
      >
        <div className='relative overflow-hidden pt-16'>
          <div
            aria-hidden='true'
            className='pointer-events-none absolute inset-x-0 top-0 -z-10 h-[520px] opacity-30 dark:opacity-15'
            style={{
              background:
                'radial-gradient(ellipse 65% 55% at 50% 5%, oklch(0.68 0.18 255 / 55%) 0%, transparent 72%)',
            }}
          />

          <div className='mx-auto max-w-7xl px-4 sm:px-6 lg:px-8'>
            <DocsHero />

            <div className='mt-12 border-t pt-6 sm:mt-16 sm:pt-8'>
              <MobileDocsNavigation />
              <div className='grid gap-10 lg:grid-cols-[210px_minmax(0,1fr)] xl:grid-cols-[210px_minmax(0,760px)_170px] xl:justify-between'>
                <DesktopDocsNavigation />

                <main
                  className='min-w-0 space-y-16 pb-4'
                  aria-label={t('docs.content.label')}
                >
                  <QuickStartSection />
                  <AuthenticationSection />
                  <OpenAISdksSection />
                  <CodingAgentsSection />
                  <ApiReferenceSection />
                  <TroubleshootingSection />
                </main>

                <OnThisPageNavigation />
              </div>
            </div>

            <div className='border-border/60 bg-muted/25 mt-12 grid gap-4 rounded-2xl border p-5 sm:grid-cols-2 sm:p-6'>
              <div>
                <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                  {t('docs.summary.baseUrl')}
                </p>
                <code className='mt-2 block text-sm font-semibold break-all'>
                  {UNIROUTERS_API_BASE_URL}
                </code>
              </div>
              <div>
                <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
                  {t('docs.summary.keyFormat')}
                </p>
                <code className='mt-2 block text-sm font-semibold break-all'>
                  {UNIROUTERS_API_KEY_PLACEHOLDER}
                </code>
              </div>
            </div>

            <DocsFooter />
          </div>
        </div>
      </PublicLayout>
    </>
  )
}
