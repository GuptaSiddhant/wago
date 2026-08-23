import { createFileRoute } from '@tanstack/react-router'
import { TeamPage } from '../../../features/settings/TeamPage'

export const Route = createFileRoute('/_app/settings/team')({
  staticData: { title: 'Team' },
  component: TeamPage,
})
