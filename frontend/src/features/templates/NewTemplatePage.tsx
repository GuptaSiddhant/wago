import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { listAccounts } from '../../api/client'
import { Button } from '../../components/ui/Button'
import { Skeleton } from '../../components/ui/Skeleton'
import { useSession } from '../../lib/session'
import { NewTemplateForm } from './TemplateForm'

export function NewTemplatePage() {
  const navigate = useNavigate()
  const { org } = useSession()
  const orgId = org?.id ?? ''

  const accountsQuery = useQuery({
    queryKey: ['accounts', orgId],
    queryFn: listAccounts,
    enabled: orgId !== '',
  })

  const accounts = accountsQuery.data?.items ?? []

  return (
    <div className="mx-auto w-full max-w-2xl px-6 py-8">
      <div className="mb-6 flex items-center gap-3">
        <Button
          variant="ghost"
          size="icon"
          aria-label="Back to templates"
          onPress={() => navigate({ to: '/templates' })}
        >
          <ArrowLeft size={18} />
        </Button>
        <div>
          <h1 className="text-lg font-semibold text-ink">Create message template</h1>
          <p className="text-sm text-ink-faint">
            Submitted to Meta for review. Usable once approved.
          </p>
        </div>
      </div>

      {accountsQuery.isLoading ? (
        <div className="flex flex-col gap-3">
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-24 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : (
        <NewTemplateForm
          accounts={accounts}
          onDone={() => navigate({ to: '/templates' })}
        />
      )}
    </div>
  )
}
