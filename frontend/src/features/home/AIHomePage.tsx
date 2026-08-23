import { useState } from 'react'
import type { FormEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useChat } from '@tanstack/ai-react'
import { Bot, MessageSquare, Send, Sparkles } from 'lucide-react'
import { listConversations } from '../../api/client'
import type { ConversationDTO } from '../../api/types'
import { aiChatConnection } from '../../api/ai'
import { useSession } from '../../lib/session'
import { timeAgo } from '../../lib/format'
import { Avatar } from '../../components/ui/Avatar'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { Skeleton } from '../../components/ui/Skeleton'
import { Spinner } from '../../components/ui/Spinner'
import { TextField } from '../../components/ui/TextField'

const suggestions = [
  'Summarize this conversation',
  'What still needs a reply?',
  'Draft a reply for me',
]

type ChatMessage = {
  id: string
  role: string
  parts: Array<{ type: string; content?: unknown }>
}

function renderText(message: ChatMessage): string {
  return message.parts
    .filter((p): p is { type: 'text'; content: string } => p.type === 'text' && typeof p.content === 'string')
    .map((p) => p.content)
    .join('')
}

export function AIHomePage() {
  const { session, org } = useSession()
  const aiEnabled = session?.ai_enabled === true
  const orgId = org?.id ?? ''
  const [activeConv, setActiveConv] = useState<ConversationDTO | null>(null)
  const [input, setInput] = useState('')

  const conversationsQuery = useQuery({
    queryKey: ['conversations', orgId, 'open'],
    queryFn: () => listConversations({ status: 'open' }),
    enabled: orgId !== '',
    refetchInterval: 20_000,
  })

  const { messages, sendMessage, isLoading, error } = useChat({
    connection: aiChatConnection,
    forwardedProps: { conversationId: activeConv?.id ?? '' },
  })

  function handleSend(text: string) {
    const content = text.trim()
    if (!content || isLoading) return
    void sendMessage(content)
    setInput('')
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (isLoading) return
    handleSend(input)
  }

  const conversations = conversationsQuery.data?.items ?? []

  return (
    <div className="flex h-full min-h-0 flex-1">
      {/* Active conversations */}
      <aside className="flex w-72 shrink-0 flex-col border-r border-zinc-800/80">
        <div className="border-b border-zinc-800/80 px-4 py-3">
          <h2 className="text-sm font-semibold text-zinc-100">Active conversations</h2>
          <p className="text-[11px] text-zinc-500">Open conversations in this org</p>
        </div>
        <div className="flex-1 overflow-y-auto py-2">
          {conversationsQuery.isLoading ? (
            <div className="space-y-2 px-3">
              {Array.from({ length: 6 }).map((_, i) => (
                <div key={i} className="flex items-center gap-3 rounded-xl px-3 py-2">
                  <Skeleton className="h-9 w-9 shrink-0 rounded-full" />
                  <div className="min-w-0 flex-1 space-y-2">
                    <Skeleton className="h-3.5 w-28" />
                    <Skeleton className="h-3 w-20" />
                  </div>
                </div>
              ))}
            </div>
          ) : conversations.length === 0 ? (
            <EmptyState
              icon={<MessageSquare size={28} />}
              title="No active conversations"
              description="Open conversations in your inbox will show up here."
            />
          ) : (
            <ul className="space-y-0.5 px-2">
              {conversations.map((c) => {
                const active = c.id === activeConv?.id
                return (
                  <li key={c.id}>
                    <button
                      type="button"
                      onClick={() => setActiveConv(c)}
                      className={`flex w-full items-start gap-3 rounded-xl px-3 py-2.5 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60 ${
                        active ? 'bg-zinc-900' : 'hover:bg-zinc-900/50'
                      }`}
                    >
                      <Avatar name={c.contact.name || c.contact.phone} size={36} />
                      <span className="min-w-0 flex-1">
                        <span className="flex items-baseline justify-between gap-2">
                          <span className="truncate text-sm font-medium text-zinc-100">
                            {c.contact.name || c.contact.phone}
                          </span>
                          <span className="shrink-0 text-[11px] text-zinc-500">
                            {timeAgo(c.last_message_at)}
                          </span>
                        </span>
                        <span className="block truncate text-xs text-zinc-500">
                          {c.last_message?.body || 'No messages yet'}
                        </span>
                      </span>
                    </button>
                  </li>
                )
              })}
            </ul>
          )}
        </div>
      </aside>

      {/* Chat */}
      <section className="flex min-w-0 flex-1 flex-col px-6 py-5">
        <div className="flex items-center gap-3 border-b border-zinc-800/80 pb-4">
          <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-emerald-600/15 text-emerald-400">
            <Bot size={18} />
          </div>
          <div className="min-w-0 flex-1">
            <div className="text-sm font-semibold text-zinc-100">
              {activeConv ? activeConv.contact.name || activeConv.contact.phone : 'WaGo assistant'}
            </div>
            <div className="text-[11px] text-zinc-500">
              {activeConv
                ? `Context: this conversation (${activeConv.contact.phone || 'no phone'})`
                : 'Answers about your support inbox'}
            </div>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto py-4">
          {!aiEnabled ? (
            <div className="flex h-full items-center justify-center">
              <EmptyState
                icon={<Bot size={36} />}
                title="AI assistant is not enabled"
                description="Set AI_ENABLED=true (with AI_BASE_URL, AI_API_KEY, AI_MODEL) in the server config to chat with your conversations."
              />
            </div>
          ) : messages.length === 0 ? (
            <div className="flex h-full items-center justify-center">
              <EmptyState
                icon={<Sparkles size={32} />}
                title={activeConv ? `Chat about ${activeConv.contact.name || 'this conversation'}` : 'Ask me anything'}
                description={
                  activeConv
                    ? 'I have the transcript of this conversation in context — ask for a summary, what needs a reply, or a draft.'
                    : 'Select a conversation on the left, then ask for a summary or a suggested reply.'
                }
              />
            </div>
          ) : (
            <div className="space-y-3">
              {messages.map((m) => (
                <div
                  key={m.id}
                  className={`flex ${m.role === 'user' ? 'justify-end' : 'justify-start'}`}
                >
                  <div
                    className={`max-w-[75%] whitespace-pre-wrap rounded-2xl px-4 py-3 text-sm ${
                      m.role === 'user'
                        ? 'bg-emerald-600 text-white'
                        : 'border border-zinc-800 bg-zinc-900/50 text-zinc-100'
                    }`}
                  >
                    {renderText(m)}
                  </div>
                </div>
              ))}
              {isLoading ? (
                <div className="flex justify-start">
                  <div className="flex items-center gap-2 rounded-2xl border border-zinc-800 bg-zinc-900/50 px-4 py-3 text-sm text-zinc-500">
                    <Spinner className="!h-3.5 !w-3.5" />
                    Thinking…
                  </div>
                </div>
              ) : null}
            </div>
          )}

          {aiEnabled && activeConv && (
            <div className="mt-4 flex flex-wrap gap-2">
              {suggestions.map((s) => (
                <Button
                  key={s}
                  size="sm"
                  variant="secondary"
                  onPress={() => handleSend(s)}
                  isDisabled={isLoading}
                >
                  <Sparkles size={13} />
                  {s}
                </Button>
              ))}
            </div>
          )}

          {error ? (
            <p role="alert" className="mt-3 text-xs text-red-400">
              {error.message}
            </p>
          ) : null}
        </div>

        <form onSubmit={handleSubmit} className="flex items-end gap-2 border-t border-zinc-800/80 pt-4">
          <div className="min-w-0 flex-1">
            <TextField
              label="Message the assistant"
              value={input}
              onChange={setInput}
              placeholder={
                activeConv
                  ? `Ask about ${activeConv.contact.name || 'this conversation'}…`
                  : 'Ask about your support inbox…'
              }
              isDisabled={!aiEnabled || isLoading}
              autoComplete="off"
            />
          </div>
          <Button type="submit" isDisabled={!aiEnabled || isLoading || input.trim() === ''}>
            {isLoading ? <Spinner className="!h-4 !w-4" /> : <Send size={16} />}
            {isLoading ? 'Thinking…' : 'Send'}
          </Button>
        </form>
      </section>
    </div>
  )
}