export interface HBarDatum {
  label: string
  value: number
}

export function HBarChart({
  title,
  unit,
  data,
  formatValue,
}: {
  title: string
  unit?: string
  data: HBarDatum[]
  formatValue: (value: number) => string
}) {
  const max = Math.max(...data.map((d) => d.value), 0)
  return (
    <div className="rounded-xl border border-edge bg-panel p-5">
      <h2 className="text-sm font-semibold text-ink">{title}</h2>
      <ul className="mt-4 space-y-3">
        {data.length === 0 ? (
          <li className="text-sm text-ink-faint">No data for this period</li>
        ) : (
          data.map((d) => {
            const pct = max > 0 ? Math.max(4, (d.value / max) * 100) : 0
            return (
              <li key={d.label}>
                <div className="mb-1 flex items-baseline justify-between gap-3 text-sm">
                  <span className="min-w-0 truncate text-ink-muted" title={d.label}>
                    {d.label}
                  </span>
                  <span className="shrink-0 tabular-nums text-ink-muted">
                    {formatValue(d.value)}
                    {unit != null && d.value === max ? (
                      <span className="ml-1 text-xs text-emerald-400">peak</span>
                    ) : null}
                  </span>
                </div>
                <div
                  role="meter"
                  aria-valuemin={0}
                  aria-valuemax={max}
                  aria-valuenow={d.value}
                  aria-label={`${d.label}: ${formatValue(d.value)}${unit ? ` ${unit}` : ''}`}
                  className="h-2 overflow-hidden rounded-full bg-panel-strong"
                >
                  <div
                    className="h-full rounded-full bg-emerald-500"
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </li>
            )
          })
        )}
      </ul>
    </div>
  )
}
