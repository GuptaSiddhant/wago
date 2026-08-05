import { createFileRoute } from '@tanstack/react-router'
import { AnalyticsPage } from '../../features/analytics/AnalyticsPage'

export const Route = createFileRoute('/_app/analytics')({
  component: AnalyticsPage,
})
