import { useState } from "react";
import type { FormEvent } from "react";
import { useNavigate } from "@tanstack/react-router";
import { ArrowLeft, LogOut } from "lucide-react";
import { updateProfile } from "../../api/client";
import { ApiError } from "../../lib/authStore";
import { useSession } from "../../lib/session";
import { Button } from "../../components/ui/Button";
import { TextField } from "../../components/ui/TextField";

/**
 * The logged-in user's personal account page. Deliberately reachable without
 * an organization selected — it lives outside the org-gated _app layout.
 */
export function AccountPage() {
  const { session, refresh, logout } = useSession();
  const navigate = useNavigate();
  const [name, setName] = useState(session?.user.name ?? "");
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  if (!session) return null;

  async function handleSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setError(null);
    setSaved(false);
    setSubmitting(true);
    try {
      await updateProfile(name.trim());
      // Pull a fresh session so /auth/me reflects the new name everywhere.
      await refresh();
      setSaved(true);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to update profile");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-canvas px-4">
      <div className="w-full max-w-sm">
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-lg font-semibold tracking-tight text-ink">
            Your account
          </h1>
          <Button
            variant="ghost"
            size="sm"
            onPress={() => {
              if (window.history.length > 1) window.history.back();
              else void navigate({ to: "/login" });
            }}
          >
            <ArrowLeft size={14} />
            Back
          </Button>
        </div>

        <form
          onSubmit={handleSubmit}
          className="space-y-4 rounded-2xl border border-edge bg-panel p-6 shadow-xl shadow-black/30"
        >
          <TextField
            label="Email"
            value={session.user.email}
            onChange={() => {}}
            isReadOnly
            description="Email changes require an administrator."
          />
          <TextField
            label="Display name"
            value={name}
            onChange={setName}
            placeholder="Jane Doe"
            isRequired
          />

          {saved ? (
            <p className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-400">
              Profile updated.
            </p>
          ) : null}
          {error ? (
            <p
              role="alert"
              className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400"
            >
              {error}
            </p>
          ) : null}

          <Button type="submit" isDisabled={submitting} className="w-full">
            {submitting ? "Saving…" : "Save changes"}
          </Button>
        </form>

        <Button variant="ghost" className="mt-4 w-full" onPress={logout}>
          <LogOut size={14} />
          Log out
        </Button>
      </div>
    </div>
  );
}
