import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { listAccounts, listTemplates } from '../../api/client'
import { Button } from '../../components/ui/Button'
import { Skeleton } from '../../components/ui/Skeleton'
import { useSession } from '../../lib/session'
import { NewBroadcastForm } from './BroadcastForm'

export function NewBroadcastPage() {
  const navigate = useNavigate()
  const { org } = useSession()
  const orgId = org?.id ?? ''

  const accountsQuery = useQuery({
    queryKey: ['accounts', orgId],
    queryFn: () => listAccounts(),
    enabled: orgId !== '',
  })
  const templatesQuery = useQuery({
    queryKey: ['templates', orgId],
    queryFn: () => listTemplates(),
    enabled: orgId !== '',
  })

  const accounts = accountsQuery.data?.items ?? []
  const templates = templatesQuery.data?.items ?? []
  const loading = accountsQuery.isLoading || templatesQuery.isLoading

  return (
    <div className="mx-auto w-full max-w-2xl px-6 py-8">
      <div className="mb-6 flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          aria-label="Back to broadcasts"
          onPress={() => navigate({ to: '/broadcast' })}
        >
          <ArrowLeft size={18} />
        </Button>
        <div>
          <h1 className="text-lg font-semibold text-ink">New broadcast</h1>
          <p className="text-sm text-ink-faint">
            Send an approved template to your contacts.
          </p>
        </div>
      </div>

      {loading ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : (
        <NewBroadcastForm
          accounts={accounts}
          templates={templates}
          onDone={() => navigate({ to: '/broadcast' })}
        />
      )}
    </div>
  )
}
