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
import { KeyRound, Route, Send } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  PYTHON_SDK_EXAMPLE,
  RESPONSE_CURL_EXAMPLE,
  TYPESCRIPT_SDK_EXAMPLE,
  UNIROUTERS_API_BASE_URL,
} from '../constants'
import { CodeSample } from './code-sample'

const QUICK_START_STEPS = [
  {
    icon: KeyRound,
    titleKey: 'docs.quickStart.stepKey.title',
    descriptionKey: 'docs.quickStart.stepKey.description',
  },
  {
    icon: Route,
    titleKey: 'docs.quickStart.stepBaseUrl.title',
    descriptionKey: 'docs.quickStart.stepBaseUrl.description',
  },
  {
    icon: Send,
    titleKey: 'docs.quickStart.stepRequest.title',
    descriptionKey: 'docs.quickStart.stepRequest.description',
  },
] as const

export function QuickStartSection() {
  const { t } = useTranslation()

  return (
    <section id='quick-start' className='scroll-mt-24'>
      <p className='text-primary text-xs font-semibold tracking-wider uppercase'>
        {t('docs.quickStart.eyebrow')}
      </p>
      <h2 className='mt-2 text-2xl font-semibold tracking-tight'>
        {t('docs.quickStart.title')}
      </h2>
      <p className='text-muted-foreground mt-3 leading-7'>
        {t('docs.quickStart.description')}
      </p>

      <ol className='mt-6 grid gap-3 md:grid-cols-3'>
        {QUICK_START_STEPS.map((step, index) => {
          const Icon = step.icon
          return (
            <li
              key={step.titleKey}
              className='border-border/60 bg-card/60 rounded-xl border p-4'
            >
              <div className='flex items-center justify-between'>
                <span className='bg-primary/10 text-primary flex size-9 items-center justify-center rounded-lg'>
                  <Icon aria-hidden='true' className='size-4' />
                </span>
                <span className='text-muted-foreground/50 text-xs font-semibold'>
                  0{index + 1}
                </span>
              </div>
              <h3 className='mt-4 text-sm font-semibold'>{t(step.titleKey)}</h3>
              <p className='text-muted-foreground mt-1.5 text-sm leading-6'>
                {t(step.descriptionKey)}
              </p>
            </li>
          )
        })}
      </ol>

      <CodeSample
        code={RESPONSE_CURL_EXAMPLE}
        label={t('docs.quickStart.firstRequest')}
        language='Shell'
        className='mt-5'
      />
    </section>
  )
}

export function AuthenticationSection() {
  const { t } = useTranslation()

  return (
    <section id='authentication' className='scroll-mt-24'>
      <h2 className='text-2xl font-semibold tracking-tight'>
        {t('docs.authentication.title')}
      </h2>
      <p className='text-muted-foreground mt-3 leading-7'>
        {t('docs.authentication.description')}
      </p>

      <div className='mt-5 grid gap-3 sm:grid-cols-2'>
        <div className='border-border/60 bg-card/60 rounded-xl border p-4'>
          <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('docs.authentication.baseUrl')}
          </p>
          <code className='mt-2 block text-sm font-semibold break-all'>
            {UNIROUTERS_API_BASE_URL}
          </code>
        </div>
        <div className='border-border/60 bg-card/60 rounded-xl border p-4'>
          <p className='text-muted-foreground text-xs font-medium tracking-wider uppercase'>
            {t('docs.authentication.header')}
          </p>
          <code className='mt-2 block text-sm font-semibold break-all'>
            Authorization: Bearer $UNIROUTERS_API_KEY
          </code>
        </div>
      </div>

      <div className='border-primary/20 bg-primary/5 mt-4 rounded-xl border p-4 text-sm leading-6'>
        <strong>{t('docs.authentication.securityTitle')}</strong>{' '}
        <span className='text-muted-foreground'>
          {t('docs.authentication.securityDescription')}
        </span>
      </div>
    </section>
  )
}

export function OpenAISdksSection() {
  const { t } = useTranslation()

  return (
    <section id='openai-sdks' className='scroll-mt-24'>
      <h2 className='text-2xl font-semibold tracking-tight'>
        {t('docs.sdks.title')}
      </h2>
      <p className='text-muted-foreground mt-3 leading-7'>
        {t('docs.sdks.description')}
      </p>
      <div className='mt-5 space-y-4'>
        <CodeSample
          code={PYTHON_SDK_EXAMPLE}
          label={t('docs.sdks.python')}
          language='Python'
        />
        <CodeSample
          code={TYPESCRIPT_SDK_EXAMPLE}
          label={t('docs.sdks.typescript')}
          language='TypeScript'
        />
      </div>
    </section>
  )
}
