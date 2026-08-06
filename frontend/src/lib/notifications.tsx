import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import type { ReactNode } from "react";
import { useNavigate, useRouterState } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { listNotifications, markNotificationsRead, sendPresence, unreadNotificationCount } from "../api/client";
import { queryClient } from "../api/queryClient";
import type { NotificationDTO } from "../api/types";
import { syncPushSubscription } from "./pwa";
import { useSession } from "./session";

const POLL_MS = 15_000;
const PRESENCE_HEARTBEAT_MS = 60_000;

// In-app notification state: the fetched notification list, the unread count,
// a flag controlling whether the dropdown is open, and actions to refresh or
// clear the list.
interface NotificationsContextValue {
  items: NotificationDTO[];
  unread: number;
  open: boolean;
  setOpen: (open: boolean) => void;
  refresh: () => void;
  markAllRead: () => Promise<void>;
}

const NotificationsContext = createContext<NotificationsContextValue | null>(null);

function requestPermissionOnce(onGranted: () => void) {
  if (typeof window === "undefined" || !("Notification" in window)) return;
  if (Notification.permission === "default") {
    // Prompt shortly after login (a user gesture has just happened); register
    // the device for Web Push once the user allows notifications.
    window.setTimeout(() => {
      void Notification.requestPermission().then((permission) => {
        if (permission === "granted") onGranted();
      });
    }, 1000);
  }
}

export function NotificationsProvider({ children }: { children: ReactNode }) {
  const { session, org } = useSession();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);
  const seenRef = useRef<Set<string> | null>(null);
  const activeConvRef = useRef<string | undefined>(undefined);

  const orgId = session?.orgs?.length ? org?.id : undefined;

  // Active conversation currently open in the inbox (suppresses duplicate pushes).
  activeConvRef.current = useRouterState({
    select: (s) => (s.location.search as Record<string, string | undefined>).conv,
  });

  useEffect(() => {
    if (session) {
      requestPermissionOnce(() => {
        void syncPushSubscription();
      });
    }
  }, [session]);

  // Keep the device registered whenever permission changes (grant/revoke).
  useEffect(() => {
    if (!("Notification" in window) || !("permissions" in navigator)) return;
    let status: PermissionStatus | undefined;
    void navigator.permissions
      .query({ name: "notifications" })
      .then((result) => {
        status = result;
        result.addEventListener("change", onChange);
      })
      .catch(() => {});
    function onChange() {
      if (Notification.permission === "granted") {
        void syncPushSubscription();
      }
    }
    return () => status?.removeEventListener("change", onChange);
  }, []);

  const listQuery = useQuery({
    queryKey: ["notifications", "list", orgId],
    queryFn: () => listNotifications(),
    enabled: !!orgId,
    refetchInterval: POLL_MS,
  });

  const unreadQuery = useQuery({
    queryKey: ["notifications", "unread", orgId],
    queryFn: () => unreadNotificationCount(),
    enabled: !!orgId,
    refetchInterval: POLL_MS,
  });

  // Presence heartbeat so the backend can distinguish active vs idle agents.
  useEffect(() => {
    if (!orgId) return;
    const ping = () => {
      sendPresence().catch(() => {});
    };
    ping();
    const timer = window.setInterval(ping, PRESENCE_HEARTBEAT_MS);
    return () => window.clearInterval(timer);
  }, [orgId]);

  // Desktop pushes for notifications that arrive while the page is open.
  useEffect(() => {
    const items = listQuery.data?.items ?? [];
    if (seenRef.current === null) {
      // Seed with everything present on first load so stale items never pop.
      seenRef.current = new Set(items.map((n) => n.id));
      return;
    }
    for (const n of items) {
      if (seenRef.current.has(n.id)) continue;
      seenRef.current.add(n.id);
      if (n.conversation_id === activeConvRef.current) continue;
      if (document.visibilityState !== "visible") continue; // Web Push handles backgrounded tabs
      if (!("Notification" in window) || Notification.permission !== "granted") continue;
      const title = `New message from ${n.contact_name || "customer"}`;
      const notification = new Notification(title, { body: n.body, tag: n.id });
      notification.onclick = () => {
        window.focus();
        void navigate({ to: "/inbox", search: { conv: n.conversation_id } });
      };
    }
  }, [listQuery.data, navigate]);

  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["notifications"] });
  }, []);

  const markAllRead = useCallback(async () => {
    await markNotificationsRead();
    void queryClient.invalidateQueries({ queryKey: ["notifications"] });
  }, []);

  const value = useMemo<NotificationsContextValue>(
    () => ({
      items: listQuery.data?.items ?? [],
      unread: unreadQuery.data?.count ?? 0,
      open,
      setOpen,
      refresh,
      markAllRead,
    }),
    [listQuery.data, unreadQuery.data, open, refresh, markAllRead],
  );

  return (
    <NotificationsContext.Provider value={value}>
      {children}
    </NotificationsContext.Provider>
  );
}

export function useNotifications(): NotificationsContextValue {
  const ctx = useContext(NotificationsContext);
  if (!ctx) throw new Error("useNotifications must be used within a NotificationsProvider");
  return ctx;
}
