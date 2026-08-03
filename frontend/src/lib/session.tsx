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

interface SessionContextValue {
  session: Session | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  org: OrgSummary | null;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  selectOrg: (orgId: string) => void;
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
    const orgId = getStoredOrgId();
    return session.orgs.find((o) => o.id === orgId) ?? session.orgs[0] ?? null;
  }, [session]);

  const login = useCallback(async (email: string, password: string) => {
    const result = await apiLogin(email, password);
    setStoredSession(result);
    setSession(result);
    const orgs = result.orgs ?? [];
    if (orgs.length > 0) {
      setStoredOrgId(orgs[0].id);
    }
  }, []);

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

  const value = useMemo(
    () => ({
      session,
      isAuthenticated: session != null,
      isLoading,
      org,
      login,
      logout,
      selectOrg,
    }),
    [session, isLoading, org, login, logout, selectOrg],
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
