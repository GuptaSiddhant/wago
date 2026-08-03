import type { ReactNode } from 'react'
import {
  Button,
  FieldError,
  Label,
  ListBox,
  ListBoxItem,
  Popover,
  Select,
  SelectValue,
  Text,
} from 'react-aria-components'
import type {
  Key,
  ListBoxItemRenderProps,
  SelectValueRenderProps,
} from 'react-aria-components'
import { Check, ChevronsUpDown } from 'lucide-react'

export interface SelectOption {
  id: string
  label: string
}

export interface WSelectProps {
  label?: ReactNode
  options: SelectOption[]
  selectedKey?: Key | null
  onSelectionChange?: (key: Key | null) => void
  placeholder?: string
  isDisabled?: boolean
  className?: string
  errorMessage?: ReactNode
  description?: ReactNode
}

export function SelectField({
  label,
  options,
  selectedKey,
  onSelectionChange,
  placeholder = 'Select…',
  isDisabled,
  className = '',
  errorMessage,
  description,
}: WSelectProps) {
  return (
    <Select
      selectedKey={selectedKey}
      onSelectionChange={(key: Key) => onSelectionChange?.(key)}
      isDisabled={isDisabled}
      aria-label={typeof label === 'string' ? label : undefined}
      className={`flex flex-col gap-1.5 ${className}`}
    >
      {label != null ? (
        <Label className="text-sm font-medium text-zinc-300">{label}</Label>
      ) : null}
      <Button className="inline-flex h-10 items-center justify-between gap-2 rounded-xl border border-zinc-700 bg-zinc-900 px-3 text-sm text-zinc-100 outline-none transition hover:bg-zinc-800 focus-visible:ring-2 focus-visible:ring-emerald-500/50 data-[disabled]:opacity-50">
        <SelectValue className="truncate text-left text-zinc-100 placeholder:text-zinc-500 [&>[slot=placeholder]]:text-zinc-500">
          {({ selectedText }: SelectValueRenderProps<unknown>) =>
            selectedText ?? placeholder
          }
        </SelectValue>
        <ChevronsUpDown size={14} className="shrink-0 text-zinc-500" />
      </Button>
      {description != null ? (
        <Text slot="description" className="text-xs text-zinc-500">
          {description}
        </Text>
      ) : null}
      <FieldError className="text-xs text-red-400">{errorMessage}</FieldError>
      <Popover className="min-w-[var(--trigger-width)] rounded-xl border border-zinc-700 bg-zinc-900 p-1 shadow-xl shadow-black/40">
        <ListBox className="max-h-72 overflow-auto outline-none" items={options}>
          {(option: SelectOption) => (
            <ListBoxItem
              id={option.id}
              textValue={option.label}
              className={({ isFocused, isSelected }: ListBoxItemRenderProps) =>
                `flex cursor-pointer items-center justify-between gap-3 rounded-lg px-3 py-2 text-sm outline-none ${
                  isFocused ? 'bg-zinc-800' : ''
                } ${isSelected ? 'text-emerald-400' : 'text-zinc-200'}`
              }
            >
              {({ isSelected }: ListBoxItemRenderProps) => (
                <>
                  <span className="truncate">{option.label}</span>
                  {isSelected ? <Check size={14} /> : null}
                </>
              )}
            </ListBoxItem>
          )}
        </ListBox>
      </Popover>
    </Select>
  )
}
