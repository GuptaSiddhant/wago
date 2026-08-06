import { ApiError, getStoredOrgId, getStoredSession } from '../lib/authStore'
import type {
  AnalyticsResponse,
  BroadcastCreateInput,
  BroadcastDetail,
  BroadcastDTO,
  CallDTO,
  CallEventDTO,
  ContactDTO,
  ContactInput,
  ConversationsResponse,
  ConversationDetailDTO,
  InviteDTO,
  InviteInfo,
  InviteInput,
  ListResponse,
  MediaMessageResult,
  MediaUploadResult,
  MessagesResponse,
  MessageTemplateDTO,
  NotificationsResponse,
  OrgSummary,
  PhoneMetaResult,
  SendMessagePayload,
  SendMessageResult,
  Session,
  TeamDTO,
  TeamMemberDTO,
  TemplateInput,
  TemplateSendInput,
  TemplateSendResult,
  TemplatesResponse,
  WaAccountDTO,
  WaAccountInput,
} from './types'

const API_BASE = '/api/wa'

export interface InboxFilters {
  search?: string
  assignee?: string
  unassigned?: boolean
  status?: string
  contact?: string
  limit?: number
  offset?: number
}

/**
 * Thin HTTPS wrapper around the Wago backend. Every call injects the stored
 * session token (Authorization) and active org (X-Org-Id) so the server can
 * scope records and authorize the request. Errors are normalized into ApiError.
 */
async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body != null && typeof init.body === 'string') {
    headers.set('Content-Type', 'application/json')
  }

  const token = getStoredSession()?.token
  const orgId = getStoredOrgId()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (orgId) headers.set('X-Org-Id', orgId)

  const res = await fetch(`${API_BASE}${path}`, { ...init, headers })

  let body: unknown = null
  try {
    body = await res.json()
  } catch {
    // A failed body read (e.g. aborted during a page navigation) must not be
    // mistaken for an empty/valid response.
    throw new ApiError(
      res.ok ? 0 : res.status,
      res.ok ? 'Invalid response from server' : `Request failed (${res.status})`,
    )
  }

  if (!res.ok) {
    throw new ApiError(
      res.status,
      (body as { message?: string } | null)?.message ?? `Request failed (${res.status})`,
    )
  }
  return body as T
}

export async function login(email: string, password: string): Promise<Session> {
  return apiFetch<Session>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  })
}

export async function me(): Promise<Session> {
  return apiFetch<Session>('/auth/me')
}

export async function createOrg(name: string): Promise<OrgSummary> {
  return apiFetch<OrgSummary>('/orgs', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export async function listConversations(filters: InboxFilters): Promise<ConversationsResponse> {
  const params = new URLSearchParams()
  if (filters.search) params.set('search', filters.search)
  if (filters.assignee) params.set('assignee', filters.assignee)
  if (filters.unassigned) params.set('unassigned', 'true')
  if (filters.status) params.set('status', filters.status)
  if (filters.contact) params.set('contact', filters.contact)
  if (filters.limit) params.set('limit', String(filters.limit))
  if (filters.offset) params.set('offset', String(filters.offset))
  const qs = params.toString()
  return apiFetch<ConversationsResponse>(`/inbox${qs ? `?${qs}` : ''}`)
}

export async function listMessages(conversationId: string): Promise<MessagesResponse> {
  return apiFetch<MessagesResponse>(`/conversations/${conversationId}/messages`)
}

export async function getConversation(conversationId: string): Promise<ConversationDetailDTO> {
  return apiFetch<ConversationDetailDTO>(`/conversations/${conversationId}`)
}

export interface ContactFilters {
  search?: string
  limit?: number
  offset?: number
}

export async function listContacts(filters: ContactFilters = {}): Promise<ListResponse<ContactDTO>> {
  const params = new URLSearchParams()
  if (filters.search) params.set('search', filters.search)
  if (filters.limit) params.set('limit', String(filters.limit))
  if (filters.offset) params.set('offset', String(filters.offset))
  const qs = params.toString()
  return apiFetch<ListResponse<ContactDTO>>(`/contacts${qs ? `?${qs}` : ''}`)
}

export async function listTeam(): Promise<ListResponse<TeamMemberDTO>> {
  return apiFetch<ListResponse<TeamMemberDTO>>('/team')
}

export async function updateTeamMember(
  userId: string,
  input: { role?: string; name?: string; team_id?: string },
): Promise<TeamMemberDTO> {
  return apiFetch<TeamMemberDTO>(`/team/${userId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
}

export async function deleteTeamMember(userId: string): Promise<{ id: string }> {
  return apiFetch<{ id: string }>(`/team/${userId}`, {
    method: 'DELETE',
  })
}

export async function createInvite(input: InviteInput): Promise<InviteDTO> {
  return apiFetch<InviteDTO>('/invites', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function listInvites(): Promise<ListResponse<InviteDTO>> {
  return apiFetch<ListResponse<InviteDTO>>('/invites')
}

export async function revokeInvite(inviteId: string): Promise<{ id: string; status: string }> {
  return apiFetch<{ id: string; status: string }>(`/invites/${inviteId}`, {
    method: 'DELETE',
  })
}

export async function inviteInfo(token: string): Promise<InviteInfo> {
  return apiFetch<InviteInfo>(`/invites/info?t=${encodeURIComponent(token)}`)
}

export async function acceptInvite(input: {
  token: string
  name: string
  password: string
}): Promise<{ email: string }> {
  return apiFetch<{ email: string }>('/invites/accept', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function listTeams(): Promise<ListResponse<TeamDTO>> {
  return apiFetch<ListResponse<TeamDTO>>('/teams')
}

export async function createTeam(name: string): Promise<TeamDTO> {
  return apiFetch<TeamDTO>('/teams', {
    method: 'POST',
    body: JSON.stringify({ name }),
  })
}

export async function updateTeam(teamId: string, name: string): Promise<TeamDTO> {
  return apiFetch<TeamDTO>(`/teams/${teamId}`, {
    method: 'PATCH',
    body: JSON.stringify({ name }),
  })
}

export async function deleteTeam(teamId: string): Promise<{ id: string }> {
  return apiFetch<{ id: string }>(`/teams/${teamId}`, {
    method: 'DELETE',
  })
}

export async function createContact(input: ContactInput): Promise<ContactDTO> {
  return apiFetch<ContactDTO>('/contacts', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function updateContact(contactId: string, input: ContactInput): Promise<ContactDTO> {
  return apiFetch<ContactDTO>(`/contacts/${contactId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
}

export async function deleteContact(contactId: string): Promise<{ id: string }> {
  return apiFetch<{ id: string }>(`/contacts/${contactId}`, {
    method: 'DELETE',
  })
}

export async function createAccount(input: WaAccountInput): Promise<WaAccountDTO> {
  return apiFetch<WaAccountDTO>('/accounts', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function updateAccount(
  accountId: string,
  input: WaAccountInput,
): Promise<WaAccountDTO> {
  return apiFetch<WaAccountDTO>(`/accounts/${accountId}`, {
    method: 'PATCH',
    body: JSON.stringify(input),
  })
}

export async function deleteAccount(accountId: string): Promise<{ id: string }> {
  return apiFetch<{ id: string }>(`/accounts/${accountId}`, {
    method: 'DELETE',
  })
}

export async function accountMeta(accountId: string): Promise<PhoneMetaResult> {
  return apiFetch<PhoneMetaResult>(`/accounts/${accountId}/meta`)
}

export async function analytics(range: string): Promise<AnalyticsResponse> {
  return apiFetch<AnalyticsResponse>(`/analytics?range=${range}`)
}

export async function listTemplates(): Promise<TemplatesResponse> {
  return apiFetch<TemplatesResponse>('/templates')
}

export async function createTemplate(input: TemplateInput): Promise<MessageTemplateDTO> {
  return apiFetch<MessageTemplateDTO>('/templates', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function deleteTemplate(templateId: string): Promise<{ id: string }> {
  return apiFetch<{ id: string }>(`/templates/${templateId}`, { method: 'DELETE' })
}

export async function syncTemplates(): Promise<TemplatesResponse> {
  return apiFetch<TemplatesResponse>('/templates/sync', { method: 'POST' })
}

/**
 * Uploads a file to a WhatsApp number's media store on Meta. The returned
 * media id is referenced when creating a template with a media header or when
 * overriding a template header media for a broadcast.
 */
export async function uploadMedia(input: {
  accountId: string
  file: File
}): Promise<MediaUploadResult> {
  const form = new FormData()
  form.append('account_id', input.accountId)
  form.append('file', input.file)
  return apiFetch<MediaUploadResult>('/media/upload', {
    method: 'POST',
    body: form,
  })
}

/**
 * Sends an approved template to a contact, creating (or reusing) the
 * conversation. This is how a chat is started from the contacts list.
 */
export async function sendTemplateToContact(input: TemplateSendInput): Promise<TemplateSendResult> {
  return apiFetch<TemplateSendResult>('/templates/send', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function listBroadcasts(): Promise<ListResponse<BroadcastDTO>> {
  return apiFetch<ListResponse<BroadcastDTO>>('/broadcasts')
}

export async function createBroadcast(input: BroadcastCreateInput): Promise<BroadcastDTO> {
  return apiFetch<BroadcastDTO>('/broadcasts', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function getBroadcast(broadcastId: string): Promise<BroadcastDetail> {
  return apiFetch<BroadcastDetail>(`/broadcasts/${broadcastId}`)
}

export async function cancelBroadcast(broadcastId: string): Promise<BroadcastDTO> {
  return apiFetch<BroadcastDTO>(`/broadcasts/${broadcastId}/cancel`, { method: 'POST' })
}

/**
 * Subscribes to live broadcast progress via the server-sent events stream
 * (`/broadcasts/{id}/events`). PocketBase's SSE routes require auth headers,
 * so this uses fetch + ReadableStream instead of the native EventSource (which
 * cannot send custom headers). The stream resolves when the server closes it.
 */
export async function subscribeBroadcastEvents(
  broadcastId: string,
  onEvent: (event: string, payload: BroadcastDTO) => void,
  signal: AbortSignal,
): Promise<void> {
  const headers = new Headers()
  const token = getStoredSession()?.token
  const orgId = getStoredOrgId()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (orgId) headers.set('X-Org-Id', orgId)

  const res = await fetch(`${API_BASE}/broadcasts/${broadcastId}/events`, { headers, signal })
  if (!res.ok || !res.body) {
    throw new ApiError(res.status, `Broadcast events stream failed (${res.status})`)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    const frames = buffer.split('\n\n')
    buffer = frames.pop() ?? ''
    for (const frame of frames) {
      let event = 'message'
      let data = ''
      for (const line of frame.split('\n')) {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        else if (line.startsWith('data:')) data = line.slice(5).trim()
      }
      if (!data) continue
      try {
        onEvent(event, JSON.parse(data) as BroadcastDTO)
      } catch {
        // Ignore malformed frames; keep streaming.
      }
    }
  }
}

export async function listAccounts(): Promise<ListResponse<WaAccountDTO>> {
  return apiFetch<ListResponse<WaAccountDTO>>('/accounts')
}

export async function assignConversation(
  conversationId: string,
  userId: string,
): Promise<{ id: string; assignee_id: string }> {
  return apiFetch(`/conversations/${conversationId}/assign`, {
    method: 'POST',
    body: JSON.stringify({ user_id: userId }),
  })
}

export async function assignRoundRobin(
  conversationId: string,
): Promise<{ id: string; assignee_id: string }> {
  return apiFetch(`/conversations/${conversationId}/assign`, {
    method: 'POST',
    body: JSON.stringify({ round_robin: true }),
  })
}

export async function markConversationRead(
  conversationId: string,
): Promise<{ id: string; unread_count: number }> {
  return apiFetch(`/conversations/${conversationId}/read`, {
    method: 'POST',
  })
}

export async function sendMessage(payload: SendMessagePayload): Promise<SendMessageResult> {
  return apiFetch<SendMessageResult>('/messages/send', {
    method: 'POST',
    body: JSON.stringify(payload),
  })
}

export async function sendMediaMessage(input: {
  conversationId: string
  file: File
  caption?: string
}): Promise<MediaMessageResult> {
  const form = new FormData()
  form.append('conversation_id', input.conversationId)
  form.append('file', input.file)
  if (input.caption) form.append('caption', input.caption)
  return apiFetch<MediaMessageResult>('/messages/media', {
    method: 'POST',
    body: form,
  })
}

export async function listNotifications(limit?: number): Promise<NotificationsResponse> {
  const qs = limit ? `?limit=${limit}` : ''
  return apiFetch<NotificationsResponse>(`/notifications${qs}`)
}

export async function unreadNotificationCount(): Promise<{ count: number }> {
  return apiFetch<{ count: number }>('/notifications/unread-count')
}

export async function markNotificationsRead(): Promise<{ updated: number }> {
  return apiFetch<{ updated: number }>('/notifications/read', { method: 'POST' })
}

export async function sendPresence(): Promise<void> {
  return apiFetch<void>('/presence', { method: 'POST' })
}

export async function getPushConfig(): Promise<{ vapid_public_key: string }> {
  return apiFetch<{ vapid_public_key: string }>('/push/config')
}

export async function pushSubscribe(body: {
  endpoint: string
  keys: { p256dh: string; auth: string }
}): Promise<void> {
  return apiFetch<void>('/push/subscribe', { method: 'POST', body: JSON.stringify(body) })
}

export async function pushUnsubscribe(endpoint: string): Promise<void> {
  return apiFetch<void>(`/push/subscribe?endpoint=${encodeURIComponent(endpoint)}`, {
    method: 'DELETE',
  })
}

/**
 * Starts an outbound call to a conversation. The server records the call as
 * ringing; the browser then negotiates media via signalCall.
 */
export async function startCall(conversationId: string): Promise<CallDTO> {
  return apiFetch<CallDTO>('/calls', {
    method: 'POST',
    body: JSON.stringify({ conversation_id: conversationId }),
  })
}

/**
 * Exchanges a WebRTC offer for the media session of a call with id `callId`.
 * The client is always the offerer; the server's pion bridge answers.
 */
export async function signalCall(callId: string, offerSdp: string): Promise<{ sdp: string }> {
  return apiFetch<{ sdp: string }>(`/calls/${callId}/signal`, {
    method: 'POST',
    body: JSON.stringify({ sdp: offerSdp }),
  })
}

/** Hangs up a call and tears down its media session. */
export async function endCall(callId: string): Promise<{ id: string; status: string }> {
  return apiFetch<{ id: string; status: string }>(`/calls/${callId}/end`, {
    method: 'POST',
  })
}

export async function listConversationCalls(conversationId: string): Promise<ListResponse<CallDTO>> {
  return apiFetch<ListResponse<CallDTO>>(`/conversations/${conversationId}/calls`)
}

/**
 * Subscribes to live call events for the active org via the `/calls/events`
 * SSE stream. Uses fetch + ReadableStream because PocketBase's SSE routes
 * require auth headers that the native EventSource cannot send.
 */
export async function subscribeCallEvents(
  onEvent: (event: string, payload: CallEventDTO) => void,
  signal: AbortSignal,
): Promise<void> {
  const headers = new Headers()
  const token = getStoredSession()?.token
  const orgId = getStoredOrgId()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  if (orgId) headers.set('X-Org-Id', orgId)

  const res = await fetch(`${API_BASE}/calls/events`, { headers, signal })
  if (!res.ok || !res.body) {
    throw new ApiError(res.status, `Call events stream failed (${res.status})`)
  }

  const reader = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''

  for (;;) {
    const { done, value } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })

    const frames = buffer.split('\n\n')
    buffer = frames.pop() ?? ''
    for (const frame of frames) {
      let event = 'message'
      let data = ''
      for (const line of frame.split('\n')) {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        else if (line.startsWith('data:')) data = line.slice(5).trim()
      }
      if (!data) continue
      try {
        onEvent(event, JSON.parse(data) as CallEventDTO)
      } catch {
        // Ignore malformed frames; keep streaming.
      }
    }
  }
}
