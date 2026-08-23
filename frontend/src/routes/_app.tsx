import { createFileRoute, redirect } from '@tanstack/react-router'
import { AppShell } from '../components/AppShell'
import { getStoredOrgId, getStoredSession, setStoredOrgId } from '../lib/authStore'

export const Route = createFileRoute('/_app')({
  beforeLoad: () => {
    const session = getStoredSession()
    if (!session) {
      throw redirect({ to: '/login' })
    }
    // Every page under this layout is org-scoped: a valid selected org is
    // mandatory. The stored id must still be one of the user's memberships
    // so stale selections (removed from org, org deleted) can't slip through.
    const orgId = getStoredOrgId()
    if (!orgId || !session.orgs.some((o) => o.id === orgId)) {
      // Single-org users get silently healed instead of bounced.
      if (session.orgs.length === 1) {
        setStoredOrgId(session.orgs[0].id)
        return
      }
      throw redirect({ to: '/select-org' })
    }
  },
  component: AppShell,
})
