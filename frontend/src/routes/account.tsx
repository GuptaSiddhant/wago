import { createFileRoute, redirect } from '@tanstack/react-router'
import { AccountPage } from '../features/auth/AccountPage'
import { getStoredSession } from '../lib/authStore'

// Personal settings are the one authenticated area that does NOT require an
// organization — this route deliberately sits outside the org-gated _app layout.
export const Route = createFileRoute('/account')({
  beforeLoad: () => {
    if (!getStoredSession()) {
      throw redirect({ to: '/login' })
    }
  },
  component: AccountPage,
})
