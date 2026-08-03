import { useQuery } from '@tanstack/react-query'
import { Smartphone } from 'lucide-react'
import { listAccounts } from '../../api/client'
import { useSession } from '../../lib/session'
import { Badge } from '../../components/ui/Badge'
import { EmptyState } from '../../components/ui/EmptyState'
import { Spinner } from '../../components/ui/Spinner'

export function NumbersPage() {
  const { org } = useSession()
  const accountsQuery = useQuery({
    queryKey: ['accounts', org?.id ?? ''],
    queryFn: listAccounts,
    enabled: org?.id != null,
  })

  const accounts = accountsQuery.data?.items ?? []

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="border-b border-zinc-800/80 px-6 py-4">
        <h1 className="text-lg font-semibold text-zinc-100">WhatsApp Numbers</h1>
        <p className="text-sm text-zinc-500">
          Meta accounts connected to this organization. Incoming messages are routed by
          phone_number_id.
        </p>
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        {accountsQuery.isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : accounts.length === 0 ? (
          <EmptyState
            icon={<Smartphone size={32} />}
            title="No WhatsApp numbers connected"
            description="Connect a Meta WhatsApp Business account to start receiving messages."
          />
        ) : (
          <ul className="grid gap-3 sm:grid-cols-2">
            {accounts.map((a) => (
              <li
                key={a.id}
                className="flex items-start gap-3 rounded-xl border border-zinc-800 bg-zinc-900/50 p-4"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400">
                  <Smartphone size={17} />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-sm font-medium text-zinc-100">
                      {a.display_name || a.phone_number_id}
                    </span>
                    <Badge tone={a.status === 'connected' ? 'green' : 'red'}>
                      {a.status}
                    </Badge>
                  </div>
                  <p className="mt-0.5 truncate text-xs text-zinc-500">
                    phone_number_id: {a.phone_number_id}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
