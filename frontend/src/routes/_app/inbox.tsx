import { createFileRoute } from '@tanstack/react-router'
import { InboxPage } from '../../features/inbox/InboxPage'

export const Route = createFileRoute('/_app/inbox')({
  staticData: { title: 'Inbox' },
  validateSearch: (search: Record<string, unknown>) => ({
    ...(typeof search.conv === 'string' ? { conv: search.conv } : {}),
  }),
  component: InboxPage,
})
