import { createFileRoute } from '@tanstack/react-router'
import { OrgPage } from '../../../features/settings/OrgPage'

export const Route = createFileRoute('/_app/settings/org')({
  component: OrgPage,
})