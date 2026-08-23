import { createFileRoute } from '@tanstack/react-router'
import { NumbersPage } from '../../../features/settings/NumbersPage'

export const Route = createFileRoute('/_app/settings/numbers')({
  staticData: { title: 'WhatsApp Numbers' },
  component: NumbersPage,
})
