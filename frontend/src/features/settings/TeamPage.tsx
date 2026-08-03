import { useQuery } from '@tanstack/react-query'
import { Users } from 'lucide-react'
import { listTeam } from '../../api/client'
import { useSession } from '../../lib/session'
import { Avatar } from '../../components/ui/Avatar'
import { Badge } from '../../components/ui/Badge'
import { EmptyState } from '../../components/ui/EmptyState'
import { Spinner } from '../../components/ui/Spinner'

function roleTone(role: string): 'amber' | 'blue' | 'green' | 'zinc' {
  switch (role) {
    case 'owner':
      return 'amber'
    case 'admin':
      return 'blue'
    case 'agent':
      return 'green'
    default:
      return 'zinc'
  }
}

export function TeamPage() {
  const { session, org } = useSession()
  const meId = session?.user.id

  const teamQuery = useQuery({
    queryKey: ['team', org?.id ?? ''],
    queryFn: listTeam,
    enabled: org?.id != null,
  })

  const members = teamQuery.data?.items ?? []

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="border-b border-zinc-800/80 px-6 py-4">
        <h1 className="text-lg font-semibold text-zinc-100">Team</h1>
        <p className="text-sm text-zinc-500">Members of this organization.</p>
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        {teamQuery.isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : members.length === 0 ? (
          <EmptyState
            icon={<Users size={32} />}
            title="No team members"
            description="Add members to share this inbox."
          />
        ) : (
          <table className="w-full border-collapse text-left text-sm">
            <thead>
              <tr className="border-b border-zinc-800 text-xs uppercase tracking-wider text-zinc-500">
                <th className="py-2.5 pr-4 font-medium">Member</th>
                <th className="py-2.5 pr-4 font-medium">Email</th>
                <th className="py-2.5 font-medium">Role</th>
              </tr>
            </thead>
            <tbody>
              {members.map((m) => (
                <tr key={m.id} className="border-b border-zinc-800/60 transition hover:bg-zinc-900/40">
                  <td className="py-3 pr-4">
                    <div className="flex items-center gap-3">
                      <Avatar name={m.name} size={32} />
                      <span className="font-medium text-zinc-100">{m.name}</span>
                      {m.id === meId ? (
                        <Badge tone="green">You</Badge>
                      ) : null}
                    </div>
                  </td>
                  <td className="py-3 pr-4 text-zinc-400">{m.email}</td>
                  <td className="py-3">
                    <Badge tone={roleTone(m.role)}>{m.role}</Badge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
