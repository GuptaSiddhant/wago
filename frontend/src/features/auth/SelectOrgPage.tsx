import { useNavigate } from '@tanstack/react-router'
import { ArrowRight, Building2, UserPlus } from 'lucide-react'
import type { OrgSummary } from '../../api/types'
import { useSession } from '../../lib/session'
import { OrgAvatar } from '../../components/OrgAvatar'
import { OrgCreateForm } from '../../components/OrgCreateForm'
import { Button } from '../../components/ui/Button'

/**
 * Mandatory org picker shown when a user has no valid active organization.
 * - >0 orgs: pick one to continue
 * - superadmin with none: inline create-org form (first-run onboarding)
 * - regular user with none: invite hint; nothing else to do here
 */
export function SelectOrgPage() {
  const { session, selectOrg, logout } = useSession()
  const navigate = useNavigate()

  if (!session) return null

  const orgs = session.orgs ?? []

  async function choose(orgId: string) {
    selectOrg(orgId)
    await navigate({ to: '/inbox' })
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950 px-4">
      <div className="w-full max-w-md">
        <div className="mb-8 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-600 text-lg font-bold text-white">
              W
            </div>
            <div>
              <h1 className="text-xl font-semibold tracking-tight text-zinc-100">WaGo</h1>
              <p className="text-xs text-zinc-500">{session.user.email}</p>
            </div>
          </div>
          <Button variant="ghost" size="sm" onPress={logout}>
            Log out
          </Button>
        </div>

        {orgs.length > 0 ? (
          <>
            <h2 className="mb-1 text-sm font-medium text-zinc-100">
              Choose an organization
            </h2>
            <p className="mb-4 text-xs text-zinc-500">
              Pick the workspace you want to work in.
            </p>
            <ul className="flex flex-col gap-2">
              {orgs.map((o: OrgSummary) => (
                <li key={o.id}>
                  <button
                    type="button"
                    onClick={() => choose(o.id)}
                    className="group flex w-full items-center gap-3 rounded-xl border border-zinc-800 bg-zinc-900/60 p-3 text-left transition hover:border-emerald-600/60 hover:bg-zinc-900"
                  >
                    <OrgAvatar org={o} size={36} />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-sm font-medium text-zinc-100">
                        {o.name}
                      </span>
                      <span className="block text-[11px] uppercase tracking-wide text-zinc-500">
                        {o.role}
                      </span>
                    </span>
                    <ArrowRight
                      size={16}
                      className="shrink-0 text-zinc-600 transition group-hover:text-emerald-400"
                    />
                  </button>
                </li>
              ))}
            </ul>
          </>
        ) : session.is_admin ? (
          <>
            <h2 className="mb-1 text-sm font-medium text-zinc-100">
              Set up your organization
            </h2>
            <p className="mb-4 text-xs text-zinc-500">
              Create your workspace to start using WaGo. You can change these
              details later.
            </p>
            <div className="rounded-2xl border border-zinc-800 bg-zinc-900/60 p-5 shadow-xl shadow-black/30">
              <OrgCreateForm
                submitLabel="Create & continue"
                onCreated={() => navigate({ to: '/inbox' })}
              />
            </div>
          </>
        ) : (
          <div className="rounded-2xl border border-zinc-800 bg-zinc-900/60 p-6 text-center shadow-xl shadow-black/30">
            <Building2 size={24} className="mx-auto mb-3 text-zinc-600" />
            <p className="text-sm text-zinc-300">No organization yet</p>
            <p className="mt-1 text-xs text-zinc-500">
              You're not part of an organization. Ask your administrator to send
              you an invite link.
            </p>
            <Button variant="ghost" size="sm" className="mt-4" onPress={logout}>
              <UserPlus size={14} />
              Sign out
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}
