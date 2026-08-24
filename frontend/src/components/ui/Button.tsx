import type { ReactNode } from 'react'
import { Button as RACButton, composeRenderProps } from 'react-aria-components'
import type { ButtonProps } from 'react-aria-components'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'
type Size = 'sm' | 'md' | 'icon'

const variants: Record<Variant, string> = {
  primary:
    'bg-emerald-700 text-white hover:bg-emerald-600 disabled:bg-emerald-700/40 shadow-sm',
  secondary:
    'bg-panel-strong text-ink hover:bg-panel-strong-hover disabled:bg-panel-strong border border-edge-strong',
  ghost:
    'bg-transparent text-ink-muted hover:bg-panel-strong-hover hover:text-ink disabled:opacity-40',
  danger: 'bg-red-700 text-white hover:bg-red-600 disabled:bg-red-700/40',
}

const sizes: Record<Size, string> = {
  sm: 'h-8 px-3 text-sm rounded-lg gap-1.5',
  md: 'h-10 px-4 text-sm rounded-xl gap-2',
  icon: 'h-9 w-9 rounded-lg',
}

export interface WButtonProps extends Omit<ButtonProps, 'className'> {
  variant?: Variant
  size?: Size
  className?: ButtonProps['className']
  children?: ReactNode
}

export function Button({
  variant = 'primary',
  size = 'md',
  className,
  children,
  ...props
}: WButtonProps) {
  return (
    <RACButton
      {...props}
      className={composeRenderProps(className, (className, { isPressed, isDisabled }) => {
        const state = isDisabled
          ? 'cursor-not-allowed'
          : isPressed
            ? 'scale-[0.98]'
            : ''
        return `inline-flex items-center justify-center font-medium select-none transition outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 ${variants[variant]} ${sizes[size]} ${state} ${className ?? ''}`
      })}
    >
      {children}
    </RACButton>
  )
}
