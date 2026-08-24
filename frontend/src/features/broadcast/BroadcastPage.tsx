import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Megaphone, Plus, X } from 'lucide-react'
import {
  cancelBroadcast,
  getBroadcast,
  listBroadcasts,
  subscribeBroadcastEvents,
} from '../../api/client'
import type { BroadcastDTO, BroadcastRecipient } from '../../api/types'
import { useSession } from '../../lib/session'
import { useConfirm } from '../../lib/confirm'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { ModalDialog } from '../../components/ui/Modal'
import { Skeleton } from '../../components/ui/Skeleton'
import { Spinner } from '../../components/ui/Spinner'

const statusTone: Record<string, 'green' | 'amber' | 'red' | 'blue' | 'zinc'> = {
  queued: 'zinc',
  running: 'blue',
  completed: 'green',
  failed: 'red',
  cancelled: 'zinc',
}

function ProgressBar({ value }: { value: number }) {
  return (
    <div className="h-1.5 overflow-hidden rounded-full bg-panel-strong">
      <div
        className="h-full rounded-full bg-emerald-500 transition-all"
        style={{ width: `${Math.min(100, Math.max(0, value * 100))}%` }}
      />
    </div>
  )
}

function recipientTone(status: string): 'green' | 'red' | 'zinc' {
  if (status === 'sent') return 'green'
  if (status === 'failed') return 'red'
  return 'zinc'
}

function BroadcastDetailModal({ broadcastId, onClose }: { broadcastId: string; onClose: () => void }) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const detailQuery = useQuery({
    queryKey: ['broadcast', broadcastId],
    queryFn: () => getBroadcast(broadcastId),
  })

  // Live updates stream over SSE; the initial query is just a fast snapshot.
  const [live, setLive] = useState<BroadcastDTO | null>(null)
  const doneRef = useRef(false)

  useEffect(() => {
    setLive(null)
    doneRef.current = false
    const controller = new AbortController()
    let closed = false

    subscribeBroadcastEvents(
      broadcastId,
      (event, payload) => {
        setLive(payload)
        const terminal = event === 'done' || ['completed', 'failed', 'cancelled'].includes(payload.status)
        if (terminal && !closed && !doneRef.current) {
          doneRef.current = true
          void queryClient.invalidateQueries({ queryKey: ['broadcasts'] })
        }
      },
      controller.signal,
    ).catch(() => {
      // Stream dropped (network error / server restart); the snapshot stays.
    })

    return () => {
      closed = true
      controller.abort()
    }
  }, [broadcastId, org?.id, queryClient])

  const bc = live ?? detailQuery.data?.broadcast
  const recipients: BroadcastRecipient[] = detailQuery.data?.recipients ?? []
  const progress = bc && bc.recipient_count > 0 ? bc.sent_count / bc.recipient_count : 0

  return (
    <ModalDialog isOpen onOpenChange={(open) => !open && onClose()} title={bc?.name ?? 'Broadcast'}>
      {!bc ? (
        <div className="flex justify-center py-8">
          <Spinner />
        </div>
      ) : (
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-3 gap-3">
            <div className="rounded-xl border border-edge bg-panel p-3">
              <div className="text-xs uppercase tracking-wider text-ink-faint">Sent</div>
              <div className="mt-1 text-lg font-semibold text-ink">{bc.sent_count}</div>
            </div>
            <div className="rounded-xl border border-edge bg-panel p-3">
              <div className="text-xs uppercase tracking-wider text-ink-faint">Failed</div>
              <div className="mt-1 text-lg font-semibold text-red-400">{bc.failed_count}</div>
            </div>
            <div className="rounded-xl border border-edge bg-panel p-3">
              <div className="text-xs uppercase tracking-wider text-ink-faint">Total</div>
              <div className="mt-1 text-lg font-semibold text-ink">{bc.recipient_count}</div>
            </div>
          </div>

          <div className="flex flex-col gap-1.5">
            <div className="flex items-center justify-between text-xs text-ink-faint">
              <span>Progress</span>
              <span>{Math.round(progress * 100)}%</span>
            </div>
            <ProgressBar value={progress} />
          </div>

          <div className="max-h-64 overflow-y-auto rounded-xl border border-edge">
            <ul className="divide-y divide-edge">
              {recipients.length === 0 ? (
                <li className="p-3 text-xs text-ink-faint">No recipients yet.</li>
              ) : (
                recipients.map((r) => (
                  <li key={r.id} className="flex items-center justify-between gap-2 px-3 py-2">
                    <div className="min-w-0">
                      <div className="truncate text-sm text-ink">{r.name || r.phone}</div>
                      <div className="truncate text-xs text-ink-faint">{r.phone}</div>
                      {r.error ? <div className="truncate text-xs text-red-400">{r.error}</div> : null}
                    </div>
                    <Badge tone={recipientTone(r.status)}>{r.status}</Badge>
                  </li>
                ))
              )}
            </ul>
          </div>
        </div>
      )}
    </ModalDialog>
  )
}

export function BroadcastPage() {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const orgId = org?.id ?? ''
  const [detailId, setDetailId] = useState<string | null>(null)

  const broadcastsQuery = useQuery({
    queryKey: ['broadcasts', orgId],
    queryFn: listBroadcasts,
    enabled: orgId !== '',
    refetchInterval: (query) =>
      query.state.data?.items.some(
        (b) => b.status === 'queued' || b.status === 'running',
      )
        ? 3000
        : false,
  })

  const cancelMutation = useMutation({
    mutationFn: cancelBroadcast,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['broadcasts', orgId] }),
  })

  const confirm = useConfirm()
  const broadcasts = broadcastsQuery.data?.items ?? []

  async function handleCancel(b: BroadcastDTO) {
    const confirmed = await confirm({
      title: 'Cancel broadcast',
      message: `"${b.name}" will stop sending. Recipients already messaged are unaffected.`,
      confirmLabel: 'Cancel broadcast',
    })
    if (!confirmed) return
    cancelMutation.mutate(b.id)
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="flex items-center justify-between gap-3 border-b border-edge px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold text-ink">Broadcasts</h1>
          <p className="text-sm text-ink-faint">
            Send an approved template to a list of contacts, paced to stay within rate limits.
          </p>
        </div>
        <Button size="sm" onPress={() => navigate({ to: '/broadcast/new' })}>
          <Plus size={14} />
          New broadcast
        </Button>
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        {broadcastsQuery.isLoading ? (
          <ul className="grid gap-3 md:grid-cols-2">
            {Array.from({ length: 4 }).map((_, i) => (
              <li key={i} className="flex flex-col gap-3 rounded-xl border border-edge bg-panel p-4">
                <Skeleton className="h-4 w-40" />
                <Skeleton className="h-3 w-24" />
                <Skeleton className="h-2.5 w-full" />
              </li>
            ))}
          </ul>
        ) : broadcasts.length === 0 ? (
                  <EmptyState
                    icon={<Megaphone size={32} />}
                    title="No broadcasts yet"
                    description="Create a broadcast to send an approved template to your contacts, batched and paced automatically."
                  />
        ) : (
          <ul className="grid gap-3 md:grid-cols-2">
            {broadcasts.map((b) => {
              const progress = b.recipient_count > 0 ? b.sent_count / b.recipient_count : 0
              const active = b.status === 'queued' || b.status === 'running'
              return (
                <li
                  key={b.id}
                  className="rounded-xl border border-edge bg-panel p-4"
                >
                  <div className="flex items-start justify-between gap-2">
                    <button
                      type="button"
                      onClick={() => setDetailId(b.id)}
                      className="min-w-0 flex-1 rounded-lg text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
                    >
                      <span className="block truncate text-sm font-medium text-ink">
                        {b.name}
                      </span>
                      <span className="mt-0.5 block truncate text-xs text-ink-faint">
                        {b.template_name || b.template_id} · {b.account_name || b.account_id}
                      </span>
                    </button>
                    <div className="flex shrink-0 items-center gap-1">
                      <Badge tone={statusTone[b.status] ?? 'zinc'}>{b.status}</Badge>
                      {active ? (
                        <Button
                          size="icon"
                          variant="ghost"
                          aria-label={`Cancel ${b.name}`}
                          onPress={() => handleCancel(b)}
                          isDisabled={cancelMutation.isPending}
                        >
                          <X size={15} className="text-ink-faint hover:text-red-400" />
                        </Button>
                      ) : null}
                    </div>
                  </div>

                  <div className="mt-3 flex items-center gap-3">
                    <div className="flex-1">
                      <ProgressBar value={progress} />
                    </div>
                    <span className="shrink-0 text-xs text-ink-faint">
                      {b.sent_count}/{b.recipient_count}
                      {b.failed_count > 0 ? (
                        <span className="text-red-400"> · {b.failed_count} failed</span>
                      ) : null}
                    </span>
                  </div>

                  <p className="mt-2 text-xs text-ink-faint">
                    {b.rate_per_minute || 60} msg/min ·{' '}
                    {b.finished_at ? `finished ${b.finished_at}` : b.started_at ? 'in progress' : 'queued'}
                  </p>
                </li>
              )
            })}
          </ul>
        )}
      </div>

      {detailId ? (
        <BroadcastDetailModal broadcastId={detailId} onClose={() => setDetailId(null)} />
      ) : null}
    </div>
  )
}
