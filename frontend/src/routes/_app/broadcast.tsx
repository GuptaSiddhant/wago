import { createFileRoute } from '@tanstack/react-router'
import { BroadcastPage } from '../../features/broadcast/BroadcastPage'

export const Route = createFileRoute('/_app/broadcast')({
  component: BroadcastPage,
})
