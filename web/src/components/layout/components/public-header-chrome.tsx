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
import type { ReactNode } from 'react'

import { cn } from '@/lib/utils'

type PublicHeaderChromeProps = {
  children: ReactNode
  scrolled: boolean
}

export function PublicHeaderChrome(props: PublicHeaderChromeProps) {
  return (
    <header className='pointer-events-none fixed inset-x-0 top-0 z-50'>
      <div
        data-slot='public-header-frame'
        className={cn(
          'pointer-events-auto mx-auto w-full max-w-7xl px-4 transition-all duration-700 ease-[cubic-bezier(0.16,1,0.3,1)] md:px-6',
          props.scrolled ? 'pt-3' : 'pt-0'
        )}
      >
        <nav
          data-slot='public-header-nav'
          className={cn(
            'flex h-16 items-center justify-between px-2 transition-all duration-700 ease-[cubic-bezier(0.16,1,0.3,1)]',
            props.scrolled &&
              'bg-background/60 ring-border/50 rounded-2xl shadow-[0_2px_16px_-6px_rgba(0,0,0,0.08),0_0_0_0.5px_rgba(0,0,0,0.02)] ring-[0.5px] backdrop-blur-2xl dark:shadow-[0_2px_16px_-6px_rgba(0,0,0,0.4)]'
          )}
        >
          {props.children}
        </nav>
      </div>
    </header>
  )
}
