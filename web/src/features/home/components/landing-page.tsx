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
import type { TFunction } from 'i18next'
import {
  ArrowRight,
  BarChart3,
  Cable,
  Code2,
  Cpu,
  LockKeyhole,
  ShieldCheck,
  Sparkles,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import {
  UNIROUTERS_HOME_DOCS_PATH,
  UNIROUTERS_HOME_HERO_COPY,
} from '../constants'
import { FaqSection } from './faq-section'
import { Stats } from './sections/stats'

function getQuickStarts(t: TFunction) {
  return [
    {
      value: 'claude-code',
      label: 'Claude Code',
      code: `# 1. ${t('Install Claude Code')}
npm install -g @anthropic-ai/claude-code

# 2. ${t('Route requests through uniRouters')}
export ANTHROPIC_BASE_URL="https://unirouters.cc"
export ANTHROPIC_AUTH_TOKEN="sk-one-xxx"

# 3. ${t('Start Claude Code')}
claude`,
    },
    {
      value: 'codex',
      label: 'Codex',
      code: `# 1. ${t('Install Codex CLI')}
npm install -g @openai/codex

# 2. ${t('Route requests through uniRouters')}
export OPENAI_BASE_URL="https://unirouters.cc/v1"
export OPENAI_API_KEY="sk-one-xxx"

# 3. ${t('Start Codex')}
codex`,
    },
    {
      value: 'python',
      label: 'Python',
      code: `from openai import OpenAI

client = OpenAI(base_url="https://unirouters.cc/v1", api_key="sk-one-xxx")

resp = client.chat.completions.create(
    model="claude-sonnet-4-5",
    messages=[{"role": "user", "content": "hi"}],
)
print(resp.choices[0].message.content)`,
    },
    {
      value: 'node',
      label: 'Node',
      code: `import OpenAI from 'openai'

const client = new OpenAI({ apiKey: 'sk-one-xxx', baseURL: 'https://unirouters.cc/v1' })

const res = await client.chat.completions.create({
  model: 'gpt-5',
  messages: [{ role: 'user', content: 'Hi' }],
})
console.log(res.choices[0].message.content)`,
    },
    {
      value: 'curl',
      label: 'curl',
      code: `curl https://unirouters.cc/v1/chat/completions \\
  -H "Authorization: Bearer sk-one-xxx" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-sonnet-4-5",
    "messages": [{"role":"user","content":"hi"}],
    "stream": true
  }'`,
    },
  ] as const
}

const features = [
  {
    icon: Cable,
    title: 'Unified API',
    description:
      'Use one API key for every leading AI model—no separate provider integrations.',
  },
  {
    icon: ShieldCheck,
    title: 'High availability',
    description:
      'Automatic multi-region failover keeps your applications online when a provider has an incident.',
  },
  {
    icon: Zap,
    title: 'Pay as you go',
    description:
      'Pay only for the tokens you use, with transparent, real-time spending details.',
  },
  {
    icon: LockKeyhole,
    title: 'Enterprise security',
    description:
      'Encrypted transport, granular API key permissions, IP allowlists, and usage alerts.',
  },
  {
    icon: Cpu,
    title: 'Multi-model routing',
    description:
      'Route each task to the best model and balance traffic automatically for lower cost and latency.',
  },
  {
    icon: BarChart3,
    title: 'Real-time monitoring',
    description:
      'Track usage, latency, and cost from a visual dashboard with exportable reports.',
  },
] as const

const models = [
  {
    provider: 'OpenAI',
    name: 'gpt-5.6-sol',
    description:
      "OpenAI's flagship model for complex professional work, advanced reasoning, coding, and long-running agentic tasks.",
    tags: ['Reasoning', 'Coding'],
    officialInput: '$5.00',
    officialOutput: '$30.00',
    saleInput: '$1.25',
    saleOutput: '$7.50',
  },
  {
    provider: 'OpenAI',
    name: 'gpt-5.6-terra',
    description:
      'Balances intelligence and cost for production workloads that need strong reasoning without flagship pricing.',
    tags: ['General', 'Efficient'],
    officialInput: '$2.50',
    officialOutput: '$15.00',
    saleInput: '$0.625',
    saleOutput: '$3.75',
  },
  {
    provider: 'Google',
    name: 'gemini-3.1-pro',
    description:
      "Google's preview Pro model for advanced multimodal understanding, agentic workflows, and complex coding tasks.",
    tags: ['Multimodal', 'Reasoning'],
    officialInput: '$2.00',
    officialOutput: '$12.00',
    saleInput: '$0.40',
    saleOutput: '$2.40',
  },
  {
    provider: 'Google',
    name: 'gemini-3.6-flash',
    description:
      'A production-ready Flash model that combines speed with strong intelligence for agentic and multimodal workloads.',
    tags: ['Multimodal', 'Efficient'],
    officialInput: '$1.50',
    officialOutput: '$7.50',
    saleInput: '$0.30',
    saleOutput: '$1.50',
  },
] as const

function Hero() {
  const { t } = useTranslation()
  const quickStarts = getQuickStarts(t)

  return (
    <section className='relative flex min-h-svh items-center overflow-hidden px-4 pt-24 pb-20 sm:px-6'>
      <div className='home-grid absolute inset-0 opacity-40' aria-hidden />
      <div className='absolute top-[-160px] left-1/2 h-[600px] w-[800px] -translate-x-1/2 rounded-full bg-[radial-gradient(circle,rgba(59,130,246,.24),rgba(139,92,246,.14)_40%,transparent_70%)] blur-3xl' />
      <div className='home-orb absolute top-1/4 left-[8%] size-64 rounded-full bg-blue-500/10 blur-3xl' />
      <div className='home-orb home-orb-delayed absolute right-[5%] bottom-[12%] size-80 rounded-full bg-violet-500/10 blur-3xl' />

      <div className='relative mx-auto flex w-full max-w-4xl flex-col items-center text-center'>
        <div className='home-hero-in home-delay-1 border-border bg-card/70 text-muted-foreground mb-7 inline-flex items-center gap-2 rounded-full border px-4 py-2 text-xs shadow-sm backdrop-blur'>
          <Zap className='size-3.5 text-blue-400' />
          {t(UNIROUTERS_HOME_HERO_COPY.taglineKey)}
        </div>
        <h1 className='home-hero-in home-delay-2 text-4xl leading-[1.05] font-black tracking-[-0.05em] sm:text-5xl md:text-7xl'>
          <span className='home-gradient-text'>uniRouters</span>
          <span className='text-foreground mt-2 block'>
            {t(UNIROUTERS_HOME_HERO_COPY.headlineKey)}
          </span>
        </h1>
        <p className='home-hero-in home-delay-3 text-muted-foreground mt-7 max-w-2xl text-base leading-8 sm:text-xl'>
          {t(
            'Stop maintaining separate AI provider integrations. Access Claude, GPT, Gemini, and 50+ leading models through one API with automatic routing and usage-based billing.'
          )}
        </p>
        <div className='home-hero-in home-delay-4 mt-9 flex w-full flex-col justify-center gap-3 sm:w-auto sm:flex-row'>
          <Link
            to='/sign-up'
            className='home-primary-button group inline-flex h-12 items-center justify-center gap-2 rounded-xl px-6 text-sm font-semibold text-white'
          >
            {t('Start for free')}
            <ArrowRight className='size-4 transition-transform group-hover:translate-x-1' />
          </Link>
          <Link
            to={UNIROUTERS_HOME_DOCS_PATH}
            className='border-border bg-background/60 text-foreground inline-flex h-12 items-center justify-center rounded-xl border px-6 text-sm font-semibold shadow-sm transition hover:border-blue-500/60 hover:bg-blue-500/5 hover:text-blue-600 dark:hover:text-blue-300'
          >
            {t('View documentation')}
          </Link>
        </div>
        <Tabs
          defaultValue='claude-code'
          className='home-hero-in home-delay-5 relative mt-12 w-full max-w-3xl gap-0 overflow-hidden rounded-2xl border border-white/10 bg-[#101018]/90 text-left shadow-2xl backdrop-blur-xl'
        >
          <div className='flex min-w-0 items-center gap-3 border-b border-white/8 px-4 py-3'>
            <div className='flex shrink-0 items-center gap-2'>
              <i className='size-3 rounded-full bg-red-400' />
              <i className='size-3 rounded-full bg-amber-400' />
              <i className='size-3 rounded-full bg-emerald-400' />
            </div>
            <span className='hidden shrink-0 font-mono text-xs text-slate-500 sm:block'>
              {t('footer.columns.docs.links.quickStart')}
            </span>
            <TabsList
              variant='line'
              className='no-scrollbar ml-auto h-8 max-w-full justify-start overflow-x-auto rounded-none p-0'
            >
              {quickStarts.map((quickStart) => (
                <TabsTrigger
                  key={quickStart.value}
                  value={quickStart.value}
                  className='h-8 flex-none px-2.5 font-mono text-xs text-slate-500 after:bg-blue-400 hover:text-slate-200 data-active:text-white dark:text-slate-500 dark:data-active:text-white'
                >
                  {quickStart.label}
                </TabsTrigger>
              ))}
            </TabsList>
          </div>
          {quickStarts.map((quickStart) => (
            <TabsContent key={quickStart.value} value={quickStart.value}>
              <pre className='min-h-[300px] overflow-x-auto p-5 font-mono text-xs leading-6 text-slate-300 sm:text-sm'>
                <code>{quickStart.code}</code>
              </pre>
            </TabsContent>
          ))}
          <Code2 className='pointer-events-none absolute right-5 bottom-5 size-6 text-white/10' />
        </Tabs>
      </div>
    </section>
  )
}

function FeatureGrid() {
  const { t } = useTranslation()

  return (
    <section className='px-4 py-20 sm:px-6 sm:py-32'>
      <div className='mx-auto max-w-7xl'>
        <AnimateInView className='mx-auto mb-14 max-w-2xl text-center'>
          <h2 className='text-foreground text-3xl font-bold tracking-tight sm:text-4xl'>
            {t('Why choose')}{' '}
            <span className='home-gradient-text'>uniRouters</span>
          </h2>
          <p className='text-muted-foreground mt-4 text-base leading-7'>
            {t(
              'More than an API proxy—a complete model access platform that handles routing, load balancing, and observability.'
            )}
          </p>
        </AnimateInView>
        <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-3'>
          {features.map((feature, index) => {
            const Icon = feature.icon
            return (
              <AnimateInView key={feature.title} delay={index * 100}>
                <article className='border-border bg-card/60 hover:bg-accent/60 group h-full rounded-2xl border p-6 shadow-sm transition duration-300'>
                  <div className='mb-5 flex size-12 items-center justify-center rounded-xl bg-blue-500/10 text-blue-400 transition-transform duration-300 group-hover:scale-110'>
                    <Icon className='size-6' />
                  </div>
                  <h3 className='text-foreground text-lg font-semibold'>
                    {t(feature.title)}
                  </h3>
                  <p className='text-muted-foreground mt-2 text-sm leading-6'>
                    {t(feature.description)}
                  </p>
                </article>
              </AnimateInView>
            )
          })}
        </div>
      </div>
    </section>
  )
}

function ModelShowcase() {
  const { t } = useTranslation()

  return (
    <section className='home-dots px-4 py-20 sm:px-6 sm:py-32'>
      <div className='mx-auto max-w-7xl'>
        <AnimateInView className='mx-auto mb-14 max-w-2xl text-center'>
          <h2 className='text-foreground text-3xl font-bold tracking-tight sm:text-4xl'>
            {t('Every leading model, covered')}
          </h2>
          <p className='text-muted-foreground mt-4 text-base leading-7'>
            {t(
              'From open source to commercial, general-purpose chat to specialized intelligence—reach it all through one API.'
            )}
          </p>
        </AnimateInView>
        <div className='grid gap-4 sm:grid-cols-2 lg:grid-cols-4'>
          {models.map((model, index) => (
            <AnimateInView key={model.name} delay={index * 100}>
              <article className='border-border bg-card flex h-full min-h-64 cursor-default flex-col rounded-2xl border p-5 shadow-sm transition duration-300 hover:-translate-y-1 hover:border-blue-500/50 hover:shadow-lg'>
                <div className='flex items-center gap-2 text-xs font-medium text-blue-400'>
                  <Sparkles className='size-4' />
                  {model.provider}
                </div>
                <h3 className='text-foreground mt-4 text-lg font-semibold'>
                  {model.name}
                </h3>
                <p className='text-muted-foreground mt-2 flex-1 text-sm leading-6'>
                  {t(model.description)}
                </p>
                <div className='mt-5 flex flex-wrap gap-2'>
                  {model.tags.map((tag) => (
                    <span
                      key={tag}
                      className='rounded-full border border-blue-400/20 bg-blue-500/10 px-2.5 py-1 text-[11px] text-blue-300'
                    >
                      {t(tag)}
                    </span>
                  ))}
                </div>
                <div className='border-border mt-5 border-t pt-4'>
                  <div className='space-y-3 font-mono'>
                    <div className='flex items-center justify-between gap-3'>
                      <div className='text-muted-foreground text-[10px] tracking-wider uppercase'>
                        {t('Input')}
                      </div>
                      <div className='flex items-baseline justify-end gap-1.5 text-xs'>
                        <del className='text-muted-foreground'>
                          {model.officialInput}
                        </del>
                        <strong className='text-sm font-semibold text-emerald-500'>
                          {model.saleInput}
                          <span className='text-muted-foreground ml-1 text-[10px] font-normal'>
                            /M tokens
                          </span>
                        </strong>
                      </div>
                    </div>
                    <div className='flex items-center justify-between gap-3'>
                      <div className='text-muted-foreground text-[10px] tracking-wider uppercase'>
                        {t('Output')}
                      </div>
                      <div className='flex items-baseline justify-end gap-1.5 text-xs'>
                        <del className='text-muted-foreground'>
                          {model.officialOutput}
                        </del>
                        <strong className='text-sm font-semibold text-emerald-500'>
                          {model.saleOutput}
                          <span className='text-muted-foreground ml-1 text-[10px] font-normal'>
                            /M tokens
                          </span>
                        </strong>
                      </div>
                    </div>
                  </div>
                </div>
              </article>
            </AnimateInView>
          ))}
        </div>
        <div className='mt-10 text-center'>
          <Link
            to='/models'
            className='text-muted-foreground hover:bg-accent hover:text-foreground group inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm transition'
          >
            {t('View all 50+ models')}
            <ArrowRight className='size-4 transition-transform group-hover:translate-x-1' />
          </Link>
        </div>
      </div>
    </section>
  )
}

function CallToAction(props: { isAuthenticated: boolean }) {
  const { t } = useTranslation()
  if (props.isAuthenticated) return null

  return (
    <section className='relative overflow-hidden px-4 py-20 sm:px-6 sm:py-32'>
      <div className='absolute bottom-0 left-0 h-96 w-[500px] rounded-full bg-cyan-500/10 blur-3xl' />
      <div className='absolute top-0 right-0 h-96 w-[500px] rounded-full bg-violet-500/10 blur-3xl' />
      <AnimateInView className='border-border bg-card/70 relative mx-auto max-w-4xl rounded-3xl border px-8 py-12 text-center shadow-xl backdrop-blur-xl sm:p-16'>
        <div className='absolute inset-0 rounded-3xl bg-gradient-to-b from-blue-500/5 via-transparent to-violet-500/5' />
        <div className='relative'>
          <h2 className='text-foreground text-3xl font-bold sm:text-4xl'>
            {t('Ready to get started?')}
          </h2>
          <p className='text-muted-foreground mx-auto mt-4 max-w-xl'>
            {t(
              'Create a free account and receive 10K tokens every month. No credit card required.'
            )}
          </p>
          <div className='mt-8 flex flex-col justify-center gap-3 sm:flex-row'>
            <Link
              to='/sign-up'
              className='home-primary-button group inline-flex h-12 items-center justify-center gap-2 rounded-xl px-6 text-sm font-semibold text-white'
            >
              {t('Register for free')}
              <ArrowRight className='size-4 transition-transform group-hover:translate-x-1' />
            </Link>
            <Link
              to='/pricing'
              className='border-border bg-background/60 text-foreground inline-flex h-12 items-center justify-center rounded-xl border px-6 text-sm font-semibold transition hover:border-blue-500/60 hover:bg-blue-500/5 hover:text-blue-600 dark:hover:text-blue-300'
            >
              {t('View pricing')}
            </Link>
          </div>
        </div>
      </AnimateInView>
    </section>
  )
}

export function LandingPage(props: { isAuthenticated: boolean }) {
  return (
    <div className='home-landing bg-background text-foreground'>
      <Hero />
      <Stats />
      <FeatureGrid />
      <ModelShowcase />
      <CallToAction isAuthenticated={props.isAuthenticated} />
      <FaqSection />
    </div>
  )
}
