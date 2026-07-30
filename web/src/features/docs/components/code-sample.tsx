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
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { cn } from '@/lib/utils'

interface CodeSampleProps {
  code: string
  label: string
  language?: string
  className?: string
}

export function CodeSample(props: CodeSampleProps) {
  const { t } = useTranslation()

  return (
    <div
      className={cn(
        'border-border/60 bg-card/70 overflow-hidden rounded-xl border shadow-sm',
        props.className
      )}
    >
      <div className='border-border/50 bg-muted/35 flex h-11 items-center justify-between border-b px-3.5'>
        <div className='flex min-w-0 items-center gap-2'>
          <span className='truncate text-xs font-medium'>{props.label}</span>
          {props.language && (
            <span className='text-muted-foreground/60 text-[10px] font-medium tracking-wider uppercase'>
              {props.language}
            </span>
          )}
        </div>
        <CopyButton
          value={props.code}
          className='size-8'
          iconClassName='size-3.5'
          aria-label={t('docs.actions.copyCode')}
        />
      </div>
      <pre
        className='overflow-x-auto p-4 text-[12px] leading-6 sm:text-[13px]'
        tabIndex={0}
      >
        <code>{props.code}</code>
      </pre>
    </div>
  )
}
