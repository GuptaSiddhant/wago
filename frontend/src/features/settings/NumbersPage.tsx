import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Smartphone, Trash2 } from 'lucide-react'
import {
  accountMeta,
  createAccount,
  deleteAccount,
  listAccounts,
  listTeams,
  updateAccount,
} from '../../api/client'
import { useSession } from '../../lib/session'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { ModalDialog } from '../../components/ui/Modal'
import { SelectField } from '../../components/ui/Select'
import { Spinner } from '../../components/ui/Spinner'
import { TextField } from '../../components/ui/TextField'
import type { TeamDTO, WaAccountDTO } from '../../api/types'

const statusOptions = [
  { id: 'disconnected', label: 'Disconnected' },
  { id: 'connected', label: 'Connected' },
]

type AccountDialogState = { mode: 'create' } | { mode: 'edit'; account: WaAccountDTO } | null

function AccountDialog({
  state,
  teams,
  onDone,
}: {
  state: Exclude<AccountDialogState, null>
  teams: TeamDTO[]
  onDone: () => void
}) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const isEdit = state.mode === 'edit'
  const existing = isEdit ? state.account : null

  const [displayName, setDisplayName] = useState(existing?.display_name ?? '')
  const [phoneNumberId, setPhoneNumberId] = useState(existing?.phone_number_id ?? '')
  const [wabaId, setWabaId] = useState(existing?.waba_id ?? '')
  const [accessToken, setAccessToken] = useState('')
  const [verifyToken, setVerifyToken] = useState('')
  const [status, setStatus] = useState(existing?.status ?? 'disconnected')
  const [teamId, setTeamId] = useState<string | null>(existing?.team_id ?? null)
  const [error, setError] = useState<string | null>(null)

  const teamOptions = teams.map((t) => ({ id: t.id, label: t.name }))

  const mutation = useMutation({
    mutationFn: () =>
      isEdit
        ? updateAccount(existing!.id, {
            display_name: displayName,
            phone_number_id: phoneNumberId,
            access_token: accessToken || undefined,
            verify_token: verifyToken || undefined,
            waba_id: wabaId || undefined,
            status,
            team_id: teamId ?? undefined,
          })
        : createAccount({
            display_name: displayName,
            phone_number_id: phoneNumberId,
            access_token: accessToken,
            verify_token: verifyToken,
            waba_id: wabaId,
            status,
            team_id: teamId ?? undefined,
          }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['accounts', org?.id ?? ''] })
      onDone()
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to save number')
    },
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!phoneNumberId.trim()) {
      setError('Phone number ID is required')
      return
    }
    if (!isEdit && !accessToken.trim()) {
      setError('Access token is required')
      return
    }
    setError(null)
    mutation.mutate()
  }

  return (
    <ModalDialog
      isOpen
      onOpenChange={(open) => !open && onDone()}
      title={isEdit ? 'Edit number' : 'Connect number'}
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField
          label="Display name"
          value={displayName}
          onChange={setDisplayName}
          placeholder="e.g. Support Line"
        />
        <TextField
          label="Phone number ID"
          value={phoneNumberId}
          onChange={setPhoneNumberId}
          placeholder="Meta phone_number_id"
          isRequired
        />
        <TextField
          label="WhatsApp Business Account (WABA) ID"
          value={wabaId}
          onChange={setWabaId}
          placeholder="Optional — required for org analytics"
        />
        <p className="-mt-2 text-xs text-zinc-500">
          Add the WABA ID and a token with <code>whatsapp_business_management</code> permission to
          unlock usage and cost analytics for this number.
        </p>
        <TextField
          label={isEdit ? 'Access token (leave blank to keep)' : 'Access token'}
          type="password"
          value={accessToken}
          onChange={setAccessToken}
          placeholder="Meta WhatsApp access token"
          isRequired={!isEdit}
        />
        <TextField
          label={isEdit ? 'Verify token (leave blank to keep)' : 'Verify token'}
          type="password"
          value={verifyToken}
          onChange={setVerifyToken}
          placeholder="Webhook verify token"
        />
        <div>
          <span className="mb-1.5 block text-sm font-medium text-zinc-300">Status</span>
          <SelectField
            options={statusOptions}
            selectedKey={status}
            onSelectionChange={(key) => {
              if (key != null) setStatus(String(key))
            }}
          />
        </div>
        <div>
          <span className="mb-1.5 block text-sm font-medium text-zinc-300">Team</span>
          <SelectField
            options={teamOptions}
            selectedKey={teamId}
            onSelectionChange={(key) => {
              if (key != null) setTeamId(String(key))
            }}
            placeholder="No team — visible to all"
          />
          <p className="mt-1 text-xs text-zinc-500">
            Conversations on this number route to this team.
          </p>
        </div>
        {error ? <p className="text-sm text-red-400">{error}</p> : null}
        <div className="mt-1 flex justify-end gap-2">
          <Button type="button" variant="ghost" onPress={onDone}>
            Cancel
          </Button>
          <Button type="submit" isDisabled={mutation.isPending}>
            {isEdit ? 'Save' : 'Connect'}
          </Button>
        </div>
      </form>
    </ModalDialog>
  )
}

const qualityTone: Record<string, 'green' | 'amber' | 'red' | 'zinc'> = {
  GREEN: 'green',
  YELLOW: 'amber',
  RED: 'red',
}

function NumberHealth({ accountId }: { accountId: string }) {
  const { org } = useSession()
  const orgId = org?.id ?? ''
  const metaQuery = useQuery({
    queryKey: ['account-meta', orgId, accountId],
    queryFn: () => accountMeta(accountId),
    enabled: orgId !== '',
    staleTime: 5 * 60_000,
  })

  if (metaQuery.isLoading) {
    return (
      <div className="mt-2 flex items-center gap-2 text-xs text-zinc-600">
        <Spinner className="h-3 w-3" /> Checking number health…
      </div>
    )
  }

  const result = metaQuery.data
  if (!result || !result.ok || !result.info) {
    return (
      <div className="mt-2 text-xs text-zinc-600">
        Meta health unavailable{result?.error ? ` — ${result.error}` : ''}
      </div>
    )
  }

  const info = result.info
  const quality = info.quality_rating as keyof typeof qualityTone
  return (
    <div className="mt-2 flex flex-wrap items-center gap-1.5">
      <Badge tone={qualityTone[quality] ?? 'zinc'}>Quality: {info.quality_rating || 'unknown'}</Badge>
      <Badge tone="zinc">{info.messaging_limit_tier || 'no tier'}</Badge>
      <Badge tone={info.code_verification_status === 'VERIFIED' ? 'green' : 'amber'}>
        {info.code_verification_status || 'not verified'}
      </Badge>
      {info.verified_name ? <Badge tone="zinc">{info.verified_name}</Badge> : null}
    </div>
  )
}

export function NumbersPage() {
  const { session, org } = useSession()
  const queryClient = useQueryClient()
  const orgId = org?.id ?? ''
  const [dialog, setDialog] = useState<AccountDialogState>(null)

  const canManageData = session?.isAdmin === true || org?.role === 'owner'

  const accountsQuery = useQuery({
    queryKey: ['accounts', orgId],
    queryFn: listAccounts,
    enabled: orgId !== '',
  })
  const teamsQuery = useQuery({
    queryKey: ['teams', orgId],
    queryFn: listTeams,
    enabled: orgId !== '',
  })

  const deleteMutation = useMutation({
    mutationFn: deleteAccount,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['accounts', orgId] })
    },
  })

  const accounts = accountsQuery.data?.items ?? []
  const teams = teamsQuery.data?.items ?? []

  function handleDelete(a: WaAccountDTO) {
    if (!window.confirm(`Disconnect ${a.display_name || a.phone_number_id}?`)) return
    deleteMutation.mutate(a.id)
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="flex items-center justify-between border-b border-zinc-800/80 px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold text-zinc-100">WhatsApp Numbers</h1>
          <p className="text-sm text-zinc-500">
            Meta accounts connected to this organization. Each number routes its conversations
            to the assigned team.
          </p>
        </div>
        {canManageData ? (
          <Button size="sm" onPress={() => setDialog({ mode: 'create' })}>
            <Plus size={14} />
            Connect number
          </Button>
        ) : null}
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
                    <Badge tone={a.status === 'connected' ? 'green' : 'red'}>{a.status}</Badge>
                  </div>
                  <p className="mt-0.5 truncate text-xs text-zinc-500">
                    phone_number_id: {a.phone_number_id}
                  </p>
                  <p className="mt-0.5 truncate text-xs text-zinc-500">
                    Team: {a.team_name ?? 'All'}
                  </p>
                  <NumberHealth accountId={a.id} />
                </div>
                {canManageData ? (
                  <div className="flex shrink-0 gap-1">
                    <Button
                      size="icon"
                      variant="ghost"
                      aria-label={`Edit ${a.display_name || a.phone_number_id}`}
                      onPress={() => setDialog({ mode: 'edit', account: a })}
                    >
                      <Pencil size={16} className="text-zinc-500 hover:text-zinc-200" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      aria-label={`Delete ${a.display_name || a.phone_number_id}`}
                      onPress={() => handleDelete(a)}
                      isDisabled={deleteMutation.isPending}
                    >
                      <Trash2 size={16} className="text-zinc-500 hover:text-red-400" />
                    </Button>
                  </div>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>

      {dialog ? (
        <AccountDialog state={dialog} teams={teams} onDone={() => setDialog(null)} />
      ) : null}
    </div>
  )
}
