import { useEffect, useRef } from 'react'
import { Bell, Check } from 'lucide-react'
import { useNavigate } from '@tanstack/react-router'
import { useNotifications } from '../lib/notifications'
import { useSession } from '../lib/session'

export function NotificationBell() {
  const { session } = useSession()
  const { items, unread, open, setOpen, markAllRead } = useNotifications()
  const navigate = useNavigate()
  const panelRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    // Marking read immediately clears the badge; the center keeps the list for context.
    void markAllRead()
  }, [open, markAllRead])

  useEffect(() => {
    if (!open) return
    function onPointerDown(event: MouseEvent) {
      if (panelRef.current && !panelRef.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', onPointerDown)
    return () => document.removeEventListener('mousedown', onPointerDown)
  }, [open, setOpen])

  if (!session) return null

  return (
    <div ref={panelRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen(!open)}
        aria-label="Notifications"
        className="relative rounded-lg p-2 text-zinc-400 transition hover:bg-zinc-900 hover:text-zinc-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
      >
        <Bell size={18} />
        {unread > 0 ? (
          <span className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-emerald-500 px-1 text-[10px] font-bold text-zinc-950">
            {unread > 99 ? '99+' : unread}
          </span>
        ) : null}
      </button>

      {open ? (
        <div className="absolute right-0 top-10 z-50 w-80 overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900 shadow-2xl">
          <div className="flex items-center justify-between border-b border-zinc-800 px-4 py-2.5">
            <span className="text-sm font-semibold">Notifications</span>
            {unread > 0 ? (
              <button
                type="button"
                onClick={() => void markAllRead()}
                className="flex items-center gap-1 text-[11px] text-emerald-400 hover:text-emerald-300"
              >
                <Check size={12} />
                Mark all read
              </button>
            ) : null}
          </div>
          <div className="max-h-96 overflow-y-auto">
            {items.length === 0 ? (
              <div className="px-4 py-10 text-center text-sm text-zinc-500">No notifications yet</div>
            ) : (
              items.map((n) => (
                <button
                  key={n.id}
                  type="button"
                  onClick={() => {
                    setOpen(false)
                    void navigate({ to: '/inbox', search: { conv: n.conversation_id } })
                  }}
                  className="flex w-full flex-col gap-0.5 border-b border-zinc-800/60 px-4 py-3 text-left transition hover:bg-zinc-800/60"
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-sm font-medium text-zinc-100">
                      {n.contact_name || 'customer'}
                    </span>
                    {!n.read ? (
                      <span className="h-2 w-2 shrink-0 rounded-full bg-emerald-500" />
                    ) : null}
                  </div>
                  <span className="line-clamp-2 text-xs text-zinc-400">{n.body}</span>
                </button>
              ))
            )}
          </div>
        </div>
      ) : null}
    </div>
  )
}
