import type { ReactNode } from 'react'
import type { Tone } from '../../lib/format'

const tones: Record<Tone, string> = {
  green: 'bg-emerald-500/10 text-emerald-400 ring-emerald-500/25',
  zinc: 'bg-zinc-500/10 text-zinc-400 ring-zinc-500/25',
  red: 'bg-red-500/10 text-red-400 ring-red-500/25',
  blue: 'bg-blue-500/10 text-blue-400 ring-blue-500/25',
  amber: 'bg-amber-500/10 text-amber-400 ring-amber-500/25',
}

export interface BadgeProps {
  tone?: Tone
  children: ReactNode
  className?: string
}

export function Badge({ tone = 'zinc', children, className = '' }: BadgeProps) {
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ring-1 ring-inset ${tones[tone]} ${className}`}
    >
      {children}
    </span>
  )
}
