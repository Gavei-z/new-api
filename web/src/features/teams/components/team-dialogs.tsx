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
import { useEffect, useRef, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Textarea } from '@/components/ui/textarea'
import { parseQuotaFromDollars } from '@/lib/format'

import { getErrorMessage } from '../lib'
import type { CreateTeamInput, CreateTeamMemberInput } from '../types'

const memberSchema = z.object({
  username: z.string().trim().min(3).max(20),
  password: z.string().min(8).max(20),
  display_name: z.string().trim().min(1).max(20),
  email: z.string().trim().email().max(50).or(z.literal('')),
})

const teamSchema = z.object({
  name: z.string().trim().min(2).max(120),
  slug: z
    .string()
    .trim()
    .min(3)
    .max(64)
    .regex(/^[a-z0-9][a-z0-9-]+[a-z0-9]$/),
  owner_username: z.string().trim().min(3).max(20),
  owner_password: z.string().min(8).max(20),
  owner_display_name: z.string().trim().min(1).max(20),
  owner_email: z.string().trim().email().max(50).or(z.literal('')),
})

const quotaSchema = z.object({
  amount: z.string().refine((value) => Number(value) > 0),
  operation: z.enum(['add', 'subtract']),
  note: z.string().trim().max(255),
})

const passwordSchema = z.object({
  password: z.string().min(8).max(20),
})

type MemberValues = z.infer<typeof memberSchema>
type TeamValues = z.infer<typeof teamSchema>
type QuotaValues = z.infer<typeof quotaSchema>
type PasswordValues = z.infer<typeof passwordSchema>

type ControlledDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CreateMemberDialog(
  props: ControlledDialogProps & {
    onSubmit: (values: CreateTeamMemberInput) => Promise<void>
  }
) {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const form = useForm<MemberValues>({
    resolver: zodResolver(memberSchema),
    defaultValues: {
      username: '',
      password: '',
      display_name: '',
      email: '',
    },
  })

  useEffect(() => {
    if (!props.open) form.reset()
  }, [form, props.open])

  const submit = async (values: MemberValues) => {
    setSubmitting(true)
    try {
      await props.onSubmit(values)
      toast.success(t('Team member created'))
      props.onOpenChange(false)
    } catch (error) {
      toast.error(getErrorMessage(error, t('Failed to create team member')))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-lg'>
        <DialogHeader>
          <DialogTitle>{t('Create Team Member')}</DialogTitle>
          <DialogDescription>
            {t(
              'The new account is forced into this team and uses the shared team balance.'
            )}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form className='grid gap-4' onSubmit={form.handleSubmit(submit)}>
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='username'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Username')}</FormLabel>
                    <FormControl>
                      <Input autoComplete='off' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='display_name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Display Name')}</FormLabel>
                    <FormControl>
                      <Input autoComplete='off' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <FormField
              control={form.control}
              name='email'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Email')}</FormLabel>
                  <FormControl>
                    <Input type='email' autoComplete='off' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='password'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Initial Password')}</FormLabel>
                  <FormControl>
                    <Input
                      type='password'
                      autoComplete='new-password'
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => props.onOpenChange(false)}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={submitting}>
                {submitting ? t('Creating...') : t('Create Member')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

export function CreateTeamDialog(
  props: ControlledDialogProps & {
    onSubmit: (values: CreateTeamInput) => Promise<void>
  }
) {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const form = useForm<TeamValues>({
    resolver: zodResolver(teamSchema),
    defaultValues: {
      name: '',
      slug: '',
      owner_username: '',
      owner_password: '',
      owner_display_name: '',
      owner_email: '',
    },
  })

  useEffect(() => {
    if (!props.open) form.reset()
  }, [form, props.open])

  const submit = async (values: TeamValues) => {
    setSubmitting(true)
    try {
      await props.onSubmit({
        name: values.name,
        slug: values.slug.toLowerCase(),
        owner: {
          username: values.owner_username,
          password: values.owner_password,
          display_name: values.owner_display_name,
          email: values.owner_email,
        },
      })
      toast.success(t('Team created'))
      props.onOpenChange(false)
    } catch (error) {
      toast.error(getErrorMessage(error, t('Failed to create team')))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>{t('Create Enterprise Team')}</DialogTitle>
          <DialogDescription>
            {t(
              'Create an isolated tenant and its first owner account. The owner remains a normal user globally.'
            )}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form className='grid gap-4' onSubmit={form.handleSubmit(submit)}>
            <div className='grid gap-4 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='name'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Team Name')}</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name='slug'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Team Slug')}</FormLabel>
                    <FormControl>
                      <Input placeholder='acme-team' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
            <div className='border-border border-t pt-4'>
              <p className='mb-3 text-sm font-medium'>{t('Owner Account')}</p>
              <div className='grid gap-4 sm:grid-cols-2'>
                <FormField
                  control={form.control}
                  name='owner_username'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Username')}</FormLabel>
                      <FormControl>
                        <Input autoComplete='off' {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='owner_display_name'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Display Name')}</FormLabel>
                      <FormControl>
                        <Input autoComplete='off' {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='owner_email'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Email')}</FormLabel>
                      <FormControl>
                        <Input type='email' autoComplete='off' {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name='owner_password'
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('Initial Password')}</FormLabel>
                      <FormControl>
                        <Input
                          type='password'
                          autoComplete='new-password'
                          {...field}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => props.onOpenChange(false)}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={submitting}>
                {submitting ? t('Creating...') : t('Create Team')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

export function QuotaDialog(
  props: ControlledDialogProps & {
    kind: 'fund' | 'adjust'
    onSubmit: (values: {
      amount: number
      operation: 'add' | 'subtract'
      note: string
    }) => Promise<void>
  }
) {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const submittingRef = useRef(false)
  const form = useForm<QuotaValues>({
    resolver: zodResolver(quotaSchema),
    defaultValues: { amount: '', operation: 'add', note: '' },
  })

  useEffect(() => {
    if (!props.open) form.reset()
  }, [form, props.open])

  const submit = async (values: QuotaValues) => {
    if (submittingRef.current) return
    const amount = parseQuotaFromDollars(Number(values.amount))
    if (amount <= 0) {
      toast.error(t('Amount must be greater than zero'))
      return
    }
    submittingRef.current = true
    setSubmitting(true)
    try {
      await props.onSubmit({ ...values, amount })
      toast.success(
        props.kind === 'fund'
          ? t('Balance transferred to team')
          : t('Team balance updated')
      )
      props.onOpenChange(false)
    } catch (error) {
      toast.error(getErrorMessage(error, t('Failed to update team balance')))
    } finally {
      submittingRef.current = false
      setSubmitting(false)
    }
  }

  const isFund = props.kind === 'fund'
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {isFund ? t('Transfer to Team Balance') : t('Adjust Team Balance')}
          </DialogTitle>
          <DialogDescription>
            {isFund
              ? t(
                  'Transfer available quota from your personal wallet into the shared team wallet.'
                )
              : t(
                  'Every adjustment is written to the immutable team balance ledger.'
                )}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form className='grid gap-4' onSubmit={form.handleSubmit(submit)}>
            {!isFund && (
              <FormField
                control={form.control}
                name='operation'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Operation')}</FormLabel>
                    <FormControl>
                      <NativeSelect
                        className='w-full'
                        value={field.value}
                        onChange={field.onChange}
                      >
                        <NativeSelectOption value='add'>
                          {t('Add Balance')}
                        </NativeSelectOption>
                        <NativeSelectOption value='subtract'>
                          {t('Subtract Balance')}
                        </NativeSelectOption>
                      </NativeSelect>
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <FormField
              control={form.control}
              name='amount'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Display Amount')}</FormLabel>
                  <FormControl>
                    <Input min='0' step='0.01' type='number' {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {!isFund && (
              <FormField
                control={form.control}
                name='note'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Audit Note')}</FormLabel>
                    <FormControl>
                      <Textarea className='min-h-20' {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            )}
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => props.onOpenChange(false)}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={submitting}>
                {submitting ? t('Submitting...') : t('Confirm')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

export function ResetPasswordDialog(
  props: ControlledDialogProps & {
    username: string
    onSubmit: (password: string) => Promise<void>
  }
) {
  const { t } = useTranslation()
  const [submitting, setSubmitting] = useState(false)
  const form = useForm<PasswordValues>({
    resolver: zodResolver(passwordSchema),
    defaultValues: { password: '' },
  })

  useEffect(() => {
    if (!props.open) form.reset()
  }, [form, props.open])

  const submit = async (values: PasswordValues) => {
    setSubmitting(true)
    try {
      await props.onSubmit(values.password)
      toast.success(t('Password reset'))
      props.onOpenChange(false)
    } catch (error) {
      toast.error(getErrorMessage(error, t('Failed to reset password')))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('Reset Member Password')}</DialogTitle>
          <DialogDescription>
            {t('Set a new password for {{username}}.', {
              username: props.username,
            })}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form className='grid gap-4' onSubmit={form.handleSubmit(submit)}>
            <FormField
              control={form.control}
              name='password'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('New Password')}</FormLabel>
                  <FormControl>
                    <Input
                      type='password'
                      autoComplete='new-password'
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button
                type='button'
                variant='outline'
                onClick={() => props.onOpenChange(false)}
              >
                {t('Cancel')}
              </Button>
              <Button type='submit' disabled={submitting}>
                {submitting ? t('Submitting...') : t('Reset Password')}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}
