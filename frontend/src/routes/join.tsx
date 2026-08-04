import { createFileRoute } from '@tanstack/react-router'
import { JoinPage } from '../features/auth/JoinPage'

export const Route = createFileRoute('/join')({
  validateSearch: (search: Record<string, unknown>) => ({
    token: typeof search.token === 'string' ? search.token : undefined,
  }),
  component: JoinPage,
})