import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { BarChart3, RefreshCw, TriangleAlert } from 'lucide-react'
import { analytics } from '../../api/client'
import { useSession } from '../../lib/session'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { SelectField } from '../../components/ui/Select'
import { Spinner } from '../../components/ui/Spinner'
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

function CategoryBreakdown({ categories=[] }: { categories: AnalyticsCategory[] }) {
  const max = Math.max(...categories.map((c) => c.cost), 0)
  return (
    <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-5">
      <h2 className="text-sm font-semibold text-zinc-100">By category</h2>
      <ul className="mt-4 space-y-3">
        {categories.length === 0 ? (
          <li className="text-sm text-zinc-500">No data for this period</li>
        ) : (
          categories.map((c) => (
            <li key={c.category}>
              <div className="mb-1 flex items-center justify-between text-sm">
                <span className="text-zinc-300">{c.category}</span>
                <span className="text-zinc-400">
                  {c.conversations} · {formatCost(c.cost)}
                </span>
              </div>
              <div className="h-2 overflow-hidden rounded-full bg-zinc-800">
                <div
                  className="h-full rounded-full bg-emerald-500"
                  style={{ width: max > 0 ? `${Math.max(4, (c.cost / max) * 100)}%` : '0%' }}
                />
              </div>
            </li>
          ))
        )}
      </ul>
    </div>
  )
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
      <header className="flex items-center justify-between gap-3 border-b border-zinc-800/80 px-6 py-4">
        <div>
          <h1 className="text-lg font-semibold text-zinc-100">Analytics</h1>
          <p className="text-sm text-zinc-500">
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
          <div className="flex justify-center py-16">
            <Spinner />
          </div>
        ) : data ? (
          <div className="space-y-6">
            {!hasNumbers ? (
              <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-5 text-sm text-zinc-400">
                No WhatsApp numbers are connected yet. Connect a number on the{' '}
                <span className="font-medium text-zinc-200">Numbers</span> page to see usage
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
                  <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-5">
                    <div className="text-xs uppercase tracking-wider text-zinc-500">Conversations</div>
                    <div className="mt-1 text-3xl font-semibold text-zinc-100">
                      {data.totals.conversations.toLocaleString()}
                    </div>
                  </div>
                  <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-5">
                    <div className="text-xs uppercase tracking-wider text-zinc-500">Meta cost</div>
                    <div className="mt-1 text-3xl font-semibold text-zinc-100">
                      {formatCost(data.totals.cost)}
                    </div>
                  </div>
                </div>

                <div className="grid gap-4 lg:grid-cols-2">
                  <div className="rounded-xl border border-zinc-800 bg-zinc-900/50 p-5">
                    <h2 className="text-sm font-semibold text-zinc-100">Per number</h2>
                    <ul className="mt-4 divide-y divide-zinc-800">
                      {data.accounts.map((a) => (
                        <li key={a.id} className="flex items-center justify-between gap-2 py-2.5">
                          <div className="min-w-0">
                            <div className="truncate text-sm text-zinc-200">
                              {a.display_name || a.phone_number_id}
                            </div>
                            <div className="truncate text-xs text-zinc-500">
                              {a.conversations.toLocaleString()} conversations
                            </div>
                          </div>
                          <span className="text-sm font-medium text-zinc-300">
                            {formatCost(a.cost)}
                          </span>
                        </li>
                      ))}
                    </ul>
                  </div>
                  <CategoryBreakdown categories={data.categories} />
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
