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
import { ArrowLeft, ArrowRight, Sparkles } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import aiChipSummitImage from '@/assets/images/home-image-generation/ai-chip-summit.webp'
import marsCommonsImage from '@/assets/images/home-image-generation/mars-commons.webp'
import nightMarketLeadersImage from '@/assets/images/home-image-generation/night-market-leaders.webp'
import orbitalAiNetworkImage from '@/assets/images/home-image-generation/orbital-ai-network.webp'
import robotChipFoundryImage from '@/assets/images/home-image-generation/robot-chip-foundry.webp'

const IMAGE_GENERATION_SLIDES = [
  {
    src: nightMarketLeadersImage,
    title: 'Night-market crossover',
    alt: 'AI-generated fictional editorial illustration of Elon Musk, Tim Cook, and Lei Jun sharing food and beer at a Chinese night market',
  },
  {
    src: marsCommonsImage,
    title: 'A shared future on Mars',
    alt: 'AI-generated fictional illustration of a cooperative human settlement on Mars',
  },
  {
    src: aiChipSummitImage,
    title: 'Intelligence meets industry',
    alt: 'AI-generated fictional editorial illustration of Donald Trump at an AI robotics and semiconductor summit',
  },
  {
    src: robotChipFoundryImage,
    title: 'The new chip foundry',
    alt: 'AI-generated illustration of humanoid robots assembling advanced AI chips beneath a luminous industrial sky',
  },
  {
    src: orbitalAiNetworkImage,
    title: 'Compute beyond the horizon',
    alt: 'AI-generated illustration of an orbital computing network linking satellites, data centers, and cities at sunrise',
  },
] as const

const SLIDE_INTERVAL_MS = 5000

function getSlideIndex(currentIndex: number, offset: number) {
  const slideCount = IMAGE_GENERATION_SLIDES.length
  return (currentIndex + offset + slideCount) % slideCount
}

export function ImageGenerationCarousel() {
  const { t } = useTranslation()
  const [activeIndex, setActiveIndex] = useState(0)
  const [isPaused, setIsPaused] = useState(false)
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false)

  const showSlide = useCallback((offset: number) => {
    setActiveIndex((currentIndex) => getSlideIndex(currentIndex, offset))
  }, [])

  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    const updatePreference = () => setPrefersReducedMotion(mediaQuery.matches)

    updatePreference()
    mediaQuery.addEventListener('change', updatePreference)
    return () => mediaQuery.removeEventListener('change', updatePreference)
  }, [])

  useEffect(() => {
    if (isPaused || prefersReducedMotion) return

    const interval = window.setInterval(() => showSlide(1), SLIDE_INTERVAL_MS)
    return () => window.clearInterval(interval)
  }, [isPaused, prefersReducedMotion, showSlide])

  const activeSlide = IMAGE_GENERATION_SLIDES[activeIndex]

  return (
    <div>
      <div
        role='region'
        aria-roledescription={t('carousel')}
        aria-label={t('AI-generated image gallery')}
        tabIndex={0}
        className='group border-border focus-visible:ring-offset-background relative aspect-3/2 overflow-hidden rounded-[2rem] border bg-slate-950 shadow-2xl shadow-blue-950/20 outline-none focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-4'
        onMouseEnter={() => setIsPaused(true)}
        onMouseLeave={() => setIsPaused(false)}
        onFocusCapture={() => setIsPaused(true)}
        onBlurCapture={(event) => {
          const nextTarget = event.relatedTarget
          if (
            !(nextTarget instanceof Node) ||
            !event.currentTarget.contains(nextTarget)
          ) {
            setIsPaused(false)
          }
        }}
        onKeyDown={(event) => {
          if (event.key === 'ArrowLeft') {
            event.preventDefault()
            showSlide(-1)
          }
          if (event.key === 'ArrowRight') {
            event.preventDefault()
            showSlide(1)
          }
        }}
      >
        {IMAGE_GENERATION_SLIDES.map((slide, index) => {
          const isActive = index === activeIndex

          return (
            <figure
              key={slide.src}
              aria-hidden={!isActive}
              data-active={isActive}
              className={`absolute inset-0 transition-opacity duration-700 motion-reduce:transition-none ${
                isActive ? 'z-10 opacity-100' : 'pointer-events-none opacity-0'
              }`}
            >
              <img
                src={slide.src}
                alt={t(slide.alt)}
                width={1536}
                height={1024}
                loading={index === 0 ? 'eager' : 'lazy'}
                decoding='async'
                className='size-full object-cover'
              />
              <div className='absolute inset-0 bg-gradient-to-t from-slate-950/90 via-slate-950/5 to-slate-950/10' />
              <figcaption className='absolute right-5 bottom-5 left-5 flex items-end justify-between gap-4 text-white sm:right-7 sm:bottom-7 sm:left-7'>
                <div>
                  <span className='mb-2 inline-flex rounded-full border border-white/15 bg-black/25 px-2.5 py-1 text-[10px] font-semibold tracking-[0.16em] text-white/75 uppercase backdrop-blur-md'>
                    {t('AI-generated concept')}
                  </span>
                  <p className='max-w-md text-lg font-semibold tracking-tight sm:text-2xl'>
                    {t(slide.title)}
                  </p>
                </div>
                <span className='shrink-0 font-mono text-xs text-white/60'>
                  {String(index + 1).padStart(2, '0')} /{' '}
                  {String(IMAGE_GENERATION_SLIDES.length).padStart(2, '0')}
                </span>
              </figcaption>
            </figure>
          )
        })}

        <div className='absolute top-5 right-5 z-20 inline-flex items-center gap-2 rounded-full border border-white/15 bg-black/25 px-3 py-1.5 text-[11px] font-medium text-white/80 backdrop-blur-md'>
          <Sparkles className='size-3.5 text-cyan-300' aria-hidden='true' />
          GPT IMAGE 2
        </div>

        <button
          type='button'
          aria-label={t('Previous image')}
          className='absolute top-1/2 left-4 z-20 flex size-10 -translate-y-1/2 items-center justify-center rounded-full border border-white/15 bg-black/30 text-white opacity-100 backdrop-blur-md transition hover:scale-105 hover:bg-black/50 focus-visible:ring-2 focus-visible:ring-white focus-visible:outline-none lg:opacity-0 lg:group-focus-within:opacity-100 lg:group-hover:opacity-100'
          onClick={() => showSlide(-1)}
        >
          <ArrowLeft className='size-4' aria-hidden='true' />
        </button>
        <button
          type='button'
          aria-label={t('Next image')}
          className='absolute top-1/2 right-4 z-20 flex size-10 -translate-y-1/2 items-center justify-center rounded-full border border-white/15 bg-black/30 text-white opacity-100 backdrop-blur-md transition hover:scale-105 hover:bg-black/50 focus-visible:ring-2 focus-visible:ring-white focus-visible:outline-none lg:opacity-0 lg:group-focus-within:opacity-100 lg:group-hover:opacity-100'
          onClick={() => showSlide(1)}
        >
          <ArrowRight className='size-4' aria-hidden='true' />
        </button>
      </div>

      <div className='mt-2 flex items-center justify-center'>
        {IMAGE_GENERATION_SLIDES.map((slide, index) => {
          const isActive = index === activeIndex

          return (
            <button
              key={slide.src}
              type='button'
              aria-label={t('Go to image {{number}}', { number: index + 1 })}
              aria-current={isActive ? 'true' : undefined}
              className={`group flex h-8 items-center justify-center rounded-full focus-visible:ring-2 focus-visible:ring-blue-500 focus-visible:ring-offset-2 focus-visible:outline-none ${
                isActive ? 'w-10' : 'w-6'
              }`}
              onClick={() => setActiveIndex(index)}
            >
              <span
                aria-hidden='true'
                className={`h-2 rounded-full transition-all duration-300 motion-reduce:transition-none ${
                  isActive
                    ? 'w-8 bg-blue-500'
                    : 'bg-muted-foreground/30 group-hover:bg-muted-foreground/60 w-2'
                }`}
              />
            </button>
          )
        })}
      </div>

      <p className='sr-only' aria-live='polite'>
        {t('Image {{current}} of {{total}}: {{title}}', {
          current: activeIndex + 1,
          total: IMAGE_GENERATION_SLIDES.length,
          title: t(activeSlide.title),
        })}
      </p>
    </div>
  )
}
