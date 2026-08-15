import { useEffect, useState } from 'react'
import type { FormEvent, ReactNode } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Button as RACButton,
  Disclosure,
  DisclosureGroup,
  DisclosurePanel,
  Heading,
} from 'react-aria-components'
import type { DisclosureRenderProps } from 'react-aria-components'
import { ChevronDown, Save } from 'lucide-react'
import { getConfig, updateConfig } from '../../api/client'
import type { AppConfig } from '../../api/types'
import { Button } from '../../components/ui/Button'
import { FormError } from '../../components/ui/FormError'
import { Skeleton } from '../../components/ui/Skeleton'
import { TextField } from '../../components/ui/TextField'

const emptyConfig: AppConfig = {
  wa_webhook_verify_token: '',
  meta_app_secret: '',
  public_base_url: '',
  ai_enabled: false,
  ai_base_url: '',
  ai_api_key: '',
  ai_model: '',
  smtp_host: '',
  smtp_port: 587,
  smtp_username: '',
  smtp_password: '',
  smtp_tls: false,
  smtp_from_address: '',
  smtp_from_name: '',
  vapid_subject: '',
  wa_notification_template: '',
  messages_per_minute: 60,
  broadcast_batch_size: 10,
  broadcast_lease_seconds: 300,
  broadcast_max_attempts: 3,
}

function ConfigSection({
  id,
  title,
  description,
  children,
}: {
  id: string
  title: string
  description?: string
  children: ReactNode
}) {
  return (
    <Disclosure id={id} className="overflow-hidden rounded-xl border border-zinc-800 bg-zinc-900/50">
      {({ isExpanded }: DisclosureRenderProps) => (
        <>
          <RACButton
            slot="trigger"
            className="flex w-full items-center justify-between gap-3 px-4 py-3 text-left outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
          >
            <span className="flex min-w-0 flex-col gap-0.5">
              <Heading slot="heading" className="text-sm font-semibold text-zinc-200">
                {title}
              </Heading>
              {description ? (
                <span className="text-xs text-zinc-500">{description}</span>
              ) : null}
            </span>
            <ChevronDown
              size={16}
              className={`shrink-0 text-zinc-500 transition-transform ${isExpanded ? 'rotate-180' : ''}`}
            />
          </RACButton>
          <DisclosurePanel className="flex flex-col gap-4 border-t border-zinc-800 px-4 py-4">
            {children}
          </DisclosurePanel>
        </>
      )}
    </Disclosure>
  )
}

export function ConfigPage() {
  const queryClient = useQueryClient()
  const [form, setForm] = useState<AppConfig>(emptyConfig)
  const [saved, setSaved] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const { data, isLoading } = useQuery({
    queryKey: ['config'],
    queryFn: getConfig,
  })

  useEffect(() => {
    if (data) {
      setForm({
        ...data,
        // If no public base URL is set, default to the address the instance is
        // reachable at so webhook connections work out of the box.
        public_base_url: data.public_base_url || window.location.href,
      })
    }
  }, [data])

  const mutation = useMutation({
    mutationFn: updateConfig,
    onSuccess: (updated) => {
      setForm(updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
      void queryClient.invalidateQueries({ queryKey: ['config'] })
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to save configuration')
    },
  })

  function set<K extends keyof AppConfig>(key: K, value: AppConfig[K]) {
    setForm((prev) => ({ ...prev, [key]: value }))
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    setError(null)
    mutation.mutate(form)
  }

  if (isLoading) {
    return (
      <div className="flex h-full min-h-0 flex-1 flex-col">
        <div className="flex-1 space-y-4 overflow-y-auto px-6 py-6">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
          <Skeleton className="h-10 w-full" />
        </div>
      </div>
    )
  }

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <div className="flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex max-w-2xl flex-col gap-6">
          <div>
            <h1 className="text-xl font-semibold text-zinc-100">Instance configuration</h1>
            <p className="mt-1 text-sm text-zinc-500">
              These values are stored in the app database and take effect immediately. Admin
              credentials (email/password) are always read from the environment and can't be
              changed here.
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-5">
            <DisclosureGroup allowsMultipleExpanded defaultExpandedKeys={['whatsapp']}>
              <div className="flex flex-col gap-3">
                <ConfigSection
                  id="whatsapp"
                  title="WhatsApp webhook"
                  description="Verify token, app secret and callback URL"
                >
                  <TextField
                    label="Webhook verify token"
                    value={form.wa_webhook_verify_token}
                    onChange={(v) => set('wa_webhook_verify_token', v)}
                    description="The token Meta sends in the GET /api/wa/webhook handshake."
                  />
                  <TextField
                    label="Meta App Secret"
                    value={form.meta_app_secret}
                    onChange={(v) => set('meta_app_secret', v)}
                    type="password"
                    autoComplete="new-password"
                    description="Used to validate the X-Hub-Signature-256 of inbound messages."
                  />
                  <TextField
                    label="Public base URL"
                    value={form.public_base_url}
                    onChange={(v) => set('public_base_url', v)}
                    placeholder={window.location.href}
                    description="Externally reachable URL used to build the webhook callback URL."
                  />
                  <TextField
                    label="WhatsApp notification template"
                    value={form.wa_notification_template}
                    onChange={(v) => set('wa_notification_template', v)}
                    description="Approved Meta template used for best-effort WhatsApp notifications."
                  />
                </ConfigSection>

                <ConfigSection
                  id="ai"
                  title="AI assistant"
                  description="OpenAI-compatible chat provider"
                >
                  <label className="flex items-center gap-2 text-sm text-zinc-300">
                    <input
                      type="checkbox"
                      checked={form.ai_enabled}
                      onChange={(e) => set('ai_enabled', e.target.checked)}
                      className="size-4 accent-emerald-500"
                    />
                    Enable AI assistant
                  </label>
                  <TextField
                    label="Base URL"
                    value={form.ai_base_url}
                    onChange={(v) => set('ai_base_url', v)}
                    placeholder="https://api.openai.com/v1"
                  />
                  <TextField
                    label="API key"
                    value={form.ai_api_key}
                    onChange={(v) => set('ai_api_key', v)}
                    type="password"
                    autoComplete="new-password"
                  />
                  <TextField
                    label="Model"
                    value={form.ai_model}
                    onChange={(v) => set('ai_model', v)}
                    placeholder="gpt-4o-mini"
                  />
                </ConfigSection>

                <ConfigSection
                  id="smtp"
                  title="SMTP (notification email)"
                  description="Relay used to email inactive agents"
                >
                  <TextField
                    label="SMTP host"
                    value={form.smtp_host}
                    onChange={(v) => set('smtp_host', v)}
                    description="Leave empty to disable email notifications."
                  />
                  <div className="grid grid-cols-2 gap-4">
                    <TextField
                      label="Port"
                      value={String(form.smtp_port)}
                      onChange={(v) => set('smtp_port', Number(v) || 0)}
                      type="number"
                    />
                    <div className="flex items-end pb-1">
                      <label className="flex items-center gap-2 text-sm text-zinc-300">
                        <input
                          type="checkbox"
                          checked={form.smtp_tls}
                          onChange={(e) => set('smtp_tls', e.target.checked)}
                          className="size-4 accent-emerald-500"
                        />
                        Implicit TLS
                      </label>
                    </div>
                  </div>
                  <TextField
                    label="Username"
                    value={form.smtp_username}
                    onChange={(v) => set('smtp_username', v)}
                  />
                  <TextField
                    label="Password"
                    value={form.smtp_password}
                    onChange={(v) => set('smtp_password', v)}
                    type="password"
                    autoComplete="new-password"
                  />
                  <div className="grid grid-cols-2 gap-4">
                    <TextField
                      label="From address"
                      value={form.smtp_from_address}
                      onChange={(v) => set('smtp_from_address', v)}
                    />
                    <TextField
                      label="From name"
                      value={form.smtp_from_name}
                      onChange={(v) => set('smtp_from_name', v)}
                    />
                  </div>
                </ConfigSection>

                <ConfigSection
                  id="push"
                  title="Web push"
                  description="VAPID contact for browser push notifications"
                >
                  <TextField
                    label="VAPID subject"
                    value={form.vapid_subject}
                    onChange={(v) => set('vapid_subject', v)}
                    description="Contact (email or URL) sent in VAPID tokens."
                  />
                </ConfigSection>

                <ConfigSection
                  id="broadcast"
                  title="Broadcast worker tuning"
                  description="Rate limits and retry behavior for broadcasts"
                >
                  <div className="grid grid-cols-2 gap-4">
                    <TextField
                      label="Messages per minute"
                      value={String(form.messages_per_minute)}
                      onChange={(v) => set('messages_per_minute', Number(v) || 0)}
                      type="number"
                    />
                    <TextField
                      label="Batch size"
                      value={String(form.broadcast_batch_size)}
                      onChange={(v) => set('broadcast_batch_size', Number(v) || 0)}
                      type="number"
                    />
                    <TextField
                      label="Lease seconds"
                      value={String(form.broadcast_lease_seconds)}
                      onChange={(v) => set('broadcast_lease_seconds', Number(v) || 0)}
                      type="number"
                    />
                    <TextField
                      label="Max attempts"
                      value={String(form.broadcast_max_attempts)}
                      onChange={(v) => set('broadcast_max_attempts', Number(v) || 0)}
                      type="number"
                    />
                  </div>
                </ConfigSection>
              </div>
            </DisclosureGroup>

            <FormError message={error} />
            <div className="flex items-center gap-3">
              <Button type="submit" isDisabled={mutation.isPending}>
                <Save size={16} />
                {mutation.isPending ? 'Saving…' : 'Save configuration'}
              </Button>
              {saved ? (
                <span className="text-sm text-emerald-400">Saved — applied immediately</span>
              ) : null}
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
