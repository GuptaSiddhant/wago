export interface UserSummary {
  id: string
  email: string
  name: string
}

export interface OrgSummary {
  id: string
  name: string
  role: string
}

export interface Session {
  token: string
  user: UserSummary
  isAdmin: boolean
  orgs: OrgSummary[]
}

export interface ContactDTO {
  id: string
  name: string
  phone: string
}

export interface AccountDTO {
  id: string
  display_name: string
}

export interface MessageDTO {
  id: string
  wamid: string
  body: string
  direction: 'inbound' | 'outbound'
  status: string
  created: string
}

export interface ConversationDTO {
  id: string
  contact: ContactDTO
  whatsapp_account: AccountDTO
  assignee_id: string
  unread_count: number
  last_message_at: string
  status: string
  last_message?: MessageDTO
}

export interface ConversationsResponse {
  items: ConversationDTO[]
}

export interface MessagesResponse {
  items: MessageDTO[]
}

export interface TeamMemberDTO {
  id: string
  name: string
  email: string
  role: string
}

export interface WaAccountDTO {
  id: string
  display_name: string
  phone_number_id: string
  status: string
}

export interface ListResponse<T> {
  items: T[]
}

export interface SendMessageResult {
  id: string
  wamid: string
  status: string
  in_window: boolean
}

export interface TemplateParam {
  type: string
  text?: string
  [key: string]: unknown
}

export interface SendMessagePayload {
  conversation_id: string
  body?: string
  template?: {
    name: string
    language: string
    parameters?: Record<string, unknown>[]
  }
}
