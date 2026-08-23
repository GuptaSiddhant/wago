import { useState } from 'react'
import { Link, Outlet } from '@tanstack/react-router'
import { BarChart3, FileText, LayoutDashboard, Megaphone, MessageSquare, Users, Settings, LogOut, Plus, Menu } from 'lucide-react'
import { OrgSwitcher } from './OrgSwitcher'
import { Avatar } from './ui/Avatar'
import { Button } from './ui/Button'
import { NotificationBell } from './NotificationBell'
import { NewOrgDialog } from './NewOrgDialog'
import { OrgAvatar } from './OrgAvatar'
import { useSession } from '../lib/session'
import { CallsProvider } from '../features/calls/CallsProvider'
import { CallOverlay } from '../features/calls/CallOverlay'

const mainNav = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/inbox', label: 'Inbox', icon: MessageSquare },
  { to: '/contacts', label: 'Contacts', icon: Users },
  { to: '/broadcast', label: 'Broadcast', icon: Megaphone },
  { to: '/templates', label: 'Templates', icon: FileText },
]

const settingsNav = [
  { to: '/settings/org', label: 'Organization' },
  { to: '/settings/team', label: 'Team' },
  { to: '/settings/numbers', label: 'WhatsApp Numbers' },
  { to: '/analytics', label: 'Analytics', icon: BarChart3 },
]

const adminNav = [{ to: '/settings/config', label: 'Instance Config' }]

function NavLink({
  to,
  label,
  icon: Icon,
  onNavigate,
}: {
  to: string
  label: string
  icon?: typeof MessageSquare
  onNavigate?: () => void
}) {
  return (
    <Link
      to={to}
      activeOptions={{ exact: to === '/inbox' }}
      onClick={onNavigate}
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

export function AppShell() {
  const { session, org, logout } = useSession()
  const [showNewOrg, setShowNewOrg] = useState(false)
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const isAdmin = session?.is_admin === true

  function closeSidebar() {
    setSidebarOpen(false)
  }

  return (
    <div className="flex h-screen overflow-hidden bg-zinc-950 text-zinc-100">
      {/* Skip to main content link for keyboard users */}
      <a
        href="#main-content"
        className="sr-only focus:not-sr-only fixed top-4 left-4 z-50 px-4 py-2 rounded-lg bg-emerald-600 text-white text-sm font-medium z-50 focus:outline-none focus:ring-2 focus:ring-emerald-500/60"
      >
        Skip to main content
      </a>
      {/* Mobile backdrop behind the drawer sidebar */}
      {sidebarOpen ? (
        <div
          className="fixed inset-0 z-30 bg-black/60 backdrop-blur-sm md:hidden"
          onClick={closeSidebar}
          aria-hidden
        />
      ) : null}

      <aside
        className={`fixed inset-y-0 left-0 z-40 flex w-64 shrink-0 flex-col border-r border-zinc-800/80 bg-zinc-950 transition-transform duration-200 ease-out md:static md:translate-x-0 ${
          sidebarOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
        role="navigation"
        aria-label="Main navigation"
      >
        <div className="flex items-center gap-2.5 px-5 pb-4 pt-5">
          <Link
            to="/"
            aria-label="Dashboard"
            onClick={closeSidebar}
            className="flex items-center gap-2.5 rounded-lg outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
          >
            {org ? (
              <OrgAvatar org={org} size={32} />
            ) : (
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-600 text-sm font-bold text-white">
                W
              </div>
            )}
            <div className="min-w-0 flex-1">
              <div className="text-sm font-semibold tracking-tight">WaGo</div>
              <div className="truncate text-[11px] text-zinc-500">{org?.name ?? 'Support inbox'}</div>
            </div>
          </Link>
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
            <NavLink
              key={item.to}
              to={item.to}
              label={item.label}
              icon={item.icon}
              onNavigate={closeSidebar}
            />
          ))}

          <div className="mt-5 flex items-center gap-2 px-3 pb-1 pt-2 text-[11px] font-semibold uppercase tracking-wider text-zinc-600">
            <Settings size={12} />
            Settings
          </div>
          {settingsNav.map((item) => (
            <NavLink
              key={item.to}
              to={item.to}
              label={item.label}
              onNavigate={closeSidebar}
            />
          ))}
          {isAdmin
            ? adminNav.map((item) => (
                <NavLink
                  key={item.to}
                  to={item.to}
                  label={item.label}
                  onNavigate={closeSidebar}
                />
              ))
            : null}
        </nav>

        <div className="border-t border-zinc-800/80 p-3">
          <div className="flex items-center gap-2.5 rounded-lg px-2 py-1.5">
            <Link
              to="/account"
              onClick={closeSidebar}
              aria-label="Your account"
              title="Your account"
              className="flex min-w-0 flex-1 items-center gap-2.5 rounded-lg outline-none transition hover:bg-zinc-900 focus-visible:ring-2 focus-visible:ring-emerald-500/60"
            >
              <Avatar name={session?.user.name} size={32} />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm text-zinc-200">
                  {session?.user.name || session?.user.email}
                </span>
                <span className="block truncate text-[11px] text-zinc-500">{session?.user.email}</span>
              </span>
            </Link>
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

      <main
        id="main-content"
        className="flex min-w-0 flex-1 flex-col"
        role="main"
      >
        {/* Mobile top bar with hamburger menu */}
        <div className="flex items-center gap-2 border-b border-zinc-800/80 px-3 py-2.5 md:hidden">
          <Button
            size="icon"
            variant="ghost"
            aria-label="Open navigation"
            onPress={() => setSidebarOpen(true)}
          >
            <Menu size={18} className="text-zinc-400" />
          </Button>
          <div className="min-w-0 flex-1 truncate text-sm font-semibold text-zinc-100">
            {org?.name ?? 'WaGo'}
          </div>
        </div>

        <CallsProvider>
          <Outlet />
          <CallOverlay />
        </CallsProvider>
      </main>

      {showNewOrg ? <NewOrgDialog onClose={() => setShowNewOrg(false)} /> : null}
    </div>
  )
}
