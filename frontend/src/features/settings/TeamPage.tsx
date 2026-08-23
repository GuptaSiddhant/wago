import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { DialogTrigger } from 'react-aria-components'
import { Pencil, Plus, Trash2, Users } from 'lucide-react'
import {
  createInvite,
  createTeam,
  deleteTeam,
  deleteTeamMember,
  listInvites,
  listTeam,
  listTeams,
  revokeInvite,
  updateTeam,
  updateTeamMember,
} from '../../api/client'
import { useSession } from '../../lib/session'
import { useConfirm } from '../../lib/confirm'
import { formatDate } from '../../lib/format'
import { Avatar } from '../../components/ui/Avatar'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { ModalDialog } from '../../components/ui/Modal'
import { SelectField } from '../../components/ui/Select'
import { Skeleton } from '../../components/ui/Skeleton'
import { TextField } from '../../components/ui/TextField'
import type { InviteDTO, TeamDTO, TeamMemberDTO } from '../../api/types'

const roleOptions = [
  { id: 'owner', label: 'Owner' },
  { id: 'admin', label: 'Admin' },
  { id: 'agent', label: 'Agent' },
  { id: 'viewer', label: 'Viewer' },
]

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

function InviteDialog({ teams }: { teams: TeamDTO[] }) {
  const { session, org } = useSession()
  const queryClient = useQueryClient()
  const [email, setEmail] = useState('')
  const [role, setRole] = useState('agent')
  const [teamId, setTeamId] = useState<string | null>(null)
  const [invite, setInvite] = useState<InviteDTO | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const canAssignOwner = session?.is_admin === true || org?.role === 'owner'
  const roleOptionsFiltered = canAssignOwner ? roleOptions : roleOptions.filter((o) => o.id !== 'owner')
  const needsTeam = role !== 'owner'

  const teamOptions = teams.map((t) => ({ id: t.id, label: t.name }))

  const createMutation = useMutation({
    mutationFn: createInvite,
    onSuccess: (result) => {
      setInvite(result)
      setEmail('')
      setRole('agent')
      setTeamId(null)
      setError(null)
      setCopied(false)
      void queryClient.invalidateQueries({ queryKey: ['invites', org?.id ?? ''] })
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to create invite')
    },
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!email.trim()) {
      setError('Email is required')
      return
    }
    if (needsTeam && !teamId) {
      setError('A team is required for this role')
      return
    }
    setError(null)
    setInvite(null)
    createMutation.mutate({
      email: email.trim(),
      role,
      team_id: needsTeam && teamId ? teamId : undefined,
    })
  }

  const joinUrl = invite?.token ? `${window.location.origin}/join?t=${invite.token}` : ''

  async function copyLink() {
    if (!joinUrl) return
    try {
      await navigator.clipboard.writeText(joinUrl)
      setCopied(true)
    } catch {
      // clipboards can fail; ignore
    }
  }

  return (
    <ModalDialog title="Invite member">
      {invite ? (
        <div className="space-y-3">
          <p className="text-sm text-zinc-300">
            Invite sent to <span className="font-medium text-zinc-100">{invite.email}</span>.
          </p>
          <div>
            <span className="mb-1 block text-xs font-medium uppercase tracking-wider text-zinc-500">
              Invite link
            </span>
            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded-lg bg-zinc-800 px-3 py-2 font-mono text-xs text-emerald-300">
                {joinUrl}
              </code>
              <Button type="button" size="sm" variant="secondary" onPress={copyLink}>
                {copied ? 'Copied' : 'Copy'}
              </Button>
            </div>
          </div>
          <p className="text-xs text-zinc-500">
            Share this link with {invite.email}. It expires in 7 days.
          </p>
          <div className="mt-2 flex justify-end">
            <Button variant="secondary" slot="close">
              Done
            </Button>
          </div>
        </div>
      ) : (
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <TextField
            label="Email"
            type="email"
            value={email}
            onChange={setEmail}
            placeholder="teammate@example.com"
          />
          <div>
            <span className="mb-1.5 block text-sm font-medium text-zinc-300">Role</span>
            <SelectField
              options={roleOptionsFiltered}
              selectedKey={role}
              onSelectionChange={(key) => {
                if (key != null) setRole(String(key))
              }}
            />
          </div>
          <div>
            <span className="mb-1.5 block text-sm font-medium text-zinc-300">Team</span>
            <SelectField
              options={teamOptions}
              selectedKey={needsTeam ? teamId : null}
              onSelectionChange={(key) => {
                if (key != null) setTeamId(String(key))
              }}
              isDisabled={!needsTeam}
              placeholder={needsTeam ? 'Select a team…' : 'Owners are not part of a team'}
            />
          </div>
          {error ? <p className="text-sm text-red-400">{error}</p> : null}
          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" variant="ghost" slot="close">
              Cancel
            </Button>
            <Button type="submit" isDisabled={createMutation.isPending}>
              {createMutation.isPending ? 'Inviting…' : 'Create invite'}
            </Button>
          </div>
        </form>
      )}
    </ModalDialog>
  )
}

type TeamDialogState = { mode: 'create' } | { mode: 'rename'; team: TeamDTO } | null

function TeamDialog({ state, onDone }: { state: Exclude<TeamDialogState, null>; onDone: () => void }) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const [name, setName] = useState(state.mode === 'rename' ? state.team.name : '')
  const [error, setError] = useState<string | null>(null)

  const isRename = state.mode === 'rename'

  const mutation = useMutation({
    mutationFn: () =>
      isRename ? updateTeam(state.team.id, name.trim()) : createTeam(name.trim()),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['teams', org?.id ?? ''] })
      onDone()
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to save team')
    },
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) {
      setError('Name is required')
      return
    }
    setError(null)
    mutation.mutate()
  }

  return (
    <ModalDialog isOpen onOpenChange={(open) => !open && onDone()} title={isRename ? 'Rename team' : 'New team'}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField
          label="Name"
          value={name}
          onChange={setName}
          placeholder="e.g. Marketing"
          autoFocus
        />
        {error ? <p className="text-sm text-red-400">{error}</p> : null}
        <div className="mt-1 flex justify-end gap-2">
          <Button type="button" variant="ghost" onPress={onDone}>
            Cancel
          </Button>
          <Button type="submit" isDisabled={mutation.isPending}>
            {isRename ? 'Rename' : 'Create'}
          </Button>
        </div>
      </form>
    </ModalDialog>
  )
}

function TeamsSection({
  teams,
  canManage,
  onTeamDialog,
}: {
  teams: TeamDTO[]
  canManage: boolean
  onTeamDialog: (state: Exclude<TeamDialogState, null>) => void
}) {
  const { org } = useSession()
  const queryClient = useQueryClient()

  const deleteMutation = useMutation({
    mutationFn: deleteTeam,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['teams', org?.id ?? ''] })
    },
  })

  const confirm = useConfirm()

  async function handleDelete(team: TeamDTO) {
    const confirmed = await confirm({
      title: 'Delete team',
      message: `"${team.name}" will be removed. Members stay in the organization.`,
      confirmLabel: 'Delete',
    })
    if (!confirmed) return
    deleteMutation.mutate(team.id)
  }

  return (
    <div className="mb-6">
      <div className="mb-2 flex items-center justify-between">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-zinc-500">Teams</h2>
        {canManage ? (
          <Button size="sm" variant="ghost" onPress={() => onTeamDialog({ mode: 'create' })}>
            <Plus size={14} />
            Add team
          </Button>
        ) : null}
      </div>
      {teams.length === 0 ? (
        <p className="rounded-xl border border-dashed border-zinc-800 px-4 py-3 text-sm text-zinc-500">
          No teams yet. Teams group members and route conversations to the right people.
        </p>
      ) : (
        <div className="flex flex-wrap gap-2">
          {teams.map((t) => (
            <div
              key={t.id}
              className="flex items-center gap-2 rounded-xl border border-zinc-800 bg-zinc-900/50 px-3 py-2"
            >
              <span className="text-sm font-medium text-zinc-100">{t.name}</span>
              <span className="text-xs text-zinc-500">
                {t.member_count} {t.member_count === 1 ? 'member' : 'members'}
              </span>
              {canManage ? (
                <>
                  <button
                    type="button"
                    aria-label={`Rename ${t.name}`}
                    className="rounded p-1 text-zinc-500 transition hover:bg-zinc-800 hover:text-zinc-200 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
                    onClick={() => onTeamDialog({ mode: 'rename', team: t })}
                  >
                    <Pencil size={13} />
                  </button>
                  <button
                    type="button"
                    aria-label={`Delete ${t.name}`}
                    className="rounded p-1 text-zinc-500 transition hover:bg-zinc-800 hover:text-red-400 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
                    onClick={() => handleDelete(t)}
                  >
                    <Trash2 size={13} />
                  </button>
                </>
              ) : null}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function inviteStatusTone(status: string): 'green' | 'amber' | 'zinc' {
  switch (status) {
    case 'pending':
      return 'green'
    default:
      return 'zinc'
  }
}

function InvitesSection({
  invites,
  canManage,
  revokePending,
  onRevoke,
}: {
  invites: InviteDTO[]
  canManage: boolean
  revokePending: boolean
  onRevoke: (inv: InviteDTO) => void
}) {
  const pending = invites.filter((i) => i.status === 'pending')
  const past = invites.filter((i) => i.status !== 'pending')

  return (
    <div className="mb-6">
      <div className="mb-2 flex items-center justify-between">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-zinc-500">
          Pending invites
        </h2>
      </div>
      {pending.length === 0 ? (
        <p className="rounded-xl border border-dashed border-zinc-800 px-4 py-3 text-sm text-zinc-500">
          No pending invites. Invite people by email — they'll get a join link to set their own
          password and sign in.
        </p>
      ) : (
        <table className="w-full border-collapse text-left text-sm">
          <thead>
            <tr className="border-b border-zinc-800 text-xs uppercase tracking-wider text-zinc-500">
              <th className="py-2.5 pr-4 font-medium">Email</th>
              <th className="py-2.5 pr-4 font-medium">Role</th>
              <th className="py-2.5 pr-4 font-medium">Team</th>
              <th className="py-2.5 pr-4 font-medium">Expires</th>
              {canManage ? <th className="py-2.5 pl-4 text-right font-medium">Actions</th> : null}
            </tr>
          </thead>
          <tbody>
            {pending.map((i) => (
              <tr key={i.id} className="border-b border-zinc-800/60">
                <td className="py-3 pr-4 text-zinc-100">{i.email}</td>
                <td className="py-3 pr-4">
                  <Badge tone={inviteStatusTone(i.status)}>{i.role}</Badge>
                </td>
                <td className="py-3 pr-4 text-zinc-400">{i.team_name ?? '—'}</td>
                <td className="py-3 pr-4 text-zinc-400">{formatDate(i.expires_at)}</td>
                {canManage ? (
                  <td className="py-3 pl-4 text-right">
                    <Button
                      size="icon"
                      variant="ghost"
                      aria-label={`Revoke invite for ${i.email}`}
                      onPress={() => onRevoke(i)}
                      isDisabled={revokePending}
                    >
                      <Trash2 size={16} className="text-zinc-500 hover:text-red-400" />
                    </Button>
                  </td>
                ) : null}
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {past.length > 0 ? (
        <>
          <h3 className="mb-2 mt-5 text-xs font-semibold uppercase tracking-wider text-zinc-600">
            Accepted & revoked
          </h3>
          <ul className="space-y-1">
            {past.map((i) => (
              <li key={i.id} className="flex items-center gap-2 text-sm text-zinc-500">
                <span className="truncate">{i.email}</span>
                <Badge tone={inviteStatusTone(i.status)}>{i.status}</Badge>
              </li>
            ))}
          </ul>
        </>
      ) : null}
    </div>
  )
}

export function TeamPage() {
  const { session, org } = useSession()
  const queryClient = useQueryClient()
  const meId = session?.user.id
  const orgId = org?.id ?? ''
  const [teamDialog, setTeamDialog] = useState<TeamDialogState>(null)
  const [updatingId, setUpdatingId] = useState<string | null>(null)

  const canManage = session?.is_admin === true || org?.role === 'owner' || org?.role === 'admin'
  const isOwnerOrSuper = session?.is_admin === true || org?.role === 'owner'

  const teamQuery = useQuery({
    queryKey: ['team', orgId],
    queryFn: listTeam,
    enabled: orgId !== '',
  })
  const teamsQuery = useQuery({
    queryKey: ['teams', orgId],
    queryFn: listTeams,
    enabled: orgId !== '',
  })
  const invitesQuery = useQuery({
    queryKey: ['invites', orgId],
    queryFn: listInvites,
    enabled: orgId !== '',
  })

  const updateMutation = useMutation({
    mutationFn: ({ userId, input }: { userId: string; input: { role?: string; team_id?: string } }) =>
      updateTeamMember(userId, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['team', orgId] })
      setUpdatingId(null)
    },
    onError: () => {
      setUpdatingId(null)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: deleteTeamMember,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['team', orgId] })
    },
  })

  const revokeMutation = useMutation({
    mutationFn: revokeInvite,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['invites', orgId] })
    },
  })

  const members = teamQuery.data?.items ?? []
  const teams = teamsQuery.data?.items ?? []
  const teamOptions = teams.map((t) => ({ id: t.id, label: t.name }))
  const invites = invitesQuery.data?.items ?? []
  const confirm = useConfirm()

  async function handleRevokeInvite(inv: InviteDTO) {
    const confirmed = await confirm({
      title: 'Revoke invite',
      message: `${inv.email} will no longer be able to join with this invite.`,
      confirmLabel: 'Revoke',
    })
    if (!confirmed) return
    revokeMutation.mutate(inv.id)
  }

  function handleRoleChange(m: TeamMemberDTO, role: string) {
    if (m.role === role) return
    setUpdatingId(m.id)
    void updateMutation.mutateAsync({ userId: m.id, input: { role } })
  }

  function handleTeamChange(m: TeamMemberDTO, teamId: string) {
    if ((m.team_id ?? '') === teamId) return
    setUpdatingId(m.id)
    void updateMutation.mutateAsync({ userId: m.id, input: { team_id: teamId } })
  }

  async function handleDelete(m: TeamMemberDTO) {
    const confirmed = await confirm({
      title: 'Remove member',
      message: `${m.name || m.email} will lose access to this organization.`,
      confirmLabel: 'Remove',
    })
    if (!confirmed) return
    deleteMutation.mutate(m.id)
  }

  function canManageMember(m: TeamMemberDTO): boolean {
    if (!canManage) return false
    if (m.id === meId) return false
    if (m.role === 'owner' && !isOwnerOrSuper) return false
    return true
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="flex items-center justify-between border-b border-zinc-800/80 px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold text-zinc-100">Team</h1>
          <p className="text-sm text-zinc-500">
            Teams group members; conversations route to the team of the WhatsApp number.
          </p>
        </div>
        {canManage ? (
          <DialogTrigger>
            <Button size="sm">
              <Plus size={14} />
              Invite member
            </Button>
            <InviteDialog teams={teams} />
          </DialogTrigger>
        ) : null}
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        {teamsQuery.isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between gap-3 rounded-xl border border-zinc-800 bg-zinc-900/50 p-4">
                <Skeleton className="h-9 w-9 rounded-full" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-3.5 w-40" />
                  <Skeleton className="h-3 w-24" />
                </div>
                <Skeleton className="h-6 w-20" />
              </div>
            ))}
          </div>
        ) : (
          <TeamsSection teams={teams} canManage={canManage} onTeamDialog={setTeamDialog} />
        )}

        <InvitesSection
          invites={invites}
          canManage={canManage}
          revokePending={revokeMutation.isPending}
          onRevoke={handleRevokeInvite}
        />

        {teamQuery.isLoading ? (
          <div className="space-y-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <div key={i} className="flex items-center gap-3 py-3">
                <Skeleton className="h-8 w-8 rounded-full" />
                <div className="flex-1 space-y-2">
                  <Skeleton className="h-3.5 w-40" />
                  <Skeleton className="h-3 w-24" />
                </div>
                <Skeleton className="h-5 w-16" />
              </div>
            ))}
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
                <th className="py-2.5 pr-4 font-medium">Role</th>
                <th className="py-2.5 font-medium">Team</th>
                {canManage ? <th className="py-2.5 pl-4 text-right font-medium">Actions</th> : null}
              </tr>
            </thead>
            <tbody>
              {members.map((m) => {
                const manageable = canManageMember(m)
                const options = isOwnerOrSuper ? roleOptions : roleOptions.filter((o) => o.id !== 'owner')
                return (
                  <tr key={m.id} className="border-b border-zinc-800/60 transition hover:bg-zinc-900/40">
                    <td className="py-3 pr-4">
                      <div className="flex items-center gap-3">
                        <Avatar name={m.name} size={32} />
                        <span className="font-medium text-zinc-100">{m.name}</span>
                        {m.id === meId ? <Badge tone="green">You</Badge> : null}
                      </div>
                    </td>
                    <td className="py-3 pr-4 text-zinc-400">{m.email}</td>
                    <td className="py-3 pr-4">
                      {manageable ? (
                        <SelectField
                          ariaLabel={`Role for ${m.name || m.email}`}
                          options={options}
                          selectedKey={m.role}
                          onSelectionChange={(key) => {
                            if (key != null) handleRoleChange(m, String(key))
                          }}
                          isDisabled={updatingId === m.id}
                        />
                      ) : (
                        <Badge tone={roleTone(m.role)}>{m.role}</Badge>
                      )}
                    </td>
                    <td className="py-3">
                      {manageable && m.role !== 'owner' ? (
                        <SelectField
                          ariaLabel={`Team for ${m.name || m.email}`}
                          options={teamOptions}
                          selectedKey={m.team_id ?? null}
                          onSelectionChange={(key) => {
                            if (key != null) handleTeamChange(m, String(key))
                          }}
                          isDisabled={updatingId === m.id}
                          placeholder="Select a team…"
                        />
                      ) : m.role === 'owner' ? (
                        <Badge tone="amber">All teams</Badge>
                      ) : (
                        <span className="text-zinc-400">{m.team_name ?? '—'}</span>
                      )}
                    </td>
                    {canManage ? (
                      <td className="py-3 pl-4 text-right">
                        {manageable ? (
                          <Button
                            size="icon"
                            variant="ghost"
                            aria-label={`Remove ${m.name || m.email}`}
                            onPress={() => handleDelete(m)}
                            isDisabled={deleteMutation.isPending}
                          >
                            <Trash2 size={16} className="text-zinc-500 hover:text-red-400" />
                          </Button>
                        ) : null}
                      </td>
                    ) : null}
                  </tr>
                )
              })}
            </tbody>
          </table>
        )}
      </div>

      {teamDialog ? <TeamDialog state={teamDialog} onDone={() => setTeamDialog(null)} /> : null}
    </div>
  )
}
