import { ApiError, getStoredOrgId, getStoredSession } from '../lib/authStore'
import type {
  ContactDTO,
  ConversationsResponse,
  CreateTeamMemberResult,
  ListResponse,
  MessagesResponse,
  SendMessagePayload,
  SendMessageResult,
  Session,
  TeamMemberDTO,
  WaAccountDTO,
} from './types'

const API_BASE = '/api/wa'

export interface InboxFilters {
  search?: string
  assignee?: string
  unassigned?: boolean
  status?: string
  limit?: number
  offset?: number
}

async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  if (init?.body != null) {
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

export async function listConversations(filters: InboxFilters): Promise<ConversationsResponse> {
  const params = new URLSearchParams()
  if (filters.search) params.set('search', filters.search)
  if (filters.assignee) params.set('assignee', filters.assignee)
  if (filters.unassigned) params.set('unassigned', 'true')
  if (filters.status) params.set('status', filters.status)
  if (filters.limit) params.set('limit', String(filters.limit))
  if (filters.offset) params.set('offset', String(filters.offset))
  const qs = params.toString()
  return apiFetch<ConversationsResponse>(`/inbox${qs ? `?${qs}` : ''}`)
}

export async function listMessages(conversationId: string): Promise<MessagesResponse> {
  return apiFetch<MessagesResponse>(`/conversations/${conversationId}/messages`)
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

export async function createTeamMember(input: {
  email: string
  name: string
  role: string
}): Promise<CreateTeamMemberResult> {
  return apiFetch<CreateTeamMemberResult>('/team', {
    method: 'POST',
    body: JSON.stringify(input),
  })
}

export async function updateTeamMember(
  userId: string,
  input: { role?: string; name?: string },
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
