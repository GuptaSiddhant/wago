import { QueryClientProvider } from '@tanstack/react-query'
import { createRootRoute, Outlet } from '@tanstack/react-router'
import { queryClient } from '../api/queryClient'
import { SessionProvider } from '../lib/session'
import { NotificationsProvider } from '../lib/notifications'

export const Route = createRootRoute({
  component: RootComponent,
})

function RootComponent() {
  return (
    <QueryClientProvider client={queryClient}>
      <SessionProvider>
        <NotificationsProvider>
          <Outlet />
        </NotificationsProvider>
      </SessionProvider>
    </QueryClientProvider>
  )
}
