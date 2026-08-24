import { createFileRoute } from '@tanstack/react-router'
import { NewBroadcastPage } from '../../../features/broadcast/NewBroadcastPage'

export const Route = createFileRoute('/_app/broadcast/new')({
  staticData: { title: 'New broadcast' },
  component: NewBroadcastPage,
})
