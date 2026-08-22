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
import { useEffect, useLayoutEffect, useRef, useState } from 'react'

import { DEFAULT_LOGO } from '@/lib/constants'

const BRAND_ENTRANCE_STORAGE_KEY = 'unirouters:brand-entrance:v1'
const BRAND_ENTRANCE_DURATION_MS = 1800

function hasPlayedBrandEntrance(): boolean {
  try {
    return window.sessionStorage.getItem(BRAND_ENTRANCE_STORAGE_KEY) === '1'
  } catch {
    return false
  }
}

function markBrandEntrancePlayed(): void {
  try {
    window.sessionStorage.setItem(BRAND_ENTRANCE_STORAGE_KEY, '1')
  } catch {
    // The animation can still run when storage is unavailable.
  }
}

function shouldShowBrandEntrance(): boolean {
  if (typeof window === 'undefined' || hasPlayedBrandEntrance()) {
    return false
  }

  return !window.matchMedia?.('(prefers-reduced-motion: reduce)').matches
}

/**
 * Plays the public-home brand lockup once per tab, then lands it on the
 * responsive header logo using the target's live DOM coordinates.
 */
export function BrandEntrance() {
  const [visible, setVisible] = useState(shouldShowBrandEntrance)
  const overlayRef = useRef<HTMLDivElement>(null)
  const moverRef = useRef<HTMLDivElement>(null)
  const logoRef = useRef<HTMLImageElement>(null)

  useEffect(() => {
    markBrandEntrancePlayed()
  }, [])

  useLayoutEffect(() => {
    if (!visible) return

    const overlay = overlayRef.current
    const mover = moverRef.current
    const sourceLogo = logoRef.current
    const targetLogo = document.querySelector<HTMLElement>(
      '[data-slot="public-home-brand-logo"]'
    )

    if (!overlay || !mover || !sourceLogo || !targetLogo) {
      setVisible(false)
      return
    }

    const sourceLogoRect = sourceLogo.getBoundingClientRect()
    const targetLogoRect = targetLogo.getBoundingClientRect()

    if (sourceLogoRect.width <= 0 || targetLogoRect.width <= 0) {
      setVisible(false)
      return
    }

    if (typeof mover.animate !== 'function') {
      setVisible(false)
      return
    }

    const destinationScale = targetLogoRect.width / sourceLogoRect.width
    const destinationX = targetLogoRect.left - sourceLogoRect.left
    const destinationY =
      targetLogoRect.top +
      targetLogoRect.height / 2 -
      (sourceLogoRect.top + sourceLogoRect.height / 2)
    const destinationTransform = `translate3d(${destinationX}px, ${destinationY}px, 0) scale(${destinationScale})`
    const animationOptions: KeyframeAnimationOptions = {
      duration: BRAND_ENTRANCE_DURATION_MS,
      fill: 'forwards',
    }

    const moverAnimation = mover.animate(
      [
        { offset: 0, transform: 'translate3d(0, 0, 0) scale(1)' },
        {
          offset: 0.5,
          transform: 'translate3d(0, 0, 0) scale(1)',
          easing: 'cubic-bezier(0.22, 1, 0.36, 1)',
        },
        {
          offset: 1,
          transform: destinationTransform,
          easing: 'cubic-bezier(0.16, 1, 0.3, 1)',
        },
      ],
      animationOptions
    )
    const overlayAnimation = overlay.animate(
      [
        { offset: 0, opacity: 1 },
        { offset: 0.78, opacity: 1 },
        { offset: 1, opacity: 0, easing: 'ease-out' },
      ],
      animationOptions
    )
    const fallbackTimeout = window.setTimeout(
      () => setVisible(false),
      BRAND_ENTRANCE_DURATION_MS + 100
    )
    let cancelled = false

    void Promise.all([moverAnimation.finished, overlayAnimation.finished])
      .then(() => {
        if (!cancelled) setVisible(false)
      })
      .catch(() => {
        // Cancelling animations during unmount is expected.
      })

    return () => {
      cancelled = true
      window.clearTimeout(fallbackTimeout)
      moverAnimation.cancel()
      overlayAnimation.cancel()
    }
  }, [visible])

  if (!visible) return null

  return (
    <div
      ref={overlayRef}
      data-slot='brand-entrance'
      aria-hidden='true'
      className='brand-entrance-screen pointer-events-none fixed inset-0 z-[70] flex items-center justify-center overflow-hidden'
    >
      <div className='brand-entrance-grid absolute inset-0' />
      <div className='brand-entrance-scan absolute inset-x-0 top-0 h-px' />
      <div className='brand-entrance-orbit brand-entrance-orbit-outer absolute' />
      <div className='brand-entrance-orbit brand-entrance-orbit-inner absolute' />

      <div
        ref={moverRef}
        className='brand-entrance-mover relative z-10 will-change-transform'
      >
        <div className='brand-entrance-lockup flex items-center'>
          <img
            ref={logoRef}
            src={DEFAULT_LOGO}
            alt=''
            className='brand-entrance-logo shrink-0 object-cover'
          />
          <span className='brand-entrance-wordmark font-semibold tracking-tight whitespace-nowrap'>
            uniRouters
          </span>
        </div>
      </div>
    </div>
  )
}
