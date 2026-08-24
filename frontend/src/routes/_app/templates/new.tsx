import { createFileRoute } from '@tanstack/react-router'
import { NewTemplatePage } from '../../../features/templates/NewTemplatePage'

export const Route = createFileRoute('/_app/templates/new')({
  staticData: { title: 'New template' },
  component: NewTemplatePage,
})
