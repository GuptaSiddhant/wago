import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Users } from 'lucide-react'
import { listContacts } from '../../api/client'
import { useSession } from '../../lib/session'
import { Avatar } from '../../components/ui/Avatar'
import { EmptyState } from '../../components/ui/EmptyState'
import { SearchField } from '../../components/ui/SearchField'
import { Spinner } from '../../components/ui/Spinner'

export function ContactsPage() {
  const { org } = useSession()
  const orgId = org?.id ?? ''
  const [search, setSearch] = useState('')

  const contactsQuery = useQuery({
    queryKey: ['contacts', orgId, search],
    queryFn: () => listContacts({ search }),
    enabled: orgId !== '',
  })

  const contacts = contactsQuery.data?.items ?? []

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="border-b border-zinc-800/80 px-6 py-4">
        <h1 className="text-lg font-semibold text-zinc-100">Contacts</h1>
        <p className="text-sm text-zinc-500">
          People your org has chatted with on WhatsApp.
        </p>
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
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
