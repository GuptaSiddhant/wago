import { createFileRoute } from '@tanstack/react-router'
import { ConfigPage } from '../../../features/settings/ConfigPage'

export const Route = createFileRoute('/_app/settings/config')({
  staticData: { title: 'Instance' },
  component: ConfigPage,
})
