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
import { ArrowRight, Check, ImageIcon, Sparkles } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { AnimateInView } from '@/components/animate-in-view'

import { ImageGenerationCarousel } from './image-generation-carousel'

const IMAGE_GENERATION_FEATURES = [
  'Generate and edit images with natural-language instructions',
  'Create high-resolution assets for campaigns, products, and prototypes',
  'Use one UniRouters key for text, multimodal, and image models',
] as const

export function ImageGenerationShowcase() {
  const { t } = useTranslation()

  return (
    <section
      id='image-generation'
      aria-labelledby='image-generation-title'
      className='home-grid relative scroll-mt-20 overflow-hidden px-4 py-20 sm:px-6 sm:py-32'
    >
      <div className='pointer-events-none absolute top-20 -left-32 size-80 rounded-full bg-blue-500/10 blur-3xl' />
      <div className='pointer-events-none absolute right-0 bottom-0 size-96 rounded-full bg-violet-500/10 blur-3xl' />
      <div className='relative mx-auto max-w-7xl'>
        <AnimateInView className='mb-12 max-w-3xl'>
          <span className='inline-flex items-center gap-2 rounded-full border border-blue-500/20 bg-blue-500/10 px-3 py-1.5 text-xs font-semibold tracking-wider text-blue-500 uppercase dark:text-blue-300'>
            <ImageIcon className='size-4' aria-hidden='true' />
            {t('GPT IMAGE 2 · VISUAL CREATION')}
          </span>
          <h2
            id='image-generation-title'
            className='text-foreground mt-5 text-3xl font-bold tracking-tight sm:text-4xl lg:text-5xl'
          >
            {t('Turn imagination into high-resolution images')}
          </h2>
          <p className='text-muted-foreground mt-5 max-w-2xl text-base leading-7 sm:text-lg'>
            {t(
              'Generate, edit, and iterate on production-ready visuals with GPT Image 2—through the same UniRouters API key you already use for text and reasoning models.'
            )}
          </p>
        </AnimateInView>

        <div className='grid items-center gap-12 lg:grid-cols-[minmax(0,1.2fr)_minmax(340px,0.8fr)] lg:gap-16'>
          <AnimateInView>
            <ImageGenerationCarousel />
          </AnimateInView>

          <AnimateInView delay={140}>
            <div className='max-w-xl'>
              <div className='mb-5 flex size-12 items-center justify-center rounded-2xl bg-gradient-to-br from-blue-500 to-violet-500 text-white shadow-lg shadow-blue-500/20'>
                <Sparkles className='size-6' aria-hidden='true' />
              </div>
              <h3 className='text-foreground text-2xl font-bold tracking-tight sm:text-3xl'>
                {t('From a prompt to a polished visual')}
              </h3>
              <p className='text-muted-foreground mt-4 leading-7'>
                {t(
                  'Create cinematic concepts, campaign artwork, product mockups, and precise image edits without changing providers or rebuilding your workflow.'
                )}
              </p>

              <ul className='mt-7 space-y-4'>
                {IMAGE_GENERATION_FEATURES.map((feature) => (
                  <li
                    key={feature}
                    className='text-foreground flex items-start gap-3 text-sm leading-6'
                  >
                    <span className='mt-0.5 flex size-5 shrink-0 items-center justify-center rounded-full bg-emerald-500/12 text-emerald-500'>
                      <Check className='size-3.5' aria-hidden='true' />
                    </span>
                    {t(feature)}
                  </li>
                ))}
              </ul>

              <div className='mt-8 flex flex-col items-start gap-3 sm:flex-row sm:items-center'>
                <Link
                  to='/pricing/$modelId'
                  params={{ modelId: 'gpt-image-2' }}
                  className='home-primary-button group inline-flex h-11 items-center justify-center gap-2 rounded-xl px-5 text-sm font-semibold text-white'
                >
                  {t('Explore GPT Image 2')}
                  <ArrowRight
                    className='size-4 transition-transform group-hover:translate-x-1'
                    aria-hidden='true'
                  />
                </Link>
                <span className='text-muted-foreground text-xs'>
                  {t('Five original concepts generated for UniRouters')}
                </span>
              </div>
            </div>
          </AnimateInView>
        </div>
      </div>
    </section>
  )
}
