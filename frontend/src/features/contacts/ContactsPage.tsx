import { useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Pencil, Plus, Trash2, Users } from 'lucide-react'
import {
  createContact,
  deleteContact,
  listContacts,
  updateContact,
} from '../../api/client'
import { useSession } from '../../lib/session'
import { Avatar } from '../../components/ui/Avatar'
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

export function ContactsPage() {
  const { session, org } = useSession()
  const queryClient = useQueryClient()
  const orgId = org?.id ?? ''
  const [search, setSearch] = useState('')
  const [dialog, setDialog] = useState<ContactDialogState>(null)

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
                <tr key={c.id} className="border-b border-zinc-800/60 transition hover:bg-zinc-900/40">
                  <td className="py-3 pr-4">
                    <div className="flex items-center gap-3">
                      <Avatar name={c.name || c.phone} size={32} />
                      <span className="font-medium text-zinc-100">{c.name || c.phone}</span>
                    </div>
                  </td>
                  <td className="py-3 text-zinc-400">{c.phone}</td>
                  {canManageData ? (
                    <td className="py-3 pl-4 text-right">
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
    </div>
  )
}
