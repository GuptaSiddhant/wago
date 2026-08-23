import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
} from "react";
import type { ReactNode } from "react";
import { useNavigate } from "@tanstack/react-router";
import { login as apiLogin, me as apiMe } from "../api/client";
import { queryClient } from "../api/queryClient";
import { syncPocketBaseAuth } from "../api/pb";
import type { OrgSummary, Session } from "../api/types";
import {
  ApiError,
  clearStoredOrgId,
  getStoredOrgId,
  getStoredSession,
  setStoredOrgId,
  setStoredSession,
} from "./authStore";

// The auth/session value exposed to consumers through useSession. It bundles
// the current user, their session token, and the active org selection.
interface SessionContextValue {
  session: Session | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  org: OrgSummary | null;
  /** Signs in and returns the destination route ("inbox" or "select-org"). */
  login: (email: string, password: string) => Promise<PostLoginRoute>;
  logout: () => void;
  selectOrg: (orgId: string) => void;
  refresh: () => Promise<Session | null>;
}

/** Where the app should send a freshly authenticated user. */
export type PostLoginRoute = "/inbox" | "/select-org";

/**
 * Resolves the mandatory org-selection policy:
 * - 0 orgs   -> onboarding picker (superadmin can create, users see invite hint)
 * - 1 org    -> auto-selected
 * - >1 orgs  -> remembered selection wins; otherwise show the picker once
 */
export function resolvePostLoginRoute(session: Session): PostLoginRoute {
  const orgs = session.orgs ?? [];
  if (orgs.length === 0) return "/select-org";
  const stored = getStoredOrgId();
  if (stored && orgs.some((o) => o.id === stored)) return "/inbox";
  if (orgs.length === 1) {
    setStoredOrgId(orgs[0].id);
    return "/inbox";
  }
  clearStoredOrgId();
  return "/select-org";
}

const SessionContext = createContext<SessionContextValue | null>(null);

export function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(() =>
    getStoredSession(),
  );
  const [isLoading, setIsLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    let cancelled = false;

    async function bootstrap() {
      const stored = getStoredSession();
      if (!stored) {
        setIsLoading(false);
        return;
      }

      // Revalidate the token and refresh org memberships.
      try {
        const fresh = await apiMe();
        if (!cancelled && fresh) {
          // /auth/me does not echo the auth token, so keep the stored one.
          const merged = fresh.token
            ? fresh
            : { ...fresh, token: stored.token };
          setSession(merged);
          setStoredSession(merged);
        }
      } catch (err) {
        // Only clear the session on a definitive auth failure (401). Transient
        // errors (e.g. an aborted fetch during a page navigation) must not log
        // the user out.
        if (!cancelled && err instanceof ApiError && err.status === 401) {
          setSession(null);
          setStoredSession(null);
          clearStoredOrgId();
        }
      } finally {
        if (!cancelled) setIsLoading(false);
      }
    }

    void bootstrap();
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    syncPocketBaseAuth();
  }, [session]);

  const org = useMemo(() => {
    if (!session?.orgs) return null;
    // Strict: an org is active only when explicitly selected and still a
    // membership of this user. The _app guard and the effect below route
    // org-less users to /select-org; no silent fallback here.
    const orgId = getStoredOrgId();
    return session.orgs.find((o) => o.id === orgId) ?? null;
  }, [session]);

  // An authenticated user without a valid active org (never selected, removed
  // from their org mid-session, or the org was deleted) must land on the
  // picker before touching any org-scoped page.
  useEffect(() => {
    if (!isLoading && session && !org) {
      void navigate({ to: "/select-org" });
    }
  }, [isLoading, session, org, navigate]);

  const login = useCallback(
    async (email: string, password: string): Promise<PostLoginRoute> => {
      const result = await apiLogin(email, password);
      setStoredSession(result);
      setSession(result);
      return resolvePostLoginRoute(result);
    },
    [],
  );

  const logout = useCallback(() => {
    setSession(null);
    setStoredSession(null);
    clearStoredOrgId();
    queryClient.clear();
    void navigate({ to: "/login" });
  }, [navigate]);

  const selectOrg = useCallback((orgId: string) => {
    setStoredOrgId(orgId);
    // Force session refresh so the memoized org picks up the change.
    setSession((prev) => (prev ? { ...prev } : prev));
  }, []);

  const refresh = useCallback(async (): Promise<Session | null> => {
    const stored = getStoredSession();
    if (!stored) return null;
    const fresh = await apiMe();
    if (!fresh) return null;
    const merged = fresh.token ? fresh : { ...fresh, token: stored.token };
    setSession(merged);
    setStoredSession(merged);
    return merged;
  }, []);

  const value = useMemo(
    () => ({
      session,
      isAuthenticated: session != null,
      isLoading,
      org,
      login,
      logout,
      selectOrg,
      refresh,
    }),
    [session, isLoading, org, login, logout, selectOrg, refresh],
  );

  return (
    <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
  );
}

export function useSession(): SessionContextValue {
  const ctx = useContext(SessionContext);
  if (!ctx) throw new Error("useSession must be used within a SessionProvider");
  return ctx;
}
