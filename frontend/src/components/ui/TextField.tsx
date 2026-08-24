import type { ReactNode } from 'react'
import { TextField as RACTextField, Input, Label, FieldError, Text } from 'react-aria-components'
import type { TextFieldProps, InputProps, InputRenderProps } from 'react-aria-components'

export interface WTextFieldProps
  extends Omit<TextFieldProps, 'className' | 'children'> {
  label?: ReactNode
  inputClassName?: string
  placeholder?: string
  type?: InputProps['type']
  autoComplete?: string
  isRequired?: boolean
  description?: ReactNode
  errorMessage?: ReactNode
}

const inputBase =
  'w-full h-10 px-3 rounded-xl bg-panel border border-edge-strong text-ink text-sm placeholder:text-ink-faint outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/30 disabled:opacity-50'

export function TextField({
  label,
  inputClassName = '',
  placeholder,
  type = 'text',
  autoComplete,
  isRequired,
  description,
  errorMessage,
  ...props
}: WTextFieldProps) {
  return (
    <RACTextField
      {...props}
      isRequired={isRequired}
      className="flex flex-col gap-1.5"
    >
      {label != null ? (
        <Label className="text-sm font-medium text-ink-muted">{label}</Label>
      ) : null}
      <Input
        type={type}
        placeholder={placeholder}
        autoComplete={autoComplete}
        aria-label={typeof label === 'string' ? label : undefined}
        className={({ isFocused, isDisabled }: InputRenderProps) => {
          const ring = isFocused ? 'border-emerald-500 ring-2 ring-emerald-500/30' : ''
          return `${inputBase} ${ring} ${isDisabled ? 'opacity-50' : ''} ${inputClassName}`
        }}
      />
      {description != null ? (
        <Text slot="description" className="text-xs text-ink-faint">
          {description}
        </Text>
      ) : null}
      <FieldError className="text-xs text-red-400">{errorMessage}</FieldError>
    </RACTextField>
  )
}
