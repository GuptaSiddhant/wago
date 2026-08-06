import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { FileText, Plus, RefreshCw, Trash2 } from 'lucide-react'
import {
  deleteTemplate,
  listAccounts,
  listTemplates,
  syncTemplates,
} from '../../api/client'
import type { MessageTemplateDTO } from '../../api/types'
import { useSession } from '../../lib/session'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { Spinner } from '../../components/ui/Spinner'
import { TemplateForm } from './TemplateForm'
import { TemplatePreview } from './TemplatePreview'

const statusTone: Record<string, 'green' | 'amber' | 'red' | 'blue' | 'zinc'> = {
  APPROVED: 'green',
  PENDING: 'amber',
  REJECTED: 'red',
  PAUSED: 'blue',
  IN_APPEAL: 'blue',
}

export function TemplatesPage() {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const orgId = org?.id ?? ''
  const [showCreate, setShowCreate] = useState(false)
  const [syncError, setSyncError] = useState<string | null>(null)

  const templatesQuery = useQuery({
    queryKey: ['templates', orgId],
    queryFn: listTemplates,
    enabled: orgId !== '',
  })
  const accountsQuery = useQuery({
    queryKey: ['accounts', orgId],
    queryFn: listAccounts,
    enabled: orgId !== '',
  })

  const deleteMutation = useMutation({
    mutationFn: deleteTemplate,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['templates', orgId] }),
  })

  const syncMutation = useMutation({
    mutationFn: syncTemplates,
    onSuccess: (res) => {
      queryClient.setQueryData(['templates', orgId], res)
      if (res.errors?.length) {
        setSyncError(res.errors.join(' · '))
      } else {
        setSyncError(null)
      }
    },
    onError: (err) => {
      setSyncError(err instanceof Error ? err.message : 'Sync failed')
    },
  })

  const templates = templatesQuery.data?.items ?? []
  const accounts = accountsQuery.data?.items ?? []

  function handleDelete(t: MessageTemplateDTO) {
    if (!window.confirm(`Delete template "${t.name}" from Meta and this workspace?`)) return
    deleteMutation.mutate(t.id)
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="flex items-center justify-between gap-3 border-b border-zinc-800/80 px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold text-zinc-100">Message templates</h1>
          <p className="text-sm text-zinc-500">
            Reusable WhatsApp templates submitted for Meta approval. Use approved templates to
            broadcast.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            size="sm"
            variant="secondary"
            aria-label="Sync template statuses from Meta"
            onPress={() => syncMutation.mutate()}
            isDisabled={syncMutation.isPending}
          >
            <RefreshCw size={14} className={syncMutation.isPending ? 'animate-spin' : ''} />
            Sync status
          </Button>
          <Button size="sm" onPress={() => setShowCreate(true)}>
            <Plus size={14} />
            New template
          </Button>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        {syncError ? (
          <div className="mb-4 rounded-xl border border-amber-500/20 bg-amber-500/5 p-3 text-sm text-amber-300">
            {syncError}
          </div>
        ) : null}

        {templatesQuery.isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : templates.length === 0 ? (
          <EmptyState
            icon={<FileText size={32} />}
            title="No message templates yet"
            description="Create a template and submit it for Meta review, or sync to import ones already in your WhatsApp Business Account."
          />
        ) : (
          <ul className="grid gap-3 md:grid-cols-2">
            {templates.map((t) => (
              <li
                key={t.id}
                className="flex flex-col gap-3 rounded-xl border border-zinc-800 bg-zinc-900/50 p-4"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="truncate font-mono text-sm font-medium text-zinc-100">
                        {t.name}
                      </span>
                      <Badge tone={statusTone[t.status] ?? 'zinc'}>{t.status}</Badge>
                    </div>
                    <p className="mt-0.5 truncate text-xs text-zinc-500">
                      {t.account_name || t.account_id} · {t.language} · {t.category}
                    </p>
                  </div>
                  <Button
                    size="icon"
                    variant="ghost"
                    aria-label={`Delete ${t.name}`}
                    onPress={() => handleDelete(t)}
                    isDisabled={deleteMutation.isPending}
                  >
                    <Trash2 size={15} className="text-zinc-500 hover:text-red-400" />
                  </Button>
                </div>

                <TemplatePreview
                  headerText={t.header_type === 'TEXT' ? t.header_text : undefined}
                  headerMedia={
                    t.header_type === 'MEDIA' && t.header_media_type
                      ? { media_type: t.header_media_type, filename: t.header_media_name }
                      : undefined
                  }
                  body={t.body}
                  footer={t.footer}
                  buttons={t.buttons}
                />

                {t.status === 'REJECTED' && t.meta_error ? (
                  <p className="rounded-lg border border-red-500/20 bg-red-500/5 px-3 py-2 text-xs text-red-400">
                    {t.meta_error}
                  </p>
                ) : null}
              </li>
            ))}
          </ul>
        )}
      </div>

      {showCreate ? (
        <TemplateForm accounts={accounts} onDone={() => setShowCreate(false)} />
      ) : null}
    </div>
  )
}
