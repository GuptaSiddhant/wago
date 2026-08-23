import { createFileRoute, redirect } from '@tanstack/react-router'
import { LoginPage } from '../features/auth/LoginPage'
import { getStoredSession } from '../lib/authStore'

export const Route = createFileRoute('/login')({
  staticData: { title: 'Sign in' },
  beforeLoad: () => {
    if (getStoredSession()) {
      throw redirect({ to: '/inbox' })
    }
  },
  component: LoginPage,
})
