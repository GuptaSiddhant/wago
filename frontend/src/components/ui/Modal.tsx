import type { ReactNode } from 'react'
import {
  Dialog,
  Heading,
  Modal,
  ModalOverlay,
} from 'react-aria-components'
import type { ModalOverlayProps, ModalRenderProps } from 'react-aria-components'

export interface ModalDialogProps extends ModalOverlayProps {
  title?: ReactNode
  children: ReactNode
}

export function ModalDialog({ title, children, ...props }: ModalDialogProps) {
  return (
    <ModalOverlay
      {...props}
      className={({ isEntering, isExiting }: ModalRenderProps) => {
        const animate = isEntering
          ? 'animate-[wago-fade-in_200ms_ease-out]'
          : isExiting
            ? 'animate-[wago-fade-out_150ms_ease-in]'
            : ''
        return `fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm ${animate}`
      }}
    >
      <Modal
        className={({ isEntering, isExiting }: ModalRenderProps) => {
          const animate = isEntering
            ? 'animate-[wago-zoom-in_200ms_ease-out]'
            : isExiting
              ? 'animate-[wago-zoom-out_150ms_ease-in]'
              : ''
          return `max-h-[calc(var(--visual-viewport-height)*0.9)] w-full max-w-[min(480px,calc(100vw-2rem))] overflow-y-auto rounded-2xl border border-edge-strong bg-panel p-6 shadow-2xl shadow-black/50 outline-none ${animate}`
        }}
      >
        <Dialog className="flex flex-col gap-4 outline-none">
          {title != null ? (
            <Heading slot="title" className="text-lg font-semibold text-ink">
              {title}
            </Heading>
          ) : null}
          {children}
        </Dialog>
      </Modal>
    </ModalOverlay>
  )
}
