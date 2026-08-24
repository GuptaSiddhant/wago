import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plug, Pencil, Plus, RefreshCw, Smartphone, Trash2 } from 'lucide-react'
import {
  accountBusinessProfile,
  accountMeta,
  accountWebhookStatus,
  connectAccountWebhook,
  createAccount,
  deleteAccount,
  listAccounts,
  listTeams,
  syncAccountBusinessProfile,
  updateAccount,
} from '../../api/client'
import { useSession } from '../../lib/session'
import { useConfirm } from '../../lib/confirm'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { Skeleton } from '../../components/ui/Skeleton'
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
        <p className="-mt-2 text-xs text-ink-faint">
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
          <span className="mb-1.5 block text-sm font-medium text-ink-muted">Status</span>
          <SelectField
            options={statusOptions}
            selectedKey={status}
            onSelectionChange={(key) => {
              if (key != null) setStatus(String(key))
            }}
          />
        </div>
        <div>
          <span className="mb-1.5 block text-sm font-medium text-ink-muted">Team</span>
          <SelectField
            options={teamOptions}
            selectedKey={teamId}
            onSelectionChange={(key) => {
              if (key != null) setTeamId(String(key))
            }}
            placeholder="No team — visible to all"
          />
          <p className="mt-1 text-xs text-ink-faint">
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

function WebhookConnect({ accountId }: { accountId: string }) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const orgId = org?.id ?? ''

  const statusQuery = useQuery({
    queryKey: ['account-webhook', orgId, accountId],
    queryFn: () => accountWebhookStatus(accountId),
    enabled: orgId !== '',
    staleTime: 60_000,
  })

  const connectMutation = useMutation({
    mutationFn: () => connectAccountWebhook(accountId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['account-webhook', orgId, accountId] })
    },
  })

  const result = statusQuery.data
  if (statusQuery.isLoading) {
    return (
      <div className="mt-2 flex items-center gap-2 text-xs text-ink-faint">
        <Spinner className="h-3 w-3" /> Checking webhook…
      </div>
    )
  }

  if (result && result.ok) {
    return (
      <div className="mt-2 flex items-center gap-2">
        <Badge tone="green">Webhook connected</Badge>
        {result.callback_url ? (
          <code className="min-w-0 truncate rounded bg-panel-strong px-1.5 py-0.5 text-[10px] text-emerald-300">
            {result.callback_url}
          </code>
        ) : null}
      </div>
    )
  }

  const missingBaseUrl = !result?.callback_url
  return (
    <div className="mt-2 space-y-2">
      {missingBaseUrl ? (
        <p className="text-xs text-amber-400">
          Webhooks can't be connected until <code>PUBLIC_BASE_URL</code> is set on this instance.
        </p>
      ) : (
        <p className="text-xs text-ink-faint">
          Meta isn't delivering messages for this number yet.
        </p>
      )}
      <Button
        size="sm"
        variant="secondary"
        onPress={() => connectMutation.mutate()}
        isDisabled={connectMutation.isPending || missingBaseUrl}
      >
        {connectMutation.isPending ? <Spinner className="h-3.5 w-3.5" /> : <Plug size={13} />}
        {connectMutation.isPending ? 'Connecting…' : 'Connect webhook'}
      </Button>
      {connectMutation.error ? (
        <p className="text-xs text-red-400">
          {connectMutation.error instanceof Error
            ? connectMutation.error.message
            : 'Failed to connect webhook'}
        </p>
      ) : null}
      {connectMutation.data && !connectMutation.data.ok ? (
        <p className="text-xs text-red-400">{connectMutation.data.error}</p>
      ) : null}
      {connectMutation.data && connectMutation.data.ok ? (
        <p className="text-xs text-emerald-400">{connectMutation.data.message}</p>
      ) : null}
    </div>
  )
}

function BusinessProfileSync({ accountId }: { accountId: string }) {
  const { org } = useSession()
  const orgId = org?.id ?? ''
  const queryClient = useQueryClient()
  const canManageData = org?.role === 'owner'

  const profileQuery = useQuery({
    queryKey: ['account-business-profile', orgId, accountId],
    queryFn: () => accountBusinessProfile(accountId),
    enabled: orgId !== '',
    staleTime: 5 * 60_000,
  })

  const syncMutation = useMutation({
    mutationFn: () => syncAccountBusinessProfile(accountId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['account-business-profile', orgId, accountId] })
    },
  })

  const profile = profileQuery.data?.profile

  return (
    <div className="mt-2 space-y-2">
      <div className="flex items-center gap-2">
        <Badge tone={syncMutation.isSuccess ? 'green' : 'zinc'}>Business profile</Badge>
        {canManageData ? (
          <Button
            size="sm"
            variant="secondary"
            onPress={() => syncMutation.mutate()}
            isDisabled={syncMutation.isPending}
          >
            {syncMutation.isPending ? <Spinner className="h-3.5 w-3.5" /> : <RefreshCw size={13} />}
            {syncMutation.isPending ? 'Syncing…' : 'Sync from org'}
          </Button>
        ) : null}
      </div>

      {syncMutation.isError ? (
        <p className="text-xs text-red-400">
          {syncMutation.error instanceof Error
            ? syncMutation.error.message
            : 'Failed to sync the business profile'}
        </p>
      ) : null}
      {syncMutation.isSuccess ? (
        <p className="text-xs text-emerald-400">{syncMutation.data?.message ?? 'Synced.'}</p>
      ) : null}

      {profileQuery.isLoading ? (
        <p className="text-xs text-ink-faint">Fetching WhatsApp profile…</p>
      ) : profileQuery.data && !profileQuery.data.ok ? (
        <p className="text-xs text-ink-faint">
          Profile unavailable{profileQuery.data.error ? ` — ${profileQuery.data.error}` : ''}
        </p>
      ) : profile ? (
        <dl className="space-y-1 text-xs text-ink-faint">
          <ProfileField label="About" value={stringOf(profile.about)} />
          <ProfileField label="Address" value={stringOf(profile.address)} />
          <ProfileField label="Description" value={stringOf(profile.description)} />
          <ProfileField label="Email" value={stringOf(profile.email)} />
          <ProfileField label="Websites" value={arrayOf(profile.websites).join(', ')} />
          <ProfileField label="Vertical" value={stringOf(profile.vertical)} />
        </dl>
      ) : null}
    </div>
  )
}

function ProfileField({ label, value }: { label: string; value: string }) {
  if (!value) return null
  return (
    <div className="flex gap-2">
      <span className="w-24 shrink-0 text-ink-faint">{label}</span>
      <span className="min-w-0 truncate text-ink-muted">{value}</span>
    </div>
  )
}

function stringOf(v: unknown): string {
  return typeof v === 'string' ? v : ''
}

function arrayOf(v: unknown): string[] {
  return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []
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
      <div className="mt-2 flex items-center gap-2 text-xs text-ink-faint">
        <Spinner className="h-3 w-3" /> Checking number health…
      </div>
    )
  }

  const result = metaQuery.data
  if (!result || !result.ok || !result.info) {
    return (
      <div className="mt-2 text-xs text-ink-faint">
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

  const canManageData = session?.is_admin === true || org?.role === 'owner'

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
  const confirm = useConfirm()

  async function handleDelete(a: WaAccountDTO) {
    const confirmed = await confirm({
      title: 'Disconnect number',
      message: `${a.display_name || a.phone_number_id} will be disconnected. This workspace will stop sending and receiving its messages.`,
      confirmLabel: 'Disconnect',
    })
    if (!confirmed) return
    deleteMutation.mutate(a.id)
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="flex items-center justify-between border-b border-edge px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold text-ink">WhatsApp Numbers</h1>
          <p className="text-sm text-ink-faint">
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
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between gap-3 rounded-xl border border-edge bg-panel p-4">
                <div className="flex items-center gap-3">
                  <Skeleton className="h-10 w-10 rounded-full" />
                  <div className="space-y-2">
                    <Skeleton className="h-3.5 w-40" />
                    <Skeleton className="h-3 w-28" />
                  </div>
                </div>
                <Skeleton className="h-6 w-20" />
              </div>
            ))}
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
                className="flex items-start gap-3 rounded-xl border border-edge bg-panel p-4"
              >
                <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-emerald-500/10 text-emerald-400">
                  <Smartphone size={17} />
                </div>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center justify-between gap-2">
                    <span className="truncate text-sm font-medium text-ink">
                      {a.display_name || a.phone_number_id}
                    </span>
                    <Badge tone={a.status === 'connected' ? 'green' : 'red'}>{a.status}</Badge>
                  </div>
                  <p className="mt-0.5 truncate text-xs text-ink-faint">
                    phone_number_id: {a.phone_number_id}
                  </p>
                  <p className="mt-0.5 truncate text-xs text-ink-faint">
                    Team: {a.team_name ?? 'All'}
                  </p>
                  <NumberHealth accountId={a.id} />
                  <WebhookConnect accountId={a.id} />
                  <BusinessProfileSync accountId={a.id} />
                </div>
                {canManageData ? (
                  <div className="flex shrink-0 gap-1">
                    <Button
                      size="icon"
                      variant="ghost"
                      aria-label={`Edit ${a.display_name || a.phone_number_id}`}
                      onPress={() => setDialog({ mode: 'edit', account: a })}
                    >
                      <Pencil size={16} className="text-ink-faint hover:text-ink" />
                    </Button>
                    <Button
                      size="icon"
                      variant="ghost"
                      aria-label={`Delete ${a.display_name || a.phone_number_id}`}
                      onPress={() => handleDelete(a)}
                      isDisabled={deleteMutation.isPending}
                    >
                      <Trash2 size={16} className="text-ink-faint hover:text-red-400" />
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
