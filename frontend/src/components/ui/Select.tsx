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
  ariaLabel?: string
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
  ariaLabel,
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
      onSelectionChange={(key: Key | null) => onSelectionChange?.(key)}
      isDisabled={isDisabled}
      aria-label={ariaLabel ?? (typeof label === 'string' ? label : undefined)}
      className={`flex flex-col gap-1.5 ${className}`}
    >
      {label != null ? (
        <Label className="text-sm font-medium text-ink-muted">{label}</Label>
      ) : null}
      <Button className="inline-flex h-10 items-center justify-between gap-2 rounded-xl border border-edge-strong bg-panel px-3 text-sm text-ink outline-none transition hover:bg-panel-strong-hover focus-visible:ring-2 focus-visible:ring-emerald-500/50 data-[disabled]:opacity-50">
        <SelectValue className="truncate text-left text-ink placeholder:text-ink-faint [&>[slot=placeholder]]:text-ink-faint">
          {({ selectedText }: SelectValueRenderProps<unknown>) =>
            selectedText || placeholder
          }
        </SelectValue>
        <ChevronsUpDown size={14} className="shrink-0 text-ink-faint" />
      </Button>
      {description != null ? (
        <Text slot="description" className="text-xs text-ink-faint">
          {description}
        </Text>
      ) : null}
      <FieldError className="text-xs text-red-400">{errorMessage}</FieldError>
      <Popover className="min-w-[var(--trigger-width)] rounded-xl border border-edge-strong bg-panel p-1 shadow-xl shadow-black/40">
        <ListBox className="max-h-72 overflow-auto outline-none" items={options}>
          {(option: SelectOption) => (
            <ListBoxItem
              id={option.id}
              textValue={option.label}
              className={({ isFocused, isSelected }: ListBoxItemRenderProps) =>
                `flex cursor-pointer items-center justify-between gap-3 rounded-lg px-3 py-2 text-sm outline-none ${
                  isFocused ? 'bg-panel-strong' : ''
                } ${isSelected ? 'text-emerald-400' : 'text-ink'}`
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
