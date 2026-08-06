// PocketBase serialises datetimes as "2026-08-03 15:26:53.837Z" (space
// separator), which native JS Date parsing handles inconsistently. Normalise
// to ISO before constructing a Date.
export function parsePBDate(value: string | undefined | null): Date | null {
  if (!value) return null
  const iso = value.replace(' ', 'T')
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? null : d
}

// Formats a PB datetime as a short local time ("14:05") or an em dash if empty.
export function formatTime(value: string | undefined | null): string {
  const d = parsePBDate(value)
  if (!d) return '—'
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

// Formats a PB datetime as a readable "Apr 3, 2026" date or an em dash if empty.
export function formatDate(value: string | undefined | null): string {
  const d = parsePBDate(value)
  if (!d) return '—'
  return d.toLocaleDateString([], { month: 'short', day: 'numeric', year: 'numeric' })
}

// Renders a PB datetime as a relative "3m / 2h / 5d" age, falling back to the
// full date for anything more than a week old.
export function timeAgo(value: string | undefined | null): string {
  const d = parsePBDate(value)
  if (!d) return '—'
  const seconds = Math.round((Date.now() - d.getTime()) / 1000)
  if (seconds < 60) return 'now'
  const minutes = Math.round(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.round(minutes / 60)
  if (hours < 24) return `${hours}h`
  const days = Math.round(hours / 24)
  if (days < 7) return `${days}d`
  return formatDate(value)
}

// First letters of a person's first and last name (or the first two letters of
// a single-word name), used for avatar placeholders.
export function initials(name: string | undefined | null, fallback = '?'): string {
  if (!name) return fallback
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return fallback
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

const avatarPalette = [
  'bg-emerald-600',
  'bg-blue-600',
  'bg-violet-600',
  'bg-amber-600',
  'bg-rose-600',
  'bg-teal-600',
  'bg-indigo-600',
]

// Deterministically picks an avatar background color from a name.
export function hashColor(name: string | undefined | null): string {
  if (!name) return avatarPalette[0]
  let hash = 0
  for (let i = 0; i < name.length; i++) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0
  }
  return avatarPalette[hash % avatarPalette.length]
}

export type Tone = 'green' | 'zinc' | 'red' | 'blue' | 'amber'

// Maps a message/record status to a Badge tone for a consistent visual cue.
export function statusTone(status: string): Tone {
  switch (status) {
    case 'open':
    case 'read':
      return 'green'
    case 'sent':
    case 'delivered':
      return 'blue'
    case 'failed':
      return 'red'
    default:
      return 'zinc'
  }
}
