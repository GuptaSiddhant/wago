import { createFileRoute } from '@tanstack/react-router'
import { ContactsPage } from '../../features/contacts/ContactsPage'

export const Route = createFileRoute('/_app/contacts')({
  staticData: { title: 'Contacts' },
  component: ContactsPage,
})
