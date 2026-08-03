import { createFileRoute, redirect } from '@tanstack/react-router'
import { AppShell } from '../components/AppShell'
import { getStoredSession } from '../lib/authStore'

export const Route = createFileRoute('/_app')({
  beforeLoad: () => {
    if (!getStoredSession()) {
      throw redirect({ to: '/login' })
    }
  },
  component: AppShell,
})
