import { createFileRoute, redirect } from '@tanstack/react-router'
import { SelectOrgPage } from '../features/auth/SelectOrgPage'
import { getStoredSession } from '../lib/authStore'

export const Route = createFileRoute('/select-org')({
  staticData: { title: 'Choose organization' },
  beforeLoad: () => {
    if (!getStoredSession()) {
      throw redirect({ to: '/login' })
    }
  },
  component: SelectOrgPage,
})
