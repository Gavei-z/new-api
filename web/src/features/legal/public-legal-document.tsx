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
import { FileText, Mail, ShieldCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Footer } from '@/components/layout/components/footer'
import { PublicLayout } from '@/components/layout/components/public-layout'
import { UNIROUTERS_HOME_NAV_LINKS } from '@/features/home/constants'
import {
  DEFAULT_LOGO,
  DEFAULT_SYSTEM_NAME,
  UNIROUTERS_SUPPORT_EMAIL,
} from '@/lib/constants'

import type {
  PublicLegalLink,
  PublicLegalSection,
} from './public-legal-content'

type PublicLegalDocumentProps = {
  titleKey: string
  descriptionKey: string
  contactKey: string
  sections: readonly PublicLegalSection[]
  relatedLinks: readonly PublicLegalLink[]
}

export function PublicLegalDocument(props: PublicLegalDocumentProps) {
  const { t } = useTranslation()
  const title = t(props.titleKey)
  const description = t(props.descriptionKey)

  return (
    <>
      <title>{`${title} | UniRouters`}</title>
      <meta name='description' content={description} />
      <PublicLayout
        showMainContainer={false}
        showThemeSwitch
        siteName='uniRouters'
        logo={
          <img
            src={DEFAULT_LOGO}
            alt={DEFAULT_SYSTEM_NAME}
            className='size-8 rounded-lg object-cover'
          />
        }
        navLinks={UNIROUTERS_HOME_NAV_LINKS}
      >
        <main className='pt-16'>
          <header className='border-border/60 bg-muted/25 border-b'>
            <div className='mx-auto max-w-6xl px-4 py-16 sm:px-6 sm:py-20'>
              <div className='text-primary flex items-center gap-2 text-sm font-semibold tracking-wider uppercase'>
                <ShieldCheck aria-hidden='true' className='size-4' />
                {t('Legal')}
              </div>
              <h1 className='mt-4 max-w-4xl text-4xl font-bold tracking-tight text-balance sm:text-5xl'>
                {title}
              </h1>
              <p className='text-muted-foreground mt-5 max-w-3xl text-base leading-7 sm:text-lg'>
                {description}
              </p>
              <p className='text-muted-foreground mt-5 text-sm'>
                {t('legal.lastUpdated')}
              </p>
            </div>
          </header>

          <div className='mx-auto grid max-w-6xl gap-10 px-4 py-12 sm:px-6 lg:grid-cols-[220px_minmax(0,1fr)] lg:gap-16 lg:py-16'>
            <aside>
              <nav
                aria-label={t('On this page')}
                className='border-border/70 bg-card/60 rounded-2xl border p-5 lg:sticky lg:top-24'
              >
                <p className='text-sm font-semibold'>{t('On this page')}</p>
                <ul className='mt-4 space-y-2.5'>
                  {props.sections.map((section) => (
                    <li key={section.id}>
                      <a
                        href={`#${section.id}`}
                        className='text-muted-foreground hover:text-foreground block text-sm leading-5 transition-colors'
                      >
                        {t(section.titleKey)}
                      </a>
                    </li>
                  ))}
                </ul>
              </nav>
            </aside>

            <article className='min-w-0'>
              <div className='border-border/70 bg-card/50 rounded-2xl border px-5 py-2 shadow-sm sm:px-8'>
                {props.sections.map((section, index) => (
                  <section
                    key={section.id}
                    id={section.id}
                    className='border-border/60 scroll-mt-28 border-b py-8 last:border-b-0'
                    aria-labelledby={`${section.id}-heading`}
                  >
                    <div className='flex items-start gap-3'>
                      <span className='bg-primary/10 text-primary mt-0.5 flex size-8 shrink-0 items-center justify-center rounded-lg text-xs font-semibold'>
                        {String(index + 1).padStart(2, '0')}
                      </span>
                      <div className='min-w-0'>
                        <h2
                          id={`${section.id}-heading`}
                          className='text-xl font-semibold tracking-tight'
                        >
                          {t(section.titleKey)}
                        </h2>
                        <div className='text-muted-foreground mt-4 space-y-4 text-sm leading-7 sm:text-base'>
                          {section.paragraphKeys.map((paragraphKey) => (
                            <p key={paragraphKey}>{t(paragraphKey)}</p>
                          ))}
                          {section.bulletKeys && (
                            <ul className='list-disc space-y-2 pl-5'>
                              {section.bulletKeys.map((bulletKey) => (
                                <li key={bulletKey}>{t(bulletKey)}</li>
                              ))}
                            </ul>
                          )}
                        </div>
                      </div>
                    </div>
                  </section>
                ))}
              </div>

              <section
                aria-labelledby='legal-contact-heading'
                className='border-primary/20 bg-primary/5 mt-8 rounded-2xl border p-6 sm:p-8'
              >
                <div className='flex flex-col gap-5 sm:flex-row sm:items-start'>
                  <span className='bg-primary text-primary-foreground flex size-10 shrink-0 items-center justify-center rounded-xl'>
                    <Mail aria-hidden='true' className='size-5' />
                  </span>
                  <div>
                    <h2
                      id='legal-contact-heading'
                      className='text-lg font-semibold'
                    >
                      {t('Contact us')}
                    </h2>
                    <p className='text-muted-foreground mt-2 text-sm leading-6'>
                      {t(props.contactKey)}
                    </p>
                    <a
                      href={`mailto:${UNIROUTERS_SUPPORT_EMAIL}`}
                      className='text-primary mt-3 inline-flex font-medium hover:underline'
                    >
                      {UNIROUTERS_SUPPORT_EMAIL}
                    </a>
                  </div>
                </div>
              </section>

              <nav
                aria-label={t('Related policies')}
                className='text-muted-foreground mt-8 flex flex-wrap items-center gap-3 text-sm'
              >
                <FileText aria-hidden='true' className='size-4' />
                <span>{t('Related policies')}:</span>
                {props.relatedLinks.map((link) => (
                  <Link
                    key={link.href}
                    to={link.href}
                    className='text-foreground font-medium hover:underline'
                  >
                    {t(link.labelKey)}
                  </Link>
                ))}
              </nav>
            </article>
          </div>
        </main>
        <Footer />
      </PublicLayout>
    </>
  )
}
