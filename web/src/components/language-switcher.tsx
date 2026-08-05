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
import { Languages, Check } from 'lucide-react'
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
  type InterfaceLanguageCode,
} from '@/i18n/languages'
import { api } from '@/lib/api'
import { cn } from '@/lib/utils'
import { useAuthStore } from '@/stores/auth-store'

type LanguageSwitcherProps = {
  variant?: 'dropdown' | 'segmented'
}

const SEGMENTED_LANGUAGE_OPTIONS = [
  { code: 'en', label: 'EN', name: 'English' },
  { code: 'zhCN', label: 'CN', name: 'Chinese' },
] as const

export function LanguageSwitcher(props: LanguageSwitcherProps = {}) {
  const { i18n, t } = useTranslation()
  const user = useAuthStore((s) => s.auth.user)
  const currentLanguage = normalizeInterfaceLanguage(i18n.language)
  const handleChangeLanguage = useCallback(
    async (code: InterfaceLanguageCode) => {
      if (code === currentLanguage) return

      await i18n.changeLanguage(code)
      if (user) {
        try {
          await api.put('/api/user/self', { language: code })
        } catch {
          // Best-effort persistence; don't block the UI on failure
        }
      }
    },
    [currentLanguage, i18n, user]
  )

  if (props.variant === 'segmented') {
    return (
      <div
        role='group'
        aria-label={t('Change language')}
        className='border-border/60 bg-muted/60 inline-flex h-8 items-center rounded-lg border p-0.5'
      >
        {SEGMENTED_LANGUAGE_OPTIONS.map((language) => {
          const isActive = currentLanguage === language.code

          return (
            <Button
              key={language.code}
              type='button'
              variant='ghost'
              size='sm'
              aria-label={`${t('Change language')}: ${t(language.name)}`}
              aria-pressed={isActive}
              className={cn(
                'h-7 min-w-9 rounded-md px-2 text-[11px] font-semibold tracking-wide',
                isActive
                  ? 'bg-primary text-primary-foreground shadow-sm hover:bg-primary/90 hover:text-primary-foreground'
                  : 'text-muted-foreground hover:bg-background/60 hover:text-foreground'
              )}
              onClick={() => void handleChangeLanguage(language.code)}
            >
              {language.label}
            </Button>
          )
        })}
      </div>
    )
  }

  return (
    <DropdownMenu modal={false}>
      <DropdownMenuTrigger
        render={<Button variant='ghost' size='icon' className='h-9 w-9' />}
      >
        <Languages className='size-[1.2rem]' />
        <span className='sr-only'>{t('Change language')}</span>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end'>
        {INTERFACE_LANGUAGE_OPTIONS.map((lang) => (
          <DropdownMenuItem
            key={lang.code}
            onClick={() => void handleChangeLanguage(lang.code)}
          >
            {lang.label}
            <Check
              size={14}
              className={cn(
                'ms-auto',
                currentLanguage !== lang.code && 'hidden'
              )}
            />
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
