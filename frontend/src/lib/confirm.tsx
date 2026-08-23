import { createContext, useCallback, useContext, useRef, useState } from "react";
import type { ReactNode } from "react";
import {
  Dialog,
  Heading,
  Modal,
  ModalOverlay,
} from "react-aria-components";
import type { ModalOverlayProps, ModalRenderProps } from "react-aria-components";
import { Button } from "../components/ui/Button";

export interface ConfirmOptions {
  title: string;
  message: string;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: "danger" | "default";
}

interface ConfirmContextValue {
  confirm: (options: ConfirmOptions) => Promise<boolean>;
}

const ConfirmContext = createContext<ConfirmContextValue | null>(null);

export function useConfirm(): ConfirmContextValue["confirm"] {
  const ctx = useContext(ConfirmContext);
  if (!ctx) {
    throw new Error("useConfirm must be used within a ConfirmProvider");
  }
  return ctx.confirm;
}

export function ConfirmProvider({ children }: { children: ReactNode }) {
  const [req, setReq] = useState<ConfirmOptions | null>(null);
  const resolverRef = useRef<((confirmed: boolean) => void) | null>(null);

  const confirm = useCallback(
    (options: ConfirmOptions) =>
      new Promise<boolean>((resolve) => {
        resolverRef.current?.(false);
        resolverRef.current = resolve;
        setReq(options);
      }),
    []
  );

  const finish = useCallback((confirmed: boolean) => {
    const resolve = resolverRef.current;
    resolverRef.current = null;
    setReq(null);
    resolve?.(confirmed);
  }, []);

  const handleOpenChange: ModalOverlayProps["onOpenChange"] = useCallback(
    (isOpen) => {
      if (!isOpen) finish(false);
    },
    [finish]
  );

  return (
    <ConfirmContext.Provider value={{ confirm }}>
      {children}
      <ModalOverlay
        isDismissable
        isOpen={req !== null}
        onOpenChange={handleOpenChange}
        className={({ isEntering, isExiting }: ModalRenderProps) =>
          `fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm ${
            isEntering
              ? "animate-[wago-fade-in_200ms_ease-out]"
              : isExiting
                ? "animate-[wago-fade-out_150ms_ease-in]"
                : ""
          }`
        }
      >
        <Modal
          className={({ isEntering, isExiting }: ModalRenderProps) =>
            `w-full max-w-md rounded-2xl border border-zinc-700/80 bg-zinc-900 p-6 shadow-2xl shadow-black/50 outline-none ${
              isEntering
                ? "animate-[wago-zoom-in_200ms_ease-out]"
                : isExiting
                  ? "animate-[wago-zoom-out_150ms_ease-in]"
                  : ""
            }`
          }
        >
          <Dialog
            role="alertdialog"
            className="flex flex-col gap-3 outline-none"
          >
            <Heading slot="title" className="text-lg font-semibold text-zinc-100">
              {req?.title}
            </Heading>
            <p slot="subtitle" className="text-sm leading-relaxed text-zinc-400">
              {req?.message}
            </p>
            <div className="mt-2 flex justify-end gap-2">
              <Button variant="secondary" onPress={() => finish(false)}>
                {req?.cancelLabel ?? "Cancel"}
              </Button>
              <Button
                variant={req?.tone === "default" ? "primary" : "danger"}
                slot={req?.confirmLabel ?? "Confirm"}
                onPress={() => finish(true)}
              >
                {req?.confirmLabel ?? "Confirm"}
              </Button>
            </div>
          </Dialog>
        </Modal>
      </ModalOverlay>
    </ConfirmContext.Provider>
  );
}
