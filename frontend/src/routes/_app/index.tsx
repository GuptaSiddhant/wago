import { createFileRoute } from '@tanstack/react-router'
import { AIHomePage } from '../../features/home/AIHomePage'

export const Route = createFileRoute('/_app/')({
  component: AIHomePage,
})