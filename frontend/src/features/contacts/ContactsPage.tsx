import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Pencil, Plus, Trash2, Users, X } from 'lucide-react'
import {
  createContact,
  deleteContact,
  listContacts,
  listConversations,
  updateContact,
} from '../../api/client'
import { useSession } from '../../lib/session'
import { formatDate } from '../../lib/format'
import { Avatar } from '../../components/ui/Avatar'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { ModalDialog } from '../../components/ui/Modal'
import { SearchField } from '../../components/ui/SearchField'
import { Spinner } from '../../components/ui/Spinner'
import { TextField } from '../../components/ui/TextField'
import type { ContactDTO } from '../../api/types'

type ContactDialogState = { mode: 'create' } | { mode: 'edit'; contact: ContactDTO } | null

function ContactDialog({
  state,
  onDone,
}: {
  state: Exclude<ContactDialogState, null>
  onDone: () => void
}) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const [name, setName] = useState(state.mode === 'edit' ? state.contact.name : '')
  const [phone, setPhone] = useState(state.mode === 'edit' ? state.contact.phone : '')
  const [error, setError] = useState<string | null>(null)

  const isEdit = state.mode === 'edit'

  const mutation = useMutation({
    mutationFn: () =>
      isEdit
        ? updateContact(state.contact.id, { name: name.trim(), phone: phone.trim() })
        : createContact({ name: name.trim(), phone: phone.trim() }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['contacts', org?.id ?? ''] })
      onDone()
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to save contact')
    },
  })

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!phone.trim()) {
      setError('Phone is required')
      return
    }
    setError(null)
    mutation.mutate()
  }

  return (
    <ModalDialog isOpen onOpenChange={(open) => !open && onDone()} title={isEdit ? 'Edit contact' : 'New contact'}>
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <TextField
          label="Name"
          value={name}
          onChange={setName}
          placeholder="Full name"
          autoFocus={!isEdit}
        />
        <TextField
          label="Phone"
          value={phone}
          onChange={setPhone}
          placeholder="+15551234567"
          isRequired
        />
        {error ? <p className="text-sm text-red-400">{error}</p> : null}
        <div className="mt-1 flex justify-end gap-2">
          <Button type="button" variant="ghost" onPress={onDone}>
            Cancel
          </Button>
          <Button type="submit" isDisabled={mutation.isPending}>
            {isEdit ? 'Save' : 'Create'}
          </Button>
        </div>
      </form>
    </ModalDialog>
  )
}

function ContactDetailDialog({ contact, onClose }: { contact: ContactDTO; onClose: () => void }) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [tags, setTags] = useState<string[]>(contact.tags ?? [])
  const [notes, setNotes] = useState(contact.notes ?? '')
  const [newTag, setNewTag] = useState('')
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const historyQuery = useQuery({
    queryKey: ['conversations', org?.id ?? '', { contact: contact.id }],
    queryFn: () => listConversations({ contact: contact.id }),
    enabled: org?.id != null,
  })

  const saveMutation = useMutation({
    mutationFn: () => updateContact(contact.id, { tags, notes: notes.trim() }),
    onSuccess: () => {
      setSaved(true)
      void queryClient.invalidateQueries({ queryKey: ['contacts', org?.id ?? ''] })
      window.setTimeout(() => setSaved(false), 1500)
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to save contact')
    },
  })

  function isDirty(): boolean {
    const currentTags = [...(contact.tags ?? [])].sort()
    const nextTags = [...tags].sort()
    const tagsChanged =
      currentTags.length !== nextTags.length || currentTags.some((t, i) => t !== nextTags[i])
    return tagsChanged || notes.trim() !== (contact.notes ?? '')
  }

  function addTag() {
    const tag = newTag.trim().replace(/^#/, '')
    if (tag && !tags.includes(tag)) {
      setTags([...tags, tag])
    }
    setNewTag('')
  }

  const conversations = historyQuery.data?.items ?? []

  function openChat(convId: string) {
    onClose()
    void navigate({ to: '/inbox', search: { conv: convId } })
  }

  return (
    <ModalDialog isOpen onOpenChange={(open) => !open && onClose()} title="Contact details">
      <div className="flex flex-col gap-5">
        <div className="flex items-center gap-3">
          <Avatar name={contact.name || contact.phone} size={44} />
          <div className="min-w-0">
            <p className="truncate text-base font-semibold text-zinc-100">
              {contact.name || 'Unnamed contact'}
            </p>
            <p className="text-sm text-zinc-400">{contact.phone}</p>
          </div>
        </div>

        <div>
          <span className="mb-1.5 block text-sm font-medium text-zinc-300">Tags</span>
          {tags.length > 0 ? (
            <div className="mb-2 flex flex-wrap gap-1.5">
              {tags.map((t, i) => (
                <Badge key={`${t}-${i}`} tone="blue">
                  {t}
                  <button
                    type="button"
                    aria-label={`Remove tag ${t}`}
                    onClick={() => setTags(tags.filter((x) => x !== t))}
                    className="text-blue-300 hover:text-white"
                  >
                    <X size={12} />
                  </button>
                </Badge>
              ))}
            </div>
          ) : (
            <p className="mb-2 text-sm text-zinc-500">No tags yet.</p>
          )}
          <div className="flex gap-2">
            <input
              value={newTag}
              onChange={(e) => setNewTag(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  addTag()
                }
              }}
              placeholder="Add a tag…"
              className="min-w-0 flex-1 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-emerald-500"
            />
            <Button type="button" size="sm" variant="secondary" onPress={addTag}>
              Add
            </Button>
          </div>
        </div>

        <div>
          <span className="mb-1.5 block text-sm font-medium text-zinc-300">Notes</span>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={3}
            placeholder="Internal notes about this contact…"
            className="w-full rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-emerald-500"
          />
        </div>

        {saved ? <p className="text-sm text-emerald-400">Saved.</p> : null}
        {error ? <p className="text-sm text-red-400">{error}</p> : null}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="ghost" onPress={onClose}>
            Close
          </Button>
          <Button
            type="button"
            isDisabled={saveMutation.isPending || !isDirty()}
            onPress={() => saveMutation.mutate()}
          >
            {saveMutation.isPending ? 'Saving…' : 'Save'}
          </Button>
        </div>

        <div className="border-t border-zinc-800 pt-4">
          <span className="mb-2 block text-sm font-semibold uppercase tracking-wider text-zinc-500">
            Conversation history
          </span>
          {historyQuery.isLoading ? (
            <div className="flex justify-center py-6">
              <Spinner />
            </div>
          ) : conversations.length === 0 ? (
            <p className="text-sm text-zinc-500">No conversations for this contact yet.</p>
          ) : (
            <ul className="space-y-1.5">
              {conversations.map((c) => (
                <li key={c.id}>
                  <button
                    type="button"
                    onClick={() => openChat(c.id)}
                    className="flex w-full items-center justify-between gap-3 rounded-lg px-2 py-2 text-left transition hover:bg-zinc-800/60"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-sm text-zinc-200">
                        {c.last_message?.body || (c.status === 'open' ? 'Open conversation' : c.status)}
                      </p>
                      <p className="truncate text-xs text-zinc-500">{c.whatsapp_account.display_name}</p>
                    </div>
                    <div className="flex shrink-0 items-center gap-2">
                      {c.unread_count > 0 ? <Badge tone="red">{c.unread_count} new</Badge> : null}
                      <span className="text-xs text-zinc-500">{formatDate(c.last_message_at)}</span>
                    </div>
                  </button>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </ModalDialog>
  )
}

export function ContactsPage() {
  const { session, org } = useSession()
  const queryClient = useQueryClient()
  const orgId = org?.id ?? ''
  const [search, setSearch] = useState('')
  const [dialog, setDialog] = useState<ContactDialogState>(null)
  const [detail, setDetail] = useState<ContactDTO | null>(null)

  const canManageData = session?.isAdmin === true || org?.role === 'owner'

  const contactsQuery = useQuery({
    queryKey: ['contacts', orgId, search],
    queryFn: () => listContacts({ search }),
    enabled: orgId !== '',
  })

  const deleteMutation = useMutation({
    mutationFn: deleteContact,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['contacts', orgId] })
    },
  })

  const contacts = contactsQuery.data?.items ?? []

  function handleDelete(c: ContactDTO) {
    if (!window.confirm(`Delete contact ${c.name || c.phone}?`)) return
    deleteMutation.mutate(c.id)
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="flex items-center justify-between border-b border-zinc-800/80 px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold text-zinc-100">Contacts</h1>
          <p className="text-sm text-zinc-500">
            People your org has chatted with on WhatsApp.
          </p>
        </div>
        {canManageData ? (
          <Button size="sm" onPress={() => setDialog({ mode: 'create' })}>
            <Plus size={14} />
            Add contact
          </Button>
        ) : null}
      </header>

      <div className="px-6 py-4">
        <SearchField
          label="Search contacts"
          placeholder="Search by name or phone…"
          value={search}
          onChange={setSearch}
          className="max-w-sm"
        />
      </div>

      <div className="flex-1 overflow-y-auto px-6 pb-6">
        {contactsQuery.isLoading ? (
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : contacts.length === 0 ? (
          <EmptyState
            icon={<Users size={32} />}
            title={search ? 'No matching contacts' : 'No contacts yet'}
            description={
              search
                ? 'Try a different name or phone number.'
                : 'Contacts are created automatically when someone messages your WhatsApp numbers.'
            }
          />
        ) : (
          <table className="w-full border-collapse text-left text-sm">
            <thead>
              <tr className="border-b border-zinc-800 text-xs uppercase tracking-wider text-zinc-500">
                <th className="py-2.5 pr-4 font-medium">Name</th>
                <th className="py-2.5 font-medium">Phone</th>
                {canManageData ? <th className="py-2.5 pl-4 text-right font-medium">Actions</th> : null}
              </tr>
            </thead>
            <tbody>
              {contacts.map((c) => (
                <tr
                  key={c.id}
                  className="cursor-pointer border-b border-zinc-800/60 transition hover:bg-zinc-900/40"
                  onClick={() => setDetail(c)}
                >
                  <td className="py-3 pr-4">
                    <div className="flex items-center gap-3">
                      <Avatar name={c.name || c.phone} size={32} />
                      <span className="font-medium text-zinc-100">{c.name || c.phone}</span>
                      {c.tags && c.tags.length > 0 ? (
                        <div className="hidden gap-1 sm:flex">
                          {c.tags.slice(0, 2).map((t, i) => (
                            <Badge key={`${t}-${i}`} tone="zinc">
                              {t}
                            </Badge>
                          ))}
                        </div>
                      ) : null}
                    </div>
                  </td>
                  <td className="py-3 text-zinc-400">{c.phone}</td>
                  {canManageData ? (
                    <td className="py-3 pl-4 text-right" onClick={(e) => e.stopPropagation()}>
                      <div className="flex justify-end gap-1">
                        <Button
                          size="icon"
                          variant="ghost"
                          aria-label={`Edit ${c.name || c.phone}`}
                          onPress={() => setDialog({ mode: 'edit', contact: c })}
                        >
                          <Pencil size={16} className="text-zinc-500 hover:text-zinc-200" />
                        </Button>
                        <Button
                          size="icon"
                          variant="ghost"
                          aria-label={`Delete ${c.name || c.phone}`}
                          onPress={() => handleDelete(c)}
                          isDisabled={deleteMutation.isPending}
                        >
                          <Trash2 size={16} className="text-zinc-500 hover:text-red-400" />
                        </Button>
                      </div>
                    </td>
                  ) : null}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {dialog ? <ContactDialog state={dialog} onDone={() => setDialog(null)} /> : null}
      {detail ? <ContactDetailDialog contact={detail} onClose={() => setDetail(null)} /> : null}
    </div>
  )
}
