import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { BarChart3, RefreshCw, TriangleAlert } from 'lucide-react'
import { analytics } from '../../api/client'
import { useSession } from '../../lib/session'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { HBarChart } from '../../components/ui/HBarChart'
import { SelectField } from '../../components/ui/Select'
import { Skeleton } from '../../components/ui/Skeleton'
import type { AnalyticsCategory } from '../../api/types'

const ranges = [
  { id: '7d', label: 'Last 7 days' },
  { id: '30d', label: 'Last 30 days' },
  { id: '90d', label: 'Last 90 days' },
]

function formatCost(cost: number): string {
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 2,
  }).format(cost)
}

function formatCount(value: number): string {
  return value.toLocaleString()
}

export function AnalyticsPage() {
  const { org } = useSession()
  const orgId = org?.id ?? ''
  const [range, setRange] = useState('30d')

  const analyticsQuery = useQuery({
    queryKey: ['analytics', orgId, range],
    queryFn: () => analytics(range),
    enabled: orgId !== '',
  })

  const data = analyticsQuery.data
  const hasNumbers = (data?.accounts?.length ?? 0) > 0

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <header className="flex items-center justify-between gap-3 border-b border-edge px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold text-ink">Analytics</h1>
          <p className="text-sm text-ink-faint">
            WhatsApp usage and cost for this organization, from Meta&apos;s conversation analytics.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <SelectField
            options={ranges}
            selectedKey={range}
            onSelectionChange={(key) => {
              if (key != null) setRange(String(key))
            }}
          />
          <Button
            size="sm"
            variant="ghost"
            aria-label="Refresh"
            onPress={() => void analyticsQuery.refetch()}
            isDisabled={analyticsQuery.isFetching}
          >
            <RefreshCw size={15} className={analyticsQuery.isFetching ? 'animate-spin' : ''} />
          </Button>
        </div>
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-6">
        {analyticsQuery.isLoading ? (
          <div className="space-y-6">
            <div className="grid grid-cols-2 gap-3 md:grid-cols-4">
              {Array.from({ length: 4 }).map((_, i) => (
                <Skeleton key={i} className="h-20 rounded-xl" />
              ))}
            </div>
            <Skeleton className="h-56 rounded-xl" />
          </div>
        ) : data ? (
          <div className="space-y-6">
            {!hasNumbers ? (
              <div className="rounded-xl border border-edge bg-panel p-5 text-sm text-ink-muted">
                No WhatsApp numbers are connected yet. Connect a number on the{' '}
                <span className="font-medium text-ink">Numbers</span> page to see usage
                analytics.
              </div>
            ) : null}

            {data.errors?.length > 0 ? (
              <div className="rounded-xl border border-amber-500/20 bg-amber-500/5 p-4">
                <div className="flex items-center gap-2 text-sm font-medium text-amber-300">
                  <TriangleAlert size={15} />
                  Some numbers couldn&apos;t report analytics
                </div>
                <ul className="mt-2 list-inside list-disc space-y-1 text-xs text-amber-200/80">
                  {data.errors.map((err, i) => (
                    <li key={i}>{err}</li>
                  ))}
                </ul>
              </div>
            ) : null}

            {hasNumbers ? (
              <>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="rounded-xl border border-edge bg-panel p-5">
                    <div className="text-xs uppercase tracking-wider text-ink-faint">Conversations</div>
                    <div className="mt-1 text-3xl font-semibold text-ink">
                      {data.totals.conversations.toLocaleString()}
                    </div>
                  </div>
                  <div className="rounded-xl border border-edge bg-panel p-5">
                    <div className="text-xs uppercase tracking-wider text-ink-faint">Meta cost</div>
                    <div className="mt-1 text-3xl font-semibold text-ink">
                      {formatCost(data.totals.cost)}
                    </div>
                  </div>
                </div>

                <div className="grid gap-4 lg:grid-cols-2">
                  <HBarChart
                    title="Conversations by number"
                    data={data.accounts.map((a) => ({
                      label: a.display_name || a.phone_number_id,
                      value: a.conversations,
                    }))}
                    formatValue={formatCount}
                  />
                  <HBarChart
                    title="Cost by number"
                    data={data.accounts.map((a) => ({
                      label: a.display_name || a.phone_number_id,
                      value: a.cost,
                    }))}
                    formatValue={formatCost}
                  />
                  <HBarChart
                    title="Conversations by category"
                    data={data.categories.map((c: AnalyticsCategory) => ({
                      label: c.category,
                      value: c.conversations,
                    }))}
                    formatValue={formatCount}
                  />
                  <HBarChart
                    title="Cost by category"
                    data={data.categories.map((c: AnalyticsCategory) => ({
                      label: c.category,
                      value: c.cost,
                    }))}
                    formatValue={formatCost}
                  />
                </div>
              </>
            ) : (
              <EmptyState
                icon={<BarChart3 size={32} />}
                title="No analytics to show"
                description="Add a WABA ID to your connected numbers to fetch usage and cost from Meta."
              />
            )}
          </div>
        ) : null}
      </div>
    </div>
  )
}
