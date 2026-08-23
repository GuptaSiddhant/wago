import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { Link, useNavigate, useSearch } from "@tanstack/react-router";
import { acceptInvite, inviteInfo } from "../../api/client";
import type { InviteInfo } from "../../api/types";
import { ApiError } from "../../lib/authStore";
import { useSession } from "../../lib/session";
import { Button } from "../../components/ui/Button";
import { Spinner } from "../../components/ui/Spinner";
import { TextField } from "../../components/ui/TextField";

export function JoinPage() {
  const { token } = useSearch({ strict: false }) as { token?: string };
  const { login } = useSession();
  const navigate = useNavigate();
  const [info, setInfo] = useState<InviteInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [accepted, setAccepted] = useState(false);

  useEffect(() => {
    if (!token) {
      setLoadError("This invite link is missing its token.");
      setLoading(false);
      return;
    }
    let cancelled = false;
    inviteInfo(token)
      .then((res) => {
        if (cancelled) return;
        if (res.status !== "pending") {
          setLoadError(
            res.expired
              ? "This invite has expired. Ask an admin for a new one."
              : "This invite has already been used."
          );
          setLoading(false);
          return;
        }
        setInfo(res);
        setLoading(false);
      })
      .catch((err) => {
        if (cancelled) return;
        setLoadError(err instanceof ApiError ? err.message : "This invite link is invalid.");
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!token) return;
    if (password.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    if (password !== confirm) {
      setError("Passwords do not match.");
      return;
    }
    setError(null);
    setSubmitting(true);
    try {
      await acceptInvite({ token, name: name.trim(), password });
      setAccepted(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to accept invite");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSignIn() {
    if (!info) return;
    try {
      const dest = await login(info.email, password);
      await navigate({ to: dest });
    } catch {
      await navigate({ to: "/login" });
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-zinc-950 px-4">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center justify-center gap-2.5">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-emerald-600 text-lg font-bold text-white">
            W
          </div>
          <div>
            <h1 className="text-xl font-semibold tracking-tight text-zinc-100">WaGo</h1>
            <p className="text-xs text-zinc-500">WhatsApp Business on the Go</p>
          </div>
        </div>

        {loading ? (
          <div className="flex justify-center py-16">
            <Spinner className="h-6 w-6" />
          </div>
        ) : loadError ? (
          <div className="rounded-2xl border border-zinc-800 bg-zinc-900/60 p-6 text-center shadow-xl shadow-black/30">
            <p className="text-sm text-zinc-400">{loadError}</p>
            <Link to="/login" className="mt-4 inline-block text-sm font-medium text-emerald-400 hover:underline">
              Back to sign in
            </Link>
          </div>
        ) : accepted ? (
          <div className="rounded-2xl border border-zinc-800 bg-zinc-900/60 p-6 text-center shadow-xl shadow-black/30">
            <p className="text-sm text-zinc-300">
              You're in! <span className="font-medium text-zinc-100">{info?.email}</span> is now a{" "}
              <span className="font-medium text-emerald-400">{info?.role}</span> at{" "}
              <span className="font-medium text-zinc-100">{info?.org_name}</span>.
            </p>
            <Button className="mt-4 w-full" onPress={handleSignIn}>
              Sign in
            </Button>
          </div>
        ) : (
          <form
            onSubmit={handleSubmit}
            className="space-y-4 rounded-2xl border border-zinc-800 bg-zinc-900/60 p-6 shadow-xl shadow-black/30"
          >
            <div>
              <h2 className="text-sm font-medium text-zinc-100">Accept your invite</h2>
              <p className="mt-1 text-xs text-zinc-500">
                You're joining <span className="font-medium text-zinc-300">{info?.org_name}</span> as a{" "}
                <span className="font-medium text-emerald-400">{info?.role}</span>
                {info?.team_name ? <> on <span className="font-medium text-zinc-300">{info.team_name}</span></> : null}
                . Set a password to create your account.
              </p>
            </div>

            <TextField label="Full name" value={name} onChange={setName} placeholder="Jane Doe" isRequired />

            <TextField
              label="Password"
              type="password"
              autoComplete="new-password"
              placeholder="At least 8 characters"
              value={password}
              onChange={setPassword}
              isRequired
            />
            <TextField
              label="Confirm password"
              type="password"
              autoComplete="new-password"
              placeholder="Repeat your password"
              value={confirm}
              onChange={setConfirm}
              isRequired
            />

            {error ? (
              <p
                role="alert"
                className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400"
              >
                {error}
              </p>
            ) : null}

            <Button type="submit" isDisabled={submitting} className="w-full">
              {submitting ? "Creating account…" : "Create account"}
            </Button>
          </form>
        )}
      </div>
    </div>
  );
}
