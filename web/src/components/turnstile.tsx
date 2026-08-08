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
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

type TurnstileWidgetOptions = {
  sitekey: string
  callback: (token: string) => void
  'error-callback': () => void
  'expired-callback': () => void
  'timeout-callback': () => void
  retry: 'auto'
  'retry-interval': number
  'refresh-expired': 'auto'
  'refresh-timeout': 'auto'
}

type TurnstileApi = {
  render: (element: HTMLElement, options: TurnstileWidgetOptions) => string
  remove: (widgetId: string) => void
}

declare global {
  interface Window {
    turnstile?: TurnstileApi
  }
}

interface TurnstileProps {
  siteKey: string
  onVerify: (token: string) => void
  onExpire?: () => void
  onError?: () => void
  className?: string
}

const turnstileScriptId = 'cf-turnstile'
const turnstileScriptUrl =
  'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'
const turnstileLoadTimeoutMs = 10_000

let pendingTurnstileLoad: Promise<TurnstileApi> | null = null

function loadTurnstile(): Promise<TurnstileApi> {
  if (window.turnstile) return Promise.resolve(window.turnstile)
  if (pendingTurnstileLoad) return pendingTurnstileLoad

  const loadPromise = new Promise<TurnstileApi>((resolve, reject) => {
    const existingElement = document.querySelector(`#${turnstileScriptId}`)
    if (existingElement && !(existingElement instanceof HTMLScriptElement)) {
      existingElement.remove()
    }

    let script = document.querySelector<HTMLScriptElement>(
      `#${turnstileScriptId}`
    )
    const isNewScript = !script
    if (!script) {
      script = document.createElement('script')
      script.id = turnstileScriptId
      script.src = turnstileScriptUrl
      script.async = true
      script.defer = true
    }

    let settled = false
    const timeoutId = window.setTimeout(() => {
      fail(new Error('Turnstile script load timed out'))
    }, turnstileLoadTimeoutMs)

    const cleanup = () => {
      window.clearTimeout(timeoutId)
      script?.removeEventListener('load', handleLoad)
      script?.removeEventListener('error', handleError)
    }
    const succeed = () => {
      if (settled || !window.turnstile) return
      settled = true
      cleanup()
      resolve(window.turnstile)
    }
    const fail = (error: Error) => {
      if (settled) return
      settled = true
      cleanup()
      if (!window.turnstile && script?.isConnected) script.remove()
      reject(error)
    }
    function handleLoad() {
      if (window.turnstile) {
        succeed()
      } else {
        fail(new Error('Turnstile API was unavailable after script load'))
      }
    }
    function handleError() {
      fail(new Error('Turnstile script failed to load'))
    }

    script.addEventListener('load', handleLoad, { once: true })
    script.addEventListener('error', handleError, { once: true })
    // Recheck after attaching listeners so a load event between the initial
    // check and listener registration cannot strand a remounted route.
    if (window.turnstile) {
      succeed()
    } else if (isNewScript) {
      document.head.appendChild(script)
    }
  })

  pendingTurnstileLoad = loadPromise
  const clearPendingLoad = () => {
    if (pendingTurnstileLoad === loadPromise) pendingTurnstileLoad = null
  }
  void loadPromise.then(clearPendingLoad, clearPendingLoad)
  return loadPromise
}

export function Turnstile(props: TurnstileProps) {
  const { t } = useTranslation()
  const containerRef = useRef<HTMLDivElement | null>(null)
  const onVerifyRef = useRef(props.onVerify)
  const onExpireRef = useRef(props.onExpire)
  const onErrorRef = useRef(props.onError)
  const [attempt, setAttempt] = useState(0)
  const [phase, setPhase] = useState<'loading' | 'widget' | 'error'>('loading')

  onVerifyRef.current = props.onVerify
  onExpireRef.current = props.onExpire
  onErrorRef.current = props.onError

  useEffect(() => {
    let disposed = false
    let failed = false
    let widgetId: string | undefined
    let api: TurnstileApi | undefined
    setPhase('loading')

    const reportError = () => {
      if (disposed || failed) return
      failed = true
      setPhase('error')
      onErrorRef.current?.()
    }

    void loadTurnstile().then((loadedApi) => {
      if (disposed || !containerRef.current) return
      api = loadedApi
      try {
        widgetId = loadedApi.render(containerRef.current, {
          sitekey: props.siteKey,
          callback: (token) => {
            if (disposed) return
            setPhase('widget')
            onVerifyRef.current(token)
          },
          'error-callback': reportError,
          'expired-callback': () => {
            if (disposed) return
            onExpireRef.current?.()
            setAttempt((current) => current + 1)
          },
          'timeout-callback': () => {
            if (disposed) return
            onExpireRef.current?.()
            setAttempt((current) => current + 1)
          },
          retry: 'auto',
          'retry-interval': 8_000,
          'refresh-expired': 'auto',
          'refresh-timeout': 'auto',
        })
        if (!widgetId) {
          reportError()
          return
        }
        setPhase('widget')
      } catch {
        reportError()
      }
    }, reportError)

    return () => {
      disposed = true
      if (!api || !widgetId) return
      try {
        api.remove(widgetId)
      } catch {
        /* The external widget may already have removed itself. */
      }
    }
  }, [attempt, props.siteKey])

  const retry = () => {
    if (!window.turnstile) {
      document.querySelector(`#${turnstileScriptId}`)?.remove()
    }
    setAttempt((current) => current + 1)
  }

  return (
    <>
      <div ref={containerRef} className={props.className} />
      {phase === 'loading' && (
        <div
          role='status'
          aria-live='polite'
          className='text-muted-foreground mt-2 text-sm'
        >
          {t('Please wait a moment, human check is initializing...')}
        </div>
      )}
      {phase === 'error' && (
        <div
          role='alert'
          className='border-destructive/30 bg-destructive/5 mt-2 flex items-center justify-between gap-3 rounded-md border px-3 py-2'
        >
          <span className='text-destructive text-sm'>
            {t('Human verification failed to load. Please retry.')}
          </span>
          <Button type='button' variant='outline' size='sm' onClick={retry}>
            {t('Retry')}
          </Button>
        </div>
      )}
    </>
  )
}
