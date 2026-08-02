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
import { AlertCircle, Bot, Braces, MessageSquareCode } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  CLAUDE_CODE_EXAMPLE,
  CODEX_CONFIG_EXAMPLE,
  CODEX_KEY_EXAMPLE,
} from '../constants'
import { CodeSample } from './code-sample'

const ENDPOINTS = [
  {
    method: 'POST',
    path: '/v1/responses',
    titleKey: 'docs.reference.responses.title',
    descriptionKey: 'docs.reference.responses.description',
    icon: Bot,
  },
  {
    method: 'POST',
    path: '/v1/chat/completions',
    titleKey: 'docs.reference.chat.title',
    descriptionKey: 'docs.reference.chat.description',
    icon: MessageSquareCode,
  },
  {
    method: 'POST',
    path: '/v1/messages',
    titleKey: 'docs.reference.messages.title',
    descriptionKey: 'docs.reference.messages.description',
    icon: Braces,
  },
] as const

const ERRORS = [
  {
    code: '401',
    titleKey: 'docs.troubleshooting.401.title',
    descriptionKey: 'docs.troubleshooting.401.description',
  },
  {
    code: '429',
    titleKey: 'docs.troubleshooting.429.title',
    descriptionKey: 'docs.troubleshooting.429.description',
  },
  {
    code: '5xx',
    titleKey: 'docs.troubleshooting.5xx.title',
    descriptionKey: 'docs.troubleshooting.5xx.description',
  },
] as const

export function CodingAgentsSection() {
  const { t } = useTranslation()

  return (
    <section id='coding-agents' className='scroll-mt-24'>
      <h2 className='text-2xl font-semibold tracking-tight'>
        {t('docs.agents.title')}
      </h2>
      <p className='text-muted-foreground mt-3 leading-7'>
        {t('docs.agents.description')}
      </p>

      <div className='mt-6 space-y-7'>
        <div>
          <div className='flex items-start gap-3'>
            <span className='bg-primary/10 text-primary mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg'>
              <Bot aria-hidden='true' className='size-4' />
            </span>
            <div>
              <h3 className='font-semibold'>{t('docs.agents.codex.title')}</h3>
              <p className='text-muted-foreground mt-1 text-sm leading-6'>
                {t('docs.agents.codex.description')}
              </p>
            </div>
          </div>
          <div className='mt-4 grid gap-4'>
            <CodeSample
              code={CODEX_KEY_EXAMPLE}
              label={t('docs.agents.codex.environment')}
              language='Shell'
            />
            <CodeSample
              code={CODEX_CONFIG_EXAMPLE}
              label='~/.codex/config.toml'
              language='TOML'
            />
          </div>
        </div>

        <div className='border-border/60 border-t pt-7'>
          <div className='flex items-start gap-3'>
            <span className='bg-primary/10 text-primary mt-0.5 flex size-9 shrink-0 items-center justify-center rounded-lg'>
              <MessageSquareCode aria-hidden='true' className='size-4' />
            </span>
            <div>
              <h3 className='font-semibold'>
                {t('docs.agents.claudeCode.title')}
              </h3>
              <p className='text-muted-foreground mt-1 text-sm leading-6'>
                {t('docs.agents.claudeCode.description')}
              </p>
            </div>
          </div>
          <CodeSample
            code={CLAUDE_CODE_EXAMPLE}
            label={t('docs.agents.claudeCode.environment')}
            language='Shell'
            className='mt-4'
          />
          <p className='text-muted-foreground mt-3 text-xs leading-5'>
            {t('docs.agents.claudeCode.baseUrlNote')}
          </p>
        </div>
      </div>
    </section>
  )
}

export function ApiReferenceSection() {
  const { t } = useTranslation()

  return (
    <section id='api-reference' className='scroll-mt-24'>
      <h2 className='text-2xl font-semibold tracking-tight'>
        {t('docs.reference.title')}
      </h2>
      <p className='text-muted-foreground mt-3 leading-7'>
        {t('docs.reference.description')}
      </p>

      <div className='mt-5 space-y-3'>
        {ENDPOINTS.map((endpoint) => {
          const Icon = endpoint.icon
          return (
            <div
              key={endpoint.path}
              className='border-border/60 bg-card/60 flex flex-col gap-3 rounded-xl border p-4 sm:flex-row sm:items-start'
            >
              <span className='bg-primary/10 text-primary flex size-9 shrink-0 items-center justify-center rounded-lg'>
                <Icon aria-hidden='true' className='size-4' />
              </span>
              <div className='min-w-0 flex-1'>
                <div className='flex flex-wrap items-center gap-2'>
                  <span className='bg-emerald-500/10 text-[10px] font-bold tracking-wider text-emerald-700 uppercase dark:text-emerald-400'>
                    {endpoint.method}
                  </span>
                  <code className='text-sm font-semibold break-all'>
                    {endpoint.path}
                  </code>
                </div>
                <h3 className='mt-2 text-sm font-semibold'>
                  {t(endpoint.titleKey)}
                </h3>
                <p className='text-muted-foreground mt-1 text-sm leading-6'>
                  {t(endpoint.descriptionKey)}
                </p>
              </div>
            </div>
          )
        })}
      </div>
    </section>
  )
}

export function TroubleshootingSection() {
  const { t } = useTranslation()

  return (
    <section id='troubleshooting' className='scroll-mt-24'>
      <h2 className='text-2xl font-semibold tracking-tight'>
        {t('docs.troubleshooting.title')}
      </h2>
      <p className='text-muted-foreground mt-3 leading-7'>
        {t('docs.troubleshooting.description')}
      </p>

      <div className='mt-5 grid gap-3'>
        {ERRORS.map((error) => (
          <div
            key={error.code}
            className='border-border/60 bg-card/60 flex gap-3 rounded-xl border p-4'
          >
            <span className='bg-destructive/10 text-destructive flex h-8 min-w-12 items-center justify-center rounded-lg px-2 font-mono text-xs font-semibold'>
              {error.code}
            </span>
            <div>
              <h3 className='text-sm font-semibold'>{t(error.titleKey)}</h3>
              <p className='text-muted-foreground mt-1 text-sm leading-6'>
                {t(error.descriptionKey)}
              </p>
            </div>
          </div>
        ))}
      </div>

      <div className='border-border/60 bg-muted/25 mt-5 flex gap-3 rounded-xl border p-4'>
        <AlertCircle
          aria-hidden='true'
          className='text-primary mt-0.5 size-4 shrink-0'
        />
        <p className='text-muted-foreground text-sm leading-6'>
          <strong className='text-foreground'>
            {t('docs.troubleshooting.supportTitle')}
          </strong>{' '}
          {t('docs.troubleshooting.supportDescription')}
        </p>
      </div>
    </section>
  )
}
