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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { Window } from 'happy-dom'
import { createInstance } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider } from 'react-i18next'

import englishMessages from '../../../../../i18n/locales/en.json'
import { UserAuthForm } from '../user-auth-form'

describe('sign-in Turnstile submission guard', () => {
  test('disables password sign-in until the required challenge has a token', async () => {
    const i18n = createInstance()
    await i18n.init({
      lng: 'en',
      fallbackLng: false,
      resources: { en: englishMessages },
      interpolation: { escapeValue: false },
    })

    const queryClient = new QueryClient()
    queryClient.setQueryData(['status'], {
      password_login_enabled: true,
      turnstile_check: true,
      turnstile_site_key: 'test-site-key',
    })

    const rootRoute = createRootRoute({ component: UserAuthForm })
    const forgotPasswordRoute = createRoute({
      getParentRoute: () => rootRoute,
      path: '/forgot-password',
      component: () => null,
    })
    const router = createRouter({
      routeTree: rootRoute.addChildren([forgotPasswordRoute]),
      history: createMemoryHistory({ initialEntries: ['/'] }),
    })
    await router.load()

    const markup = renderToStaticMarkup(
      <QueryClientProvider client={queryClient}>
        <I18nextProvider i18n={i18n}>
          <RouterProvider router={router} />
        </I18nextProvider>
      </QueryClientProvider>
    )
    const domWindow = new Window()
    domWindow.document.body.innerHTML = markup

    const submitButton = [
      ...domWindow.document.querySelectorAll('button[type="submit"]'),
    ].find((button) => button.textContent?.includes('Sign in'))

    assert.ok(submitButton)
    assert.notEqual(submitButton.getAttribute('disabled'), null)

    domWindow.close()
    queryClient.clear()
  })
})
