import { hashColor, initials } from '../../lib/format'

export interface AvatarProps {
  name?: string | null
  size?: number
  className?: string
}

export function Avatar({ name, size = 36, className = '' }: AvatarProps) {
  const bg = hashColor(name)
  return (
    <span
      aria-hidden="true"
      className={`inline-flex shrink-0 select-none items-center justify-center rounded-full font-semibold text-white ${bg} ${className}`}
      style={{ width: size, height: size, fontSize: Math.round(size * 0.4) }}
    >
      {initials(name)}
    </span>
  )
}
