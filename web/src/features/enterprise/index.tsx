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
import { zodResolver } from '@hookform/resolvers/zod'
import {
  ArrowRight,
  BadgeCheck,
  BarChart3,
  Building2,
  Check,
  CircleDollarSign,
  KeyRound,
  UsersRound,
} from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { PublicLayout } from '@/components/layout'
import { Footer } from '@/components/layout/components/footer'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { createEnterpriseInquiry } from '@/features/teams/api'
import { getErrorMessage } from '@/features/teams/lib'

import { ENTERPRISE_QUOTE_BENEFITS } from './constants'

const inquirySchema = z
  .object({
    company_name: z.string().trim().min(2).max(120),
    contact_name: z.string().trim().min(2).max(80),
    email: z.string().trim().email().max(120).or(z.literal('')),
    phone: z.string().trim().max(40),
    wechat: z.string().trim().max(80),
    team_size: z.string().max(40),
    message: z.string().trim().max(2000),
    privacy_accepted: z.boolean().refine(Boolean),
    website: z.string().max(0),
  })
  .refine((values) => Boolean(values.email || values.phone || values.wechat), {
    message: 'Please provide at least one contact method',
    path: ['email'],
  })

type InquiryValues = z.infer<typeof inquirySchema>

const defaultValues: InquiryValues = {
  company_name: '',
  contact_name: '',
  email: '',
  phone: '',
  wechat: '',
  team_size: '',
  message: '',
  privacy_accepted: false,
  website: '',
}

function EnterpriseInquiryForm() {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const [submitted, setSubmitted] = useState(false)
  const form = useForm<InquiryValues>({
    resolver: zodResolver(inquirySchema),
    defaultValues,
  })

  const submit = async (values: InquiryValues) => {
    setSubmitting(true)
    try {
      await createEnterpriseInquiry(values)
      setSubmitted(true)
      form.reset(defaultValues)
      toast.success(t('Your enterprise inquiry has been submitted'))
    } catch (error) {
      toast.error(
        getErrorMessage(error, t('Failed to submit enterprise inquiry'))
      )
    } finally {
      setSubmitting(false)
    }
  }

  if (submitted) {
    return (
      <Card className='border-emerald-500/30 bg-emerald-500/5'>
        <CardContent className='flex min-h-80 flex-col items-center justify-center p-8 text-center'>
          <div className='mb-5 flex size-14 items-center justify-center rounded-full bg-emerald-500/10 text-emerald-600'>
            <Check className='size-7' />
          </div>
          <h3 className='text-xl font-semibold'>
            {t('Thanks, we received your request')}
          </h3>
          <p className='text-muted-foreground mt-2 max-w-md'>
            {t(
              'Our enterprise team will review your requirements and contact you using the information provided.'
            )}
          </p>
          <Button
            variant='outline'
            className='mt-6'
            onClick={() => setSubmitted(false)}
          >
            {t('Submit another inquiry')}
          </Button>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className='shadow-xl shadow-blue-950/5'>
      <CardContent className='p-5 sm:p-7'>
        <Form {...form}>
          <form className='grid gap-5' onSubmit={form.handleSubmit(submit)}>
            <div className='grid gap-5 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='company_name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Company Name')}</FormLabel>
                    <FormControl>
                      <Input placeholder={t('Your company')} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='contact_name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Contact Name')}</FormLabel>
                    <FormControl>
                      <Input
                        placeholder={t('How should we address you?')}
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <div className='grid gap-5 sm:grid-cols-3'>
              <FormField
                control={form.control}
                name='email'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Work Email')}</FormLabel>
                    <FormControl>
                      <Input
                        type='email'
                        placeholder='name@company.com'
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='phone'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Phone')}</FormLabel>
                    <FormControl>
                      <Input placeholder='+86' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='wechat'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('WeChat')}</FormLabel>
                    <FormControl>
                      <Input placeholder={t('WeChat ID')} {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>

            <FormField
              control={form.control}
              name='team_size'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Expected Team Size')}</FormLabel>
                  <FormControl>
                    <NativeSelect
                      className='w-full'
                      value={field.value}
                      onChange={field.onChange}
                    >
                      <NativeSelectOption value=''>
                        {t('Select team size')}
                      </NativeSelectOption>
                      <NativeSelectOption value='1-10'>1–10</NativeSelectOption>
                      <NativeSelectOption value='11-50'>
                        11–50
                      </NativeSelectOption>
                      <NativeSelectOption value='51-200'>
                        51–200
                      </NativeSelectOption>
                      <NativeSelectOption value='201+'>201+</NativeSelectOption>
                    </NativeSelect>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='message'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Notes')}</FormLabel>
                  <FormControl>
                    <Textarea
                      className='min-h-28'
                      placeholder={t(
                        'Add any preferred models, concurrency, compliance, support, or onboarding notes.'
                      )}
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('No API keys or other secrets, please.')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='website'
              render={({ field }) => (
                <div
                  aria-hidden='true'
                  className='absolute -left-[10000px] h-px w-px overflow-hidden'
                >
                  <label htmlFor='enterprise-website'>Website</label>
                  <input
                    id='enterprise-website'
                    tabIndex={-1}
                    autoComplete='off'
                    {...field}
                  />
                </div>
              )}
            />

            <FormField
              control={form.control}
              name='privacy_accepted'
              render={({ field }) => (
                <FormItem className='flex grid-cols-[auto_1fr] items-start gap-x-3 gap-y-1'>
                  <FormControl>
                    <Checkbox
                      checked={field.value}
                      onCheckedChange={(checked) =>
                        field.onChange(checked === true)
                      }
                    />
                  </FormControl>
                  <div>
                    <FormLabel className='font-normal'>
                      {t(
                        'I agree that UniRouters may use this information to contact me about enterprise services.'
                      )}
                    </FormLabel>
                    <FormMessage />
                  </div>
                </FormItem>
              )}
            />

            <Button type='submit' size='lg' disabled={submitting}>
              {submitting ? t('Submitting...') : t('Get Enterprise Quote')}
              {!submitting && <ArrowRight />}
            </Button>
            <p className='text-muted-foreground text-center text-xs'>
              {t(
                'Submitting this form does not create an account or commit you to a purchase.'
              )}
            </p>
          </form>
        </Form>
      </CardContent>
    </Card>
  )
}

const capabilities = [
  {
    icon: UsersRound,
    title: 'Team member management',
    description:
      'Create member accounts within your team and manage access from one workspace.',
  },
  {
    icon: CircleDollarSign,
    title: 'One shared team balance',
    description:
      'Fund one team wallet and let every team member consume from the same auditable balance.',
  },
  {
    icon: BarChart3,
    title: 'Member and model usage',
    description:
      'Understand requests, token volume, and cost by member and model from one team console.',
  },
]

const steps = [
  {
    number: '01',
    title: 'Tell us what you need',
    description:
      'Share your team size, preferred models, and any notes that will help us understand your needs.',
  },
  {
    number: '02',
    title: 'Confirm the enterprise plan',
    description:
      'We align the commercial terms, funding method, operational limits, and onboarding schedule.',
  },
  {
    number: '03',
    title: 'Invite your team',
    description:
      'Your team owner creates member accounts and tracks shared usage from the enterprise console.',
  },
]

export function Enterprise() {
  const { t } = useTranslation()

  return (
    <PublicLayout showMainContainer={false}>
      <main>
        <section className='relative overflow-hidden border-b pt-28 pb-20 sm:pt-36 sm:pb-28'>
          <div className='pointer-events-none absolute inset-0'>
            <div className='absolute top-0 left-1/2 h-[36rem] w-[60rem] -translate-x-1/2 rounded-full bg-blue-500/10 blur-3xl' />
            <div className='absolute right-0 bottom-0 size-80 rounded-full bg-cyan-400/10 blur-3xl' />
          </div>
          <div className='relative container mx-auto px-4'>
            <div className='mx-auto max-w-4xl text-center'>
              <div className='border-border bg-background/70 mb-6 inline-flex items-center gap-2 rounded-full border px-3 py-1 text-sm backdrop-blur'>
                <Building2 className='text-primary size-4' />
                {t('UniRouters Enterprise Service')}
              </div>
              <h1 className='text-4xl font-bold tracking-tight text-balance sm:text-6xl'>
                {t('A shared AI gateway built for real teams')}
              </h1>
              <p className='text-muted-foreground mx-auto mt-6 max-w-2xl text-lg leading-8 text-balance'>
                {t(
                  'Centralize AI access, shared funding, member management, and usage visibility for your team in one place.'
                )}
              </p>
              <div className='mt-8 flex flex-col justify-center gap-3 sm:flex-row'>
                <Button size='lg' render={<a href='#enterprise-quote' />}>
                  {t('Get Enterprise Quote')}
                  <ArrowRight />
                </Button>
                <Button size='lg' variant='outline' render={<a href='/docs' />}>
                  {t('Read the Docs')}
                </Button>
              </div>
              <div className='text-muted-foreground mt-8 flex flex-wrap justify-center gap-x-6 gap-y-2 text-sm'>
                {[
                  'Dedicated team workspace',
                  'Auditable balance ledger',
                  'Role-aware console',
                ].map((item) => (
                  <span key={item} className='flex items-center gap-1.5'>
                    <BadgeCheck className='size-4 text-emerald-600' />
                    {t(item)}
                  </span>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section className='py-20 sm:py-28'>
          <div className='container mx-auto px-4'>
            <div className='mx-auto max-w-2xl text-center'>
              <p className='text-primary text-sm font-semibold tracking-wider uppercase'>
                {t('Designed for team collaboration')}
              </p>
              <h2 className='mt-3 text-3xl font-bold tracking-tight sm:text-4xl'>
                {t('Everything your team needs in one workspace')}
              </h2>
            </div>
            <div className='mx-auto mt-12 grid max-w-6xl gap-4 md:grid-cols-3'>
              {capabilities.map((capability) => {
                const Icon = capability.icon
                return (
                  <Card key={capability.title} className='bg-muted/20'>
                    <CardContent className='p-6'>
                      <div className='bg-primary/10 text-primary mb-5 flex size-11 items-center justify-center rounded-xl'>
                        <Icon className='size-5' />
                      </div>
                      <h3 className='text-lg font-semibold'>
                        {t(capability.title)}
                      </h3>
                      <p className='text-muted-foreground mt-2 leading-6'>
                        {t(capability.description)}
                      </p>
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          </div>
        </section>

        <section className='bg-muted/35 border-y py-20 sm:py-28'>
          <div className='container mx-auto px-4'>
            <div className='mx-auto grid max-w-6xl gap-12 lg:grid-cols-[1fr_1.1fr] lg:items-center'>
              <div>
                <p className='text-primary text-sm font-semibold tracking-wider uppercase'>
                  {t('Team collaboration')}
                </p>
                <h2 className='mt-3 text-3xl font-bold tracking-tight'>
                  {t('Clear roles, smooth collaboration')}
                </h2>
                <p className='text-muted-foreground mt-4 leading-7'>
                  {t(
                    'Team owners can organize members, shared funding, and usage insights from one enterprise console.'
                  )}
                </p>
                <div className='mt-7 space-y-4'>
                  {[
                    {
                      icon: KeyRound,
                      title: 'Team Owner',
                      text: 'Create member accounts, manage shared funding, and review team usage.',
                    },
                    {
                      icon: UsersRound,
                      title: 'Team member',
                      text: "Use the team's AI models and API access through the shared balance.",
                    },
                    {
                      icon: BarChart3,
                      title: 'Usage insights',
                      text: 'Review usage by member and model to support budgeting and planning.',
                    },
                  ].map((item) => {
                    const Icon = item.icon
                    return (
                      <div key={item.title} className='flex gap-3'>
                        <Icon className='text-primary mt-0.5 size-5 shrink-0' />
                        <div>
                          <p className='font-medium'>{t(item.title)}</p>
                          <p className='text-muted-foreground mt-0.5 text-sm leading-6'>
                            {t(item.text)}
                          </p>
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
              <Card className='bg-background'>
                <CardContent className='p-6 sm:p-8'>
                  <div className='mb-6 flex items-center gap-3'>
                    <div className='bg-primary/10 flex size-10 items-center justify-center rounded-xl'>
                      <Building2 className='text-primary size-5' />
                    </div>
                    <div>
                      <p className='font-semibold'>{t('Example Team')}</p>
                      <p className='text-muted-foreground text-sm'>
                        acme-research
                      </p>
                    </div>
                    <BadgeCheck className='ml-auto size-5 text-emerald-600' />
                  </div>
                  <div className='grid gap-3 sm:grid-cols-2'>
                    {[
                      ['Available balance', 'Shared wallet'],
                      ['Usage visibility', 'By member + model'],
                      ['Member accounts', 'Managed by the team'],
                      ['Billing history', 'Auditable'],
                    ].map(([label, value]) => (
                      <div
                        key={label}
                        className='bg-muted/50 rounded-lg border p-4'
                      >
                        <p className='text-muted-foreground text-xs'>
                          {t(label)}
                        </p>
                        <p className='mt-1 font-medium'>{t(value)}</p>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        </section>

        <section className='py-20 sm:py-28'>
          <div className='container mx-auto px-4'>
            <div className='mx-auto max-w-6xl'>
              <div className='max-w-2xl'>
                <p className='text-primary text-sm font-semibold tracking-wider uppercase'>
                  {t('Simple onboarding')}
                </p>
                <h2 className='mt-3 text-3xl font-bold tracking-tight'>
                  {t('From requirements to a working team')}
                </h2>
              </div>
              <div className='mt-10 grid gap-4 md:grid-cols-3'>
                {steps.map((step) => (
                  <div
                    key={step.number}
                    className='border-border relative rounded-xl border p-6'
                  >
                    <span className='text-primary/60 text-sm font-semibold'>
                      {step.number}
                    </span>
                    <h3 className='mt-4 text-lg font-semibold'>
                      {t(step.title)}
                    </h3>
                    <p className='text-muted-foreground mt-2 leading-6'>
                      {t(step.description)}
                    </p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </section>

        <section
          id='enterprise-quote'
          className='bg-muted/35 scroll-mt-16 border-t py-20 sm:py-28'
        >
          <div className='container mx-auto px-4'>
            <div className='mx-auto grid max-w-6xl gap-10 lg:grid-cols-[0.8fr_1.2fr]'>
              <div className='lg:pt-8'>
                <p className='text-primary text-sm font-semibold tracking-wider uppercase'>
                  {t('Enterprise quote')}
                </p>
                <h2 className='mt-3 text-3xl font-bold tracking-tight sm:text-4xl'>
                  {t('Tell us about your team')}
                </h2>
                <p className='text-muted-foreground mt-4 leading-7'>
                  {t(
                    'Leave your notes and a contact method. We will use them only to discuss your enterprise service needs.'
                  )}
                </p>
                <div className='mt-8 space-y-3 text-sm'>
                  {ENTERPRISE_QUOTE_BENEFITS.map((item) => (
                    <div key={item} className='flex items-center gap-2'>
                      <Check className='size-4 text-emerald-600' />
                      {t(item)}
                    </div>
                  ))}
                </div>
              </div>
              <EnterpriseInquiryForm />
            </div>
          </div>
        </section>
      </main>
      <Footer />
    </PublicLayout>
  )
}
