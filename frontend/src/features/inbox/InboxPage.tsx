import { useEffect, useMemo, useRef, useState } from 'react'
import type { FormEvent, KeyboardEvent } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Send, Inbox as InboxIcon, CheckCheck, Check, Plus, X, Clock3 } from 'lucide-react'
import { Route } from '../../routes/_app/inbox'
import {
  assignConversation,
  assignRoundRobin,
  getConversation,
  listConversations,
  listMessages,
  markConversationRead,
  sendMessage,
  updateContact,
} from '../../api/client'
import type {
  ContactDTO,
  ConversationDTO,
  ConversationDetailDTO,
  MessageDTO,
  SendMessagePayload,
} from '../../api/types'
import { ApiError } from '../../lib/authStore'
import { formatTime, statusTone, timeAgo } from '../../lib/format'
import { useSession } from '../../lib/session'
import { Avatar } from '../../components/ui/Avatar'
import { Badge } from '../../components/ui/Badge'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { SearchField } from '../../components/ui/SearchField'
import { SelectField } from '../../components/ui/Select'
import { Spinner } from '../../components/ui/Spinner'
import { TextField } from '../../components/ui/TextField'

const statusOptions = [
  { id: '', label: 'All conversations' },
  { id: 'open', label: 'Open' },
  { id: 'closed', label: 'Closed' },
]

export function InboxPage() {
  const { conv } = Route.useSearch()
  const navigate = Route.useNavigate()
  const queryClient = useQueryClient()
  const { session, org } = useSession()
  const meId = session?.user.id
  const orgId = org?.id ?? ''

  const [search, setSearch] = useState('')
  const [status, setStatus] = useState('')
  const [unassigned, setUnassigned] = useState(false)

  const conversationsQuery = useQuery({
    queryKey: ['conversations', orgId, { search, status, unassigned }],
    queryFn: () => listConversations({ search, status: status || undefined, unassigned }),
    enabled: orgId !== '',
    refetchInterval: 10_000,
  })

  const messagesQuery = useQuery({
    queryKey: ['messages', orgId, conv],
    queryFn: () => listMessages(conv!),
    enabled: conv != null && orgId !== '',
    refetchInterval: 8_000,
  })

  const detailQuery = useQuery({
    queryKey: ['conversation', orgId, conv],
    queryFn: () => getConversation(conv!),
    enabled: conv != null && orgId !== '',
    refetchInterval: 30_000,
  })

  const readMutation = useMutation({
    mutationFn: markConversationRead,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['conversations'] })
    },
  })

  useEffect(() => {
    if (conv) readMutation.mutate(conv)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [conv])

  function selectConversation(id: string) {
    void navigate({ search: (prev) => ({ ...prev, conv: id }) })
  }

  const conversations = conversationsQuery.data?.items ?? []
  const selected = conversations.find((c) => c.id === conv) ?? null

  return (
    <div className="flex h-full min-h-0 flex-1">
      <aside className="flex w-80 shrink-0 flex-col border-r border-zinc-800/80">
        <div className="space-y-2 p-3">
          <SearchField
            label="Search conversations"
            placeholder="Search name or phone…"
            value={search}
            onChange={setSearch}
            className="w-full"
          />
          <div className="flex items-center gap-2">
            <div className="min-w-0 flex-1">
              <SelectField
                aria-label="Filter by status"
                options={statusOptions}
                selectedKey={status}
                onSelectionChange={(k) => setStatus(typeof k === 'string' ? k : '')}
                className="min-w-0"
              />
            </div>
            <button
              type="button"
              onClick={() => setUnassigned((v) => !v)}
              aria-pressed={unassigned}
              className={`h-9 shrink-0 rounded-lg border px-2.5 text-xs font-medium transition ${
                unassigned
                  ? 'border-emerald-500/50 bg-emerald-500/10 text-emerald-400'
                  : 'border-zinc-700 text-zinc-400 hover:bg-zinc-900'
              }`}
            >
              Unassigned
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto pb-4">
          {conversationsQuery.isLoading ? (
            <div className="flex justify-center py-10">
              <Spinner />
            </div>
          ) : conversations.length === 0 ? (
            <EmptyState
              icon={<InboxIcon size={32} />}
              title="No conversations"
              description="Incoming WhatsApp messages will appear here."
            />
          ) : (
            <ul className="space-y-0.5 px-2">
              {conversations.map((c) => (
                <ConversationRow
                  key={c.id}
                  conversation={c}
                  active={c.id === conv}
                  onSelect={() => selectConversation(c.id)}
                />
              ))}
            </ul>
          )}
        </div>
      </aside>

      <section className="flex min-w-0 flex-1 flex-col">
        {selected ? (
          <Thread
            key={selected.id}
            conversation={selected}
            detail={detailQuery.data ?? null}
            messages={messagesQuery.data?.items ?? null}
            isLoadingMessages={messagesQuery.isLoading}
            meId={meId}
          />
        ) : (
          <div className="flex flex-1 items-center justify-center">
            <EmptyState
              icon={<InboxIcon size={36} />}
              title="Select a conversation"
              description="Choose a conversation on the left to view and reply."
            />
          </div>
        )}
      </section>
    </div>
  )
}

function ConversationRow({
  conversation,
  active,
  onSelect,
}: {
  conversation: ConversationDTO
  active: boolean
  onSelect: () => void
}) {
  const { contact, unread_count, last_message_at, last_message, whatsapp_account } = conversation
  return (
    <li>
      <button
        type="button"
        onClick={onSelect}
        className={`flex w-full items-start gap-3 rounded-xl px-3 py-2.5 text-left transition ${
          active ? 'bg-zinc-900' : 'hover:bg-zinc-900/50'
        }`}
      >
        <Avatar name={contact.name || contact.phone} size={40} />
        <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between gap-2">
            <span className="truncate text-sm font-medium text-zinc-100">
              {contact.name || contact.phone}
            </span>
            <span className="shrink-0 text-[11px] text-zinc-500">
              {timeAgo(last_message_at)}
            </span>
          </div>
          <div className="mt-0.5 flex items-center justify-between gap-2">
            <span className="truncate text-[13px] text-zinc-500">
              {last_message?.direction === 'outbound' ? (
                <CheckCheck size={12} className="mr-0.5 inline text-zinc-500" />
              ) : null}
              {last_message?.body ?? 'No messages yet'}
            </span>
            {unread_count > 0 ? (
              <span className="flex h-5 min-w-5 shrink-0 items-center justify-center rounded-full bg-emerald-500 px-1.5 text-[11px] font-semibold text-emerald-950">
                {unread_count > 99 ? '99+' : unread_count}
              </span>
            ) : null}
          </div>
          <div className="mt-0.5 text-[11px] text-zinc-600">{whatsapp_account.display_name}</div>
        </div>
      </button>
    </li>
  )
}

function Thread({
  conversation,
  detail,
  messages,
  isLoadingMessages,
  meId,
}: {
  conversation: ConversationDTO
  detail: ConversationDetailDTO | null
  messages: MessageDTO[] | null
  isLoadingMessages: boolean
  meId?: string
}) {
  const queryClient = useQueryClient()
  const { session, org } = useSession()
  const { contact, whatsapp_account, status, assignee_id } = conversation
  const bottomRef = useRef<HTMLDivElement>(null)

  const canEditDetails =
    session?.is_admin === true ||
    org?.role === 'owner' ||
    org?.role === 'admin' ||
    org?.role === 'agent'

  const displayMessages = useMemo(() => (messages ? [...messages].reverse() : []), [messages])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ block: 'end' })
  }, [displayMessages.length])

  const assignMutation = useMutation({
    mutationFn: () => assignConversation(conversation.id, meId!),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['conversations'] })
    },
  })

  const rrMutation = useMutation({
    mutationFn: () => assignRoundRobin(conversation.id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['conversations'] })
      void queryClient.invalidateQueries({ queryKey: ['conversation'] })
    },
  })

  const isAssignedToMe = meId != null && assignee_id === meId

  function invalidateContact() {
    void queryClient.invalidateQueries({ queryKey: ['conversation'] })
    void queryClient.invalidateQueries({ queryKey: ['contacts'] })
  }

  return (
    <div className="flex h-full min-h-0 flex-col">
      <header className="flex items-center gap-3 border-b border-zinc-800/80 px-5 py-3">
        <Avatar name={contact.name || contact.phone} size={36} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h2 className="truncate text-sm font-semibold text-zinc-100">
              {contact.name || contact.phone}
            </h2>
            <Badge tone={statusTone(status)}>{status}</Badge>
            {isAssignedToMe ? <Badge tone="blue">You</Badge> : null}
          </div>
          <p className="truncate text-xs text-zinc-500">
            {contact.phone} · {whatsapp_account.display_name}
          </p>
        </div>
        {meId ? (
          <div className="flex shrink-0 items-center gap-2">
            <Button
              variant="secondary"
              size="sm"
              onPress={() => rrMutation.mutate()}
              isDisabled={rrMutation.isPending}
            >
              {rrMutation.isPending ? 'Assigning…' : 'Round-robin'}
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onPress={() => assignMutation.mutate()}
              isDisabled={assignMutation.isPending || isAssignedToMe}
            >
              {assignMutation.isPending ? 'Assigning…' : isAssignedToMe ? 'Assigned' : 'Assign to me'}
            </Button>
          </div>
        ) : null}
      </header>

      <div className="flex min-h-0 flex-1">
        <div className="flex min-w-0 flex-1 flex-col">
          <div className="flex-1 overflow-y-auto px-5 py-4">
            {isLoadingMessages && messages == null ? (
              <div className="flex justify-center py-10">
                <Spinner />
              </div>
            ) : displayMessages.length === 0 ? (
              <EmptyState title="No messages" description="Messages in this conversation will appear here." />
            ) : (
              <ul className="space-y-2">
                {displayMessages.map((m) => (
                  <MessageBubble key={m.id} message={m} />
                ))}
              </ul>
            )}
            <div ref={bottomRef} />
          </div>

          <Composer
            conversationId={conversation.id}
            inWindow={detail?.in_window}
            onSent={() => {
              void queryClient.invalidateQueries({ queryKey: ['conversations'] })
              void queryClient.invalidateQueries({ queryKey: ['conversation'] })
            }}
          />
        </div>

        <ContactSidebar
          contact={detail?.contact ?? conversation.contact}
          detail={detail}
          canEdit={canEditDetails}
          onChanged={invalidateContact}
        />
      </div>
    </div>
  )
}

function MessageBubble({ message }: { message: MessageDTO }) {
  const outbound = message.direction === 'outbound'
  const failed = message.status === 'failed'
  return (
    <li className={`flex ${outbound ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`max-w-[70%] rounded-2xl px-3.5 py-2 text-sm ${
          outbound
            ? failed
              ? 'rounded-br-md bg-red-950 text-red-100 ring-1 ring-inset ring-red-500/40'
              : 'rounded-br-md bg-emerald-600 text-emerald-50'
            : 'rounded-bl-md bg-zinc-800 text-zinc-100'
        }`}
      >
        <p className="whitespace-pre-wrap break-words">{message.body}</p>
        <div
          className={`mt-1 flex items-center justify-end gap-1 text-[10px] ${
            outbound ? (failed ? 'text-red-300/80' : 'text-emerald-200/80') : 'text-zinc-500'
          }`}
        >
          <span>{formatTime(message.created)}</span>
          <MessageStatus message={message} />
        </div>
      </div>
    </li>
  )
}

function MessageStatus({ message }: { message: MessageDTO }) {
  if (message.direction !== 'outbound') return null
  switch (message.status) {
    case 'read':
      return (
        <CheckCheck size={11} className="text-sky-300" aria-label="Read" title="Read" />
      )
    case 'delivered':
      return (
        <CheckCheck size={11} aria-label="Delivered" title="Delivered" />
      )
    case 'failed':
      return (
        <span className="font-semibold uppercase tracking-wide" aria-label="Failed">
          Failed
        </span>
      )
    default:
      return <Check size={11} aria-label="Sent" title="Sent" />
  }
}

function Composer({
  conversationId,
  inWindow,
  onSent,
}: {
  conversationId: string
  inWindow?: boolean
  onSent: () => void
}) {
  const queryClient = useQueryClient()
  const canFreeForm = inWindow !== false

  const [draft, setDraft] = useState('')
  const [tmplName, setTmplName] = useState('')
  const [tmplLang, setTmplLang] = useState('en_US')
  const [params, setParams] = useState<string[]>([''])
  const [error, setError] = useState<string | null>(null)

  const sendMutation = useMutation({
    mutationFn: (payload: SendMessagePayload) => sendMessage(payload),
    onSuccess: () => {
      setDraft('')
      setTmplName('')
      setTmplLang('en_US')
      setParams([''])
      setError(null)
      void queryClient.invalidateQueries({ queryKey: ['messages', conversationId] })
      onSent()
    },
  })

  function buildPayload(): SendMessagePayload | null {
    if (canFreeForm) {
      const body = draft.trim()
      if (!body) return null
      return { conversation_id: conversationId, body }
    }
    const name = tmplName.trim()
    if (!name) return null
    const parameters = params
      .map((p) => p.trim())
      .filter(Boolean)
      .map((text) => ({ type: 'text', text }))
    return {
      conversation_id: conversationId,
      template: { name, language: tmplLang.trim() || 'en_US', parameters },
    }
  }

  function submit() {
    if (sendMutation.isPending) return
    const payload = buildPayload()
    if (!payload) return
    setError(null)
    sendMutation.mutate(payload, {
      onError: (err) => {
        setError(err instanceof ApiError ? err.message : 'Failed to send message')
      },
    })
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    submit()
  }

  function handleKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      submit()
    }
  }

  const canSend = canFreeForm ? draft.trim() !== '' : tmplName.trim() !== ''

  return (
    <form onSubmit={handleSubmit} className="border-t border-zinc-800/80 p-3">
      {error ? (
        <p
          role="alert"
          className="mb-2 rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-400"
        >
          {error}
        </p>
      ) : null}

      {canFreeForm ? (
        <>
          <div className="mb-2 flex items-center gap-1.5 text-[11px] text-emerald-400/80">
            <Clock3 size={12} />
            <span>24h customer service window open — free-form replies allowed.</span>
          </div>
          <div className="flex items-end gap-2">
            <textarea
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={handleKeyDown}
              rows={2}
              placeholder="Type a message… (Enter to send)"
              aria-label="Message"
              className="min-h-10 flex-1 resize-none rounded-xl border border-zinc-700 bg-zinc-900 px-3.5 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-500 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/30"
            />
            <Button
              type="submit"
              isDisabled={sendMutation.isPending || !canSend}
              aria-label="Send message"
            >
              <Send size={16} />
              <span className="hidden sm:inline">Send</span>
            </Button>
          </div>
        </>
      ) : (
        <div className="space-y-3">
          <p
            role="status"
            className="flex items-start gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-xs text-amber-400"
          >
            <Clock3 size={14} className="mt-0.5 shrink-0" />
            <span>
              The 24h customer service window has closed. Outbound messages require an approved Meta
              template.
            </span>
          </p>
          <div className="grid gap-3 sm:grid-cols-2">
            <TextField
              label="Template name"
              value={tmplName}
              onChange={setTmplName}
              placeholder="order_confirmation"
            />
            <TextField
              label="Language code"
              value={tmplLang}
              onChange={setTmplLang}
              placeholder="en_US"
            />
          </div>
          <TemplateParameters params={params} onChange={setParams} />
          <div className="flex justify-end">
            <Button
              type="submit"
              isDisabled={sendMutation.isPending || !canSend}
              aria-label="Send template message"
            >
              <Send size={16} />
              <span>{sendMutation.isPending ? 'Sending…' : 'Send template'}</span>
            </Button>
          </div>
        </div>
      )}
    </form>
  )
}

function TemplateParameters({
  params,
  onChange,
}: {
  params: string[]
  onChange: (next: string[]) => void
}) {
  function update(index: number, value: string) {
    onChange(params.map((p, i) => (i === index ? value : p)))
  }
  function remove(index: number) {
    onChange(params.filter((_, i) => i !== index))
  }
  function add() {
    onChange([...params, ''])
  }
  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-zinc-300">
          Parameters{' '}
          <span className="text-xs text-zinc-500">{'({{1}}, {{2}}, …)'}</span>
        </span>
        <Button type="button" size="sm" variant="ghost" onPress={add}>
          <Plus size={14} />
          Add
        </Button>
      </div>
      {params.map((p, i) => (
        <div key={i} className="flex items-center gap-2">
          <input
            value={p}
            onChange={(e) => update(i, e.target.value)}
            placeholder={`Value for {{${i + 1}}}`}
            aria-label={`Template parameter ${i + 1}`}
            className="h-10 w-full rounded-xl border border-zinc-700 bg-zinc-900 px-3 text-sm text-zinc-100 placeholder:text-zinc-500 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/30"
          />
          {params.length > 1 ? (
            <Button
              type="button"
              size="icon"
              variant="ghost"
              aria-label={`Remove parameter ${i + 1}`}
              onPress={() => remove(i)}
            >
              <X size={14} className="text-zinc-500 hover:text-red-400" />
            </Button>
          ) : null}
        </div>
      ))}
    </div>
  )
}

function ContactSidebar({
  contact,
  detail,
  canEdit,
  onChanged,
}: {
  contact: ContactDTO
  detail: ConversationDetailDTO | null
  canEdit: boolean
  onChanged: () => void
}) {
  const tagsMutation = useMutation({
    mutationFn: (tags: string[]) => updateContact(contact.id, { tags }),
    onSuccess: onChanged,
  })

  function addTag(tag: string) {
    const next = [...(contact.tags ?? []), tag]
    tagsMutation.mutate(next)
  }
  function removeTag(tag: string) {
    const next = (contact.tags ?? []).filter((t) => t !== tag)
    tagsMutation.mutate(next)
  }

  return (
    <aside className="flex w-72 shrink-0 flex-col border-l border-zinc-800/80">
      <div className="border-b border-zinc-800/80 px-4 py-3">
        <p className="text-xs font-medium uppercase tracking-wider text-zinc-500">Contact</p>
        <h3 className="mt-1 truncate text-sm font-semibold text-zinc-100">
          {contact.name || contact.phone}
        </h3>
        <p className="truncate text-xs text-zinc-500">{contact.phone}</p>
      </div>

      <div className="flex-1 space-y-5 overflow-y-auto px-4 py-4">
        <section aria-labelledby="sidebar-status">
          <p id="sidebar-status" className="mb-2 text-xs font-medium uppercase tracking-wider text-zinc-500">
            Conversation
          </p>
          <dl className="space-y-1.5 text-[13px]">
            <div className="flex items-center justify-between gap-2">
              <dt className="text-zinc-500">Account</dt>
              <dd className="truncate text-zinc-300">
                {detail?.whatsapp_account.display_name ?? '—'}
              </dd>
            </div>
            <div className="flex items-center justify-between gap-2">
              <dt className="text-zinc-500">Assignee</dt>
              <dd className="truncate text-zinc-300">
                {detail?.assignee_name ?? (detail?.assignee_id ? 'Unknown' : 'Unassigned')}
              </dd>
            </div>
            {detail?.team_name ? (
              <div className="flex items-center justify-between gap-2">
                <dt className="text-zinc-500">Team</dt>
                <dd className="truncate text-zinc-300">{detail.team_name}</dd>
              </div>
            ) : null}
          </dl>
        </section>

        <section aria-labelledby="sidebar-tags">
          <p id="sidebar-tags" className="mb-2 text-xs font-medium uppercase tracking-wider text-zinc-500">
            Tags
          </p>
          <TagEditor
            tags={contact.tags ?? []}
            onAdd={addTag}
            onRemove={removeTag}
            disabled={!canEdit}
            isPending={tagsMutation.isPending}
          />
        </section>

        <NotesEditor
          contact={contact}
          canEdit={canEdit}
          onSaved={onChanged}
        />
      </div>
    </aside>
  )
}

function TagEditor({
  tags,
  onAdd,
  onRemove,
  disabled,
  isPending,
}: {
  tags: string[]
  onAdd: (tag: string) => void
  onRemove: (tag: string) => void
  disabled: boolean
  isPending: boolean
}) {
  const [value, setValue] = useState('')

  function submit(e: FormEvent) {
    e.preventDefault()
    const tag = value.trim()
    if (tag && !tags.includes(tag)) onAdd(tag)
    setValue('')
  }

  return (
    <div>
      {tags.length > 0 ? (
        <ul className="flex flex-wrap gap-1.5">
          {tags.map((t) => (
            <li
              key={t}
              className="inline-flex items-center gap-1 rounded-full bg-emerald-500/10 py-0.5 pl-2.5 pr-1 text-xs text-emerald-400 ring-1 ring-inset ring-emerald-500/25"
            >
              {t}
              {!disabled ? (
                <button
                  type="button"
                  onClick={() => onRemove(t)}
                  aria-label={`Remove tag ${t}`}
                  className="rounded-full p-0.5 text-emerald-400/70 transition hover:bg-emerald-500/20 hover:text-emerald-200"
                >
                  <X size={11} />
                </button>
              ) : null}
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-xs text-zinc-600">{disabled ? 'No tags.' : 'No tags yet.'}</p>
      )}
      {!disabled ? (
        <form onSubmit={submit} className="mt-2 flex gap-1.5">
          <input
            value={value}
            onChange={(e) => setValue(e.target.value)}
            placeholder="Add a tag…"
            aria-label="New tag"
            className="h-9 w-full rounded-lg border border-zinc-700 bg-zinc-900 px-2.5 text-xs text-zinc-100 placeholder:text-zinc-500 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/30"
          />
          <Button
            type="submit"
            size="icon"
            aria-label="Add tag"
            isDisabled={isPending || value.trim() === ''}
          >
            <Plus size={14} />
          </Button>
        </form>
      ) : null}
    </div>
  )
}

function NotesEditor({
  contact,
  canEdit,
  onSaved,
}: {
  contact: ContactDTO
  canEdit: boolean
  onSaved: () => void
}) {
  const [draft, setDraft] = useState(contact.notes ?? '')
  const [saved, setSaved] = useState(true)

  const saveMutation = useMutation({
    mutationFn: (notes: string) => updateContact(contact.id, { notes }),
    onSuccess: () => {
      setSaved(true)
      onSaved()
    },
  })

  useEffect(() => {
    setDraft(contact.notes ?? '')
    setSaved(true)
  }, [contact.id, contact.notes])

  function submit(e: FormEvent) {
    e.preventDefault()
    saveMutation.mutate(draft)
  }

  return (
    <form onSubmit={submit} aria-label="Notes">
      <div className="mb-2 flex items-center justify-between">
        <p className="text-xs font-medium uppercase tracking-wider text-zinc-500">Notes</p>
        {canEdit && !saved ? (
          <span className="text-[11px] text-zinc-500">Unsaved</span>
        ) : null}
      </div>
      <textarea
        value={draft}
        onChange={(e) => {
          setDraft(e.target.value)
          setSaved(false)
        }}
        readOnly={!canEdit}
        rows={4}
        placeholder={canEdit ? 'Add internal notes…' : 'No notes.'}
        aria-label="Notes"
        className="w-full resize-none rounded-xl border border-zinc-700 bg-zinc-900 px-3 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-500 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/30 read-only:opacity-70"
      />
      {canEdit ? (
        <div className="mt-2 flex justify-end">
          <Button
            type="submit"
            size="sm"
            isDisabled={saveMutation.isPending || saved}
          >
            {saveMutation.isPending ? 'Saving…' : 'Save notes'}
          </Button>
        </div>
      ) : null}
    </form>
  )
}
