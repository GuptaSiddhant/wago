import { useMemo, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Image as ImageIcon, Paperclip, X } from 'lucide-react'
import { createBroadcast, listContacts, uploadMedia } from '../../api/client'
import type { MessageTemplateDTO, WaAccountDTO } from '../../api/types'
import { useSession } from '../../lib/session'
import { useTemplateVariables } from '../../lib/useTemplateVariables'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { FormError } from '../../components/ui/FormError'
import { ModalDialog } from '../../components/ui/Modal'
import { SearchField } from '../../components/ui/SearchField'
import { SelectField } from '../../components/ui/Select'
import { TextField } from '../../components/ui/TextField'
import { TemplatePreview } from '../templates/TemplatePreview'

export function BroadcastForm({
  accounts,
  templates,
  onDone,
}: {
  accounts: WaAccountDTO[]
  templates: MessageTemplateDTO[]
  onDone: () => void
}) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const orgId = org?.id ?? ''

  const approved = templates.filter((t) => t.status === 'APPROVED')
  const [name, setName] = useState('')
  const [accountId, setAccountId] = useState<string | null>(accounts[0]?.id ?? null)
  const [templateId, setTemplateId] = useState<string | null>(null)
  const [rate, setRate] = useState('60')
  const [batch, setBatch] = useState('10')
  const [allContacts, setAllContacts] = useState(true)
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [contactSearch, setContactSearch] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [overrideMediaId, setOverrideMediaId] = useState('')
  const [overrideMediaType, setOverrideMediaType] = useState('')
  const [overrideMediaName, setOverrideMediaName] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const accountTemplates = useMemo(
    () => approved.filter((t) => t.account_id === accountId),
    [approved, accountId],
  )
  const selectedTemplate = approved.find((t) => t.id === templateId) ?? null

  const { variableCount, values, setValues } = useTemplateVariables(selectedTemplate?.body ?? '')

  const contactsQuery = useQuery({
    queryKey: ['contacts', orgId, contactSearch],
    queryFn: () => listContacts({ search: contactSearch, limit: 100 }),
    enabled: orgId !== '' && !allContacts,
  })
  const contacts = contactsQuery.data?.items ?? []

  const mutation = useMutation({
    mutationFn: () =>
      createBroadcast({
        name,
        account_id: accountId ?? '',
        template_id: templateId ?? '',
        params: values.map((v) => ({ type: 'text', text: v })),
        rate_per_minute: Math.max(1, Number.parseInt(rate, 10) || 60),
        batch_size: Math.max(1, Number.parseInt(batch, 10) || 10),
        all_contacts: allContacts,
        contact_ids: allContacts ? undefined : [...selected],
        header_media_type: overrideMediaType || undefined,
        header_media_id: overrideMediaId || undefined,
        header_media_name: overrideMediaName || undefined,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['broadcasts', orgId] })
      onDone()
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to create broadcast')
    },
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) => uploadMedia({ accountId: accountId ?? '', file }),
    onSuccess: (result) => {
      setOverrideMediaType(result.media_type)
      setOverrideMediaId(result.media_id)
      setOverrideMediaName(result.filename)
      setError(null)
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to upload media')
    },
  })

  const canOverrideMedia =
    selectedTemplate?.header_media_type === 'IMAGE' ||
    selectedTemplate?.header_media_type === 'VIDEO' ||
    selectedTemplate?.header_media_type === 'DOCUMENT'

  function handleMediaFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file || !accountId) {
      setError(accountId ? 'Choose a file to attach' : 'Select the WhatsApp number first')
      return
    }
    uploadMutation.mutate(file)
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) return setError('Broadcast name is required')
    if (!accountId) return setError('Select a WhatsApp number')
    if (!templateId) return setError('Select an approved template')
    if (!allContacts && selected.size === 0) return setError('Select at least one contact')
    setError(null)
    mutation.mutate()
  }

  function toggleContact(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <ModalDialog
      isOpen
      onOpenChange={(open) => !open && onDone()}
      title="New broadcast"
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField
          label="Broadcast name"
          value={name}
          onChange={setName}
          placeholder="Friday flash sale"
          isRequired
        />

        <div className="grid gap-3 sm:grid-cols-2">
          <SelectField
            label="WhatsApp number"
            options={accounts.map((a) => ({ id: a.id, label: a.display_name || a.phone_number_id }))}
            selectedKey={accountId}
            onSelectionChange={(key) => {
              setAccountId(String(key))
              setTemplateId(null)
            }}
          />
          <SelectField
            label="Approved template"
            options={accountTemplates.map((t) => ({ id: t.id, label: t.name }))}
            selectedKey={templateId}
            onSelectionChange={(key) => setTemplateId(String(key))}
            placeholder={accountTemplates.length ? 'Select…' : 'No approved templates'}
            isDisabled={!accountId || accountTemplates.length === 0}
          />
        </div>

        {selectedTemplate ? (
          <div>
            <span className="text-sm font-medium text-zinc-300">Template</span>
            <div className="mt-2">
              <TemplatePreview
                headerText={selectedTemplate.header_type === 'TEXT' ? selectedTemplate.header_text : undefined}
                headerMedia={
                  selectedTemplate.header_type === 'MEDIA' && selectedTemplate.header_media_type
                    ? {
                        media_type:
                          overrideMediaType || selectedTemplate.header_media_type,
                        filename: overrideMediaName || selectedTemplate.header_media_name,
                      }
                    : undefined
                }
                body={selectedTemplate.body}
                footer={selectedTemplate.footer}
                buttons={selectedTemplate.buttons}
                values={values}
              />
            </div>

            {canOverrideMedia ? (
              <div className="mt-3 rounded-xl border border-zinc-800 bg-zinc-900/40 p-3">
                <span className="text-sm font-medium text-zinc-300">Header media</span>
                <p className="mt-0.5 text-xs text-zinc-500">
                  Uses the template's
                  <span className="ml-1 font-medium uppercase text-zinc-400">
                    {overrideMediaType || selectedTemplate.header_media_type}
                  </span>{' '}
                  header. Optionally overwrite it for this broadcast only.
                </p>
                {overrideMediaId ? (
                  <div className="mt-2 flex items-center gap-2 rounded-lg bg-zinc-900 border border-zinc-700 px-3 py-2">
                    <ImageIcon size={14} className="shrink-0 text-emerald-400" />
                    <span className="min-w-0 flex-1 truncate text-sm text-zinc-200">
                      {overrideMediaName}
                    </span>
                    <button
                      type="button"
                      aria-label="Use template media instead"
                      className="shrink-0 rounded-full p-1 text-zinc-500 hover:bg-zinc-800 hover:text-red-400"
                      onClick={() => {
                        setOverrideMediaId('')
                        setOverrideMediaType('')
                        setOverrideMediaName('')
                      }}
                    >
                      <X size={13} />
                    </button>
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploadMutation.isPending}
                    className="mt-2 flex items-center justify-center gap-2 rounded-lg border border-dashed border-zinc-600 bg-zinc-900 px-3 py-2 text-sm text-zinc-400 transition hover:border-emerald-500 hover:text-emerald-400 disabled:opacity-60"
                  >
                    <Paperclip size={14} />
                    {uploadMutation.isPending ? 'Uploading…' : 'Overwrite with a file'}
                  </button>
                )}
                <input
                  ref={fileInputRef}
                  type="file"
                  className="hidden"
                  accept="image/*,video/mp4,video/3gpp,video/quicktime,application/pdf"
                  onChange={handleMediaFile}
                />
              </div>
            ) : null}
            {variableCount > 0 ? (
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                {Array.from({ length: variableCount }, (_, i) => (
                  <TextField
                    key={i}
                    label={`{{${i + 1}}}`}
                    value={values[i] ?? ''}
                    onChange={(v) => setValues((prev) => prev.map((p, j) => (j === i ? v : p)))}
                    placeholder={`Value ${i + 1}`}
                  />
                ))}
              </div>
            ) : null}
          </div>
        ) : null}

        <div className="grid gap-3 sm:grid-cols-2">
          <TextField
            label="Rate (messages per minute)"
            type="number"
            value={rate}
            onChange={setRate}
            description="Global sustained send rate, shared by all running broadcasts."
          />
          <TextField
            label="Batch size"
            type="number"
            value={batch}
            onChange={setBatch}
            description="How many recipients are pulled and sent at once per tick."
          />
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm font-medium text-zinc-300">Recipients</span>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant={allContacts ? 'primary' : 'secondary'}
              onPress={() => setAllContacts(true)}
            >
              All contacts
            </Button>
            <Button
              size="sm"
              variant={!allContacts ? 'primary' : 'secondary'}
              onPress={() => setAllContacts(false)}
            >
              Pick contacts
            </Button>
          </div>

          {allContacts ? (
            <p className="text-xs text-zinc-500">
              The template will be sent to every contact in this workspace.
            </p>
          ) : (
            <>
              <SearchField
                label="Search contacts"
                placeholder="Search contacts…"
                value={contactSearch}
                onChange={setContactSearch}
              />
              <div className="max-h-48 overflow-y-auto rounded-xl border border-zinc-800">
                {contactsQuery.isLoading ? (
                  <p className="p-3 text-xs text-zinc-500">Loading contacts…</p>
                ) : contacts.length === 0 ? (
                  <p className="p-3 text-xs text-zinc-500">No contacts found.</p>
                ) : (
                  <ul className="divide-y divide-zinc-800">
                    {contacts.map((c) => (
                      <li key={c.id}>
                        <button
                          type="button"
                          onClick={() => toggleContact(c.id)}
                          className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left hover:bg-zinc-900"
                        >
                          <span className="min-w-0">
                            <span className="block truncate text-sm text-zinc-200">
                              {c.name || c.phone}
                            </span>
                            <span className="block truncate text-xs text-zinc-500">{c.phone}</span>
                          </span>
                          <span
                            className={`flex h-5 w-5 shrink-0 items-center justify-center rounded border text-xs ${
                              selected.has(c.id)
                                ? 'border-emerald-500 bg-emerald-500 text-white'
                                : 'border-zinc-600 text-transparent'
                            }`}
                          >
                            ✓
                          </span>
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
              <p className="text-xs text-zinc-500">{selected.size} selected</p>
            </>
          )}
        </div>

        {selectedTemplate && approved.length === 0 ? (
          <div className="rounded-lg border border-amber-500/20 bg-amber-500/5 px-3 py-2 text-sm text-amber-300">
            <Badge tone="amber">No approved templates</Badge> — submit a template and wait for Meta
            approval before broadcasting.
          </div>
        ) : null}

        <FormError message={error} />

        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" variant="ghost" onPress={onDone}>
            Cancel
          </Button>
          <Button type="submit" isDisabled={mutation.isPending}>
            {mutation.isPending ? 'Creating…' : 'Start broadcast'}
          </Button>
        </div>
      </form>
    </ModalDialog>
  )
}
