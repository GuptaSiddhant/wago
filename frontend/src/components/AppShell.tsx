import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Link, Outlet } from '@tanstack/react-router'
import { BarChart3, FileText, Megaphone, MessageSquare, Users, Settings, LogOut, Plus } from 'lucide-react'
import { createOrg } from '../api/client'
import { OrgSwitcher } from './OrgSwitcher'
import { Avatar } from './ui/Avatar'
import { Button } from './ui/Button'
import { FormError } from './ui/FormError'
import { ModalDialog } from './ui/Modal'
import { TextField } from './ui/TextField'
import { NotificationBell } from './NotificationBell'
import { useSession } from '../lib/session'
import { CallsProvider } from '../features/calls/CallsProvider'
import { CallOverlay } from '../features/calls/CallOverlay'

const mainNav = [
  { to: '/inbox', label: 'Inbox', icon: MessageSquare },
  { to: '/contacts', label: 'Contacts', icon: Users },
  { to: '/broadcast', label: 'Broadcast', icon: Megaphone },
  { to: '/templates', label: 'Templates', icon: FileText },
]

const settingsNav = [
  { to: '/settings/team', label: 'Team' },
  { to: '/settings/numbers', label: 'WhatsApp Numbers' },
  { to: '/analytics', label: 'Analytics', icon: BarChart3 },
]

function NavLink({
  to,
  label,
  icon: Icon,
}: {
  to: string
  label: string
  icon?: typeof MessageSquare
}) {
  return (
    <Link
      to={to}
      activeOptions={{ exact: to === '/inbox' }}
      className="group flex items-center gap-3 rounded-lg px-3 py-2 text-sm text-zinc-400 transition hover:bg-zinc-900 hover:text-zinc-100"
      activeProps={{
        className:
          'bg-zinc-900 text-zinc-100 font-medium [&>svg]:text-emerald-500',
      }}
    >
      {Icon ? <Icon size={17} className="text-zinc-500 transition group-hover:text-zinc-300" /> : null}
      {label}
    </Link>
  )
}

function NewOrgDialog({ onClose }: { onClose: () => void }) {
  const { refresh, selectOrg } = useSession()
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () => createOrg(name.trim()),
    onSuccess: async (org) => {
      // Refresh memberships so the new org appears in the switcher, then jump
      // into it so it becomes the active org on first use.
      await refresh()
      selectOrg(org.id)
      onClose()
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to create organization')
    },
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) return setError('Organization name is required')
    setError(null)
    mutation.mutate()
  }

  return (
    <ModalDialog isOpen onOpenChange={(open) => !open && onClose()} title="New organization">
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField
          label="Organization name"
          value={name}
          onChange={setName}
          placeholder="Acme Inc."
          isRequired
          autoFocus
        />
        <FormError message={error} />
        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" variant="ghost" onPress={onClose}>
            Cancel
          </Button>
          <Button type="submit" isDisabled={mutation.isPending}>
            {mutation.isPending ? 'Creating…' : 'Create organization'}
          </Button>
        </div>
      </form>
    </ModalDialog>
  )
}

export function AppShell() {
  const { session, org, logout } = useSession()
  const [showNewOrg, setShowNewOrg] = useState(false)
  const isAdmin = session?.is_admin === true

  return (
    <div className="flex h-screen overflow-hidden bg-zinc-950 text-zinc-100">
      <aside className="flex w-64 shrink-0 flex-col border-r border-zinc-800/80 bg-zinc-950">
        <div className="flex items-center gap-2.5 px-5 pb-4 pt-5">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-600 text-sm font-bold text-white">
            W
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold tracking-tight">WaGo</div>
            <div className="truncate text-[11px] text-zinc-500">{org?.name ?? 'Support inbox'}</div>
          </div>
          <NotificationBell />
        </div>

        <div className="flex items-center gap-2 px-3">
          <div className="min-w-0 flex-1">
            <OrgSwitcher />
          </div>
          {isAdmin ? (
            <Button
              size="icon"
              variant="ghost"
              aria-label="New organization"
              onPress={() => setShowNewOrg(true)}
            >
              <Plus size={16} className="text-zinc-500 hover:text-emerald-400" />
            </Button>
          ) : null}
        </div>

        <nav className="mt-4 flex-1 space-y-0.5 overflow-y-auto px-3">
          {mainNav.map((item) => (
            <NavLink key={item.to} to={item.to} label={item.label} icon={item.icon} />
          ))}

          <div className="mt-5 flex items-center gap-2 px-3 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wider text-zinc-600">
            <Settings size={12} />
            Settings
          </div>
          {settingsNav.map((item) => (
            <NavLink key={item.to} to={item.to} label={item.label} />
          ))}
        </nav>

        <div className="border-t border-zinc-800/80 p-3">
          <div className="flex items-center gap-2.5 rounded-lg px-2 py-1.5">
            <Avatar name={session?.user.name} size={32} />
            <div className="min-w-0 flex-1">
              <div className="truncate text-sm text-zinc-200">
                {session?.user.name || session?.user.email}
              </div>
              <div className="truncate text-[11px] text-zinc-500">{session?.user.email}</div>
            </div>
            <button
              type="button"
              onClick={logout}
              aria-label="Log out"
              title="Log out"
              className="rounded-lg p-2 text-zinc-500 transition hover:bg-zinc-900 hover:text-red-400"
            >
              <LogOut size={16} />
            </button>
          </div>
        </div>
      </aside>

      <main className="flex min-w-0 flex-1 flex-col">
        <CallsProvider>
          <Outlet />
          <CallOverlay />
        </CallsProvider>
      </main>

      {showNewOrg ? <NewOrgDialog onClose={() => setShowNewOrg(false)} /> : null}
    </div>
  )
}
