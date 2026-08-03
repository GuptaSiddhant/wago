import { SearchField as RACSearchField, Input, Button } from 'react-aria-components'
import { Search, X } from 'lucide-react'

export interface WSearchFieldProps {
  label?: string
  placeholder?: string
  value?: string
  onChange?: (value: string) => void
  className?: string
  inputClassName?: string
}

export function SearchField({
  label,
  placeholder = 'Search…',
  value,
  onChange,
  className = '',
  inputClassName = '',
}: WSearchFieldProps) {
  return (
    <RACSearchField
      value={value}
      onChange={onChange}
      aria-label={label}
      className={`relative flex items-center ${className}`}
    >
      <Search size={15} className="pointer-events-none absolute left-3 text-zinc-500" />
      <Input
        placeholder={placeholder}
        className={`h-9 w-full rounded-lg border border-transparent bg-zinc-900 pl-9 pr-9 text-sm text-zinc-100 placeholder:text-zinc-500 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/30 ${inputClassName}`}
      />
      <Button
        slot="clear"
        aria-label="Clear search"
        className="absolute right-2 rounded p-0.5 text-zinc-500 outline-none hover:text-zinc-200 focus-visible:ring-2 focus-visible:ring-emerald-500/50"
      >
        <X size={14} />
      </Button>
    </RACSearchField>
  )
}
