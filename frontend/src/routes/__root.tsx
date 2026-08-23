import { useEffect } from 'react'
import { QueryClientProvider } from '@tanstack/react-query'
import { createRootRoute, Outlet, useRouterState } from '@tanstack/react-router'
import { queryClient } from '../api/queryClient'
import { SessionProvider } from '../lib/session'
import { NotificationsProvider } from '../lib/notifications'
import { ToastContextProvider } from '../lib/toast'
import { ConfirmProvider } from '../lib/confirm'
import { pageTitleFromMatches } from '../lib/pageTitle'

export const Route = createRootRoute({
  component: RootComponent,
})

function RootComponent() {
  const matches = useRouterState({ select: (s) => s.matches })
  const pageTitle = pageTitleFromMatches(matches)

  useEffect(() => {
    document.title = pageTitle ? `${pageTitle} · Wago` : 'Wago'
  }, [pageTitle])

  return (
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        <NotificationsProvider>
          <ToastContextProvider>
            <ConfirmProvider>
              <Outlet />
            </ConfirmProvider>
          </ToastContextProvider>
        </NotificationsProvider>
      </SessionProvider>
    </QueryClientProvider>
  )
}
