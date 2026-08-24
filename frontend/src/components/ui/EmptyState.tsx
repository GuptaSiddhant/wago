import type { ReactNode } from 'react'

export interface EmptyStateProps {
  icon?: ReactNode
  title: string
  description?: ReactNode
  action?: ReactNode
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
      {icon ? <div className="text-ink-faint">{icon}</div> : null}
      <div>
        <h3 className="text-sm font-semibold text-ink">{title}</h3>
        {description ? (
          <p className="mt-1 text-sm text-ink-faint">{description}</p>
        ) : null}
      </div>
      {action}
    </div>
  )
}
