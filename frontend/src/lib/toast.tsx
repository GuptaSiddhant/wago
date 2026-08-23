import { createContext, useCallback, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { CheckCircle, AlertCircle, AlertTriangle, Info, X } from "lucide-react";

export type ToastType = "success" | "error" | "warning" | "info";

export interface ToastOptions {
  type?: ToastType;
  duration?: number;
  action?: { label: string; onAction: () => void };
}

export interface ToastItem {
  id: string;
  message: string;
  type: ToastType;
  action?: { label: string; onAction: () => void };
}

interface ToastContextValue {
  show: (message: string, options?: ToastOptions) => void;
  success: (message: string) => void;
  error: (message: string) => void;
  info: (message: string) => void;
  warning: (message: string) => void;
}

const ToastContext = createContext<ToastContextValue | null>(null);

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext);
  if (!ctx) {
    throw new Error("useToast must be used within a ToastContextProvider");
  }
  return ctx;
}

const ICONS: Record<ToastType, typeof Info> = {
  success: CheckCircle,
  error: AlertCircle,
  warning: AlertTriangle,
  info: Info,
};

const ICON_COLORS: Record<ToastType, string> = {
  success: "text-emerald-400",
  error: "text-red-400",
  warning: "text-amber-400",
  info: "text-sky-400",
};

const CARD_STYLES: Record<ToastType, string> = {
  success: "border-emerald-500/30",
  error: "border-red-500/30",
  warning: "border-amber-500/30",
  info: "border-sky-500/30",
};

function ToastCard({ toast, onDismiss }: { toast: ToastItem; onDismiss: () => void }) {
  const Icon = ICONS[toast.type];
  return (
    <div
      className={`pointer-events-auto flex items-start gap-3 rounded-xl border ${CARD_STYLES[toast.type]} bg-zinc-900/95 p-3 shadow-xl shadow-black/50`}
    >
      <Icon size={18} className={`mt-0.5 shrink-0 ${ICON_COLORS[toast.type]}`} aria-hidden="true" />
      <p className="min-w-0 flex-1 text-sm text-zinc-100">{toast.message}</p>
      {toast.action && (
        <button
          type="button"
          onClick={() => {
            toast.action?.onAction();
            onDismiss();
          }}
          className="shrink-0 text-xs font-medium text-emerald-400 underline hover:text-emerald-300"
        >
          {toast.action.label}
        </button>
      )}
      <button
        type="button"
        onClick={onDismiss}
        aria-label="Dismiss notification"
        className="shrink-0 rounded p-0.5 text-zinc-500 transition hover:text-zinc-200 focus-visible:outline-2 focus-visible:outline-zinc-400"
      >
        <X size={16} aria-hidden="true" />
      </button>
    </div>
  );
}

export function ToastContextProvider({ children }: { children: ReactNode }) {
  const [queue, setQueue] = useState<ToastItem[]>([]);

  const dismiss = useCallback((id: string) => {
    setQueue((prev) => prev.filter((t) => t.id !== id));
  }, []);

  const show = useCallback(
    (message: string, options?: ToastOptions) => {
      const id = crypto.randomUUID();
      setQueue((prev) => [
        ...prev.slice(-4),
        { id, message, type: options?.type ?? "info", action: options?.action },
      ]);
      if (options?.action === undefined || options.duration !== undefined) {
        const duration =
          options?.duration ??
          (options?.type === "error" ? 8000 : 5000);
        window.setTimeout(() => dismiss(id), duration);
      }
    },
    [dismiss]
  );

  const value = useMemo<ToastContextValue>(
    () => ({
      show,
      success: (message) => show(message, { type: "success" }),
      error: (message) => show(message, { type: "error" }),
      info: (message) => show(message, { type: "info" }),
      warning: (message) => show(message, { type: "warning" }),
    }),
    [show]
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div
        role="region"
        aria-label="Notifications"
        className="pointer-events-none fixed bottom-4 right-4 z-[100] flex w-80 flex-col gap-2 md:w-96"
      >
        <div aria-live="polite" aria-relevant="additions text" className="flex flex-col gap-2">
          {queue.map((t) => (
            <ToastCard key={t.id} toast={t} onDismiss={() => dismiss(t.id)} />
          ))}
        </div>
      </div>
    </ToastContext.Provider>
  );
}
