import type { ReactNode } from 'react'
import { Dialog, Heading, Modal, ModalOverlay } from 'react-aria-components'
import type { ModalOverlayProps, ModalRenderProps } from 'react-aria-components'

export interface DrawerProps extends ModalOverlayProps {
  title?: ReactNode
  children: ReactNode
}

export function Drawer({ title, children, ...props }: DrawerProps) {
  return (
    <ModalOverlay
      {...props}
      className={({ isEntering, isExiting }: ModalRenderProps) =>
        `fixed inset-0 z-50 flex justify-end bg-black/60 backdrop-blur-sm ${
          isEntering
            ? 'animate-[wago-fade-in_200ms_ease-out]'
            : isExiting
              ? 'animate-[wago-fade-out_150ms_ease-in]'
              : ''
        }`
      }
    >
      <Modal
        className={({ isEntering, isExiting }: ModalRenderProps) =>
          `h-full max-h-none w-full max-w-md overflow-y-auto border-l border-edge bg-panel p-6 shadow-2xl shadow-black/50 outline-none ${
            isEntering
              ? 'animate-[wago-slide-in-right_250ms_ease-out]'
              : isExiting
                ? 'animate-[wago-slide-out-right_200ms_ease-in]'
                : ''
          }`
        }
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
