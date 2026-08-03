import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { DialogTrigger } from 'react-aria-components'
import { Pencil, Plus, Trash2, Users } from 'lucide-react'
import {
  createTeam,
  createTeamMember,
  deleteTeam,
  deleteTeamMember,
  listTeam,
  listTeams,
  updateTeam,
  updateTeamMember,
} from '../../api/client'
import { useSession } from '../../lib/session'
import { Avatar } from '../../components/ui/Avatar'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { ModalDialog } from '../../components/ui/Modal'
import { SelectField } from '../../components/ui/Select'
import { Spinner } from '../../components/ui/Spinner'
import { TextField } from '../../components/ui/TextField'
import type { TeamDTO, TeamMemberDTO } from '../../api/types'

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

function AddMemberDialog({ teams }: { teams: TeamDTO[] }) {
  const { session, org } = useSession()
  const queryClient = useQueryClient()
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [role, setRole] = useState('agent')
  const [teamId, setTeamId] = useState<string | null>(null)
  const [generatedPassword, setGeneratedPassword] = useState('')
  const [error, setError] = useState<string | null>(null)

  const canAssignOwner = session?.isAdmin === true || org?.role === 'owner'
  const roleOptionsFiltered = canAssignOwner ? roleOptions : roleOptions.filter((o) => o.id !== 'owner')
  const needsTeam = role !== 'owner'

  const teamOptions = teams.map((t) => ({ id: t.id, label: t.name }))

  const createMutation = useMutation({
    mutationFn: createTeamMember,
    onSuccess: (result) => {
      setGeneratedPassword(result.generated_password ?? '')
      setEmail('')
      setName('')
      setRole('agent')
      setTeamId(null)
      setError(null)
      void queryClient.invalidateQueries({ queryKey: ['team', org?.id ?? ''] })
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to add member')
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
    setGeneratedPassword('')
    createMutation.mutate({
      email: email.trim(),
      name: name.trim(),
      role,
      team_id: needsTeam && teamId ? teamId : undefined,
    })
  }

  return (
    <ModalDialog title="Add member">
      {generatedPassword ? (
        <div>
          <p className="text-sm text-zinc-300">
            <span className="font-medium text-amber-300">Temporary password:</span>{' '}
            <code className="rounded bg-zinc-800 px-1.5 py-0.5 font-mono text-amber-200">
              {generatedPassword}
            </code>
          </p>
          <p className="mt-1 text-xs text-zinc-500">
            Share it with {email || 'the new member'} so they can sign in.
          </p>
          <div className="mt-4 flex justify-end">
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
          <TextField label="Name" value={name} onChange={setName} placeholder="Full name" />
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
              {createMutation.isPending ? 'Adding…' : 'Add member'}
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

  function handleDelete(team: TeamDTO) {
    if (!window.confirm(`Delete the "${team.name}" team?`)) return
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
                    className="rounded p-1 text-zinc-500 transition hover:bg-zinc-800 hover:text-zinc-200"
                    onClick={() => onTeamDialog({ mode: 'rename', team: t })}
                  >
                    <Pencil size={13} />
                  </button>
                  <button
                    type="button"
                    aria-label={`Delete ${t.name}`}
                    className="rounded p-1 text-zinc-500 transition hover:bg-zinc-800 hover:text-red-400"
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

export function TeamPage() {
  const { session, org } = useSession()
  const queryClient = useQueryClient()
  const meId = session?.user.id
  const orgId = org?.id ?? ''
  const [teamDialog, setTeamDialog] = useState<TeamDialogState>(null)
  const [updatingId, setUpdatingId] = useState<string | null>(null)

  const canManage = session?.isAdmin === true || org?.role === 'owner' || org?.role === 'admin'
  const isOwnerOrSuper = session?.isAdmin === true || org?.role === 'owner'

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

  const members = teamQuery.data?.items ?? []
  const teams = teamsQuery.data?.items ?? []
  const teamOptions = teams.map((t) => ({ id: t.id, label: t.name }))

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

  function handleDelete(m: TeamMemberDTO) {
    if (!window.confirm(`Remove ${m.name || m.email} from this organization?`)) return
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
              Add member
            </Button>
            <AddMemberDialog teams={teams} />
          </DialogTrigger>
        ) : null}
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        {teamsQuery.isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : (
          <TeamsSection teams={teams} canManage={canManage} onTeamDialog={setTeamDialog} />
        )}

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
