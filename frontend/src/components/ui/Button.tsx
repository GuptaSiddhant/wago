import type { ReactNode } from 'react'
import { Button as RACButton, composeRenderProps } from 'react-aria-components'
import type { ButtonProps } from 'react-aria-components'

type Variant = 'primary' | 'secondary' | 'ghost' | 'danger'
type Size = 'sm' | 'md' | 'icon'

const variants: Record<Variant, string> = {
  primary:
    'bg-emerald-600 text-white hover:bg-emerald-500 disabled:bg-emerald-600/40 shadow-sm',
  secondary:
    'bg-zinc-800 text-zinc-100 hover:bg-zinc-700 disabled:bg-zinc-800/60 border border-zinc-700',
  ghost:
    'bg-transparent text-zinc-300 hover:bg-zinc-800/70 hover:text-zinc-100 disabled:opacity-40',
  danger: 'bg-red-600 text-white hover:bg-red-500 disabled:bg-red-600/40',
}

const sizes: Record<Size, string> = {
  sm: 'h-8 px-3 text-sm rounded-lg gap-1.5',
  md: 'h-10 px-4 text-sm rounded-xl gap-2',
  icon: 'h-8 w-8 rounded-lg',
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
