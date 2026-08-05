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
  tags?: string[]
  notes?: string
}

export interface ContactInput {
  name?: string
  phone?: string
  tags?: string[]
  notes?: string
}

export interface AccountDTO {
  id: string
  display_name: string
}

export interface TeamDTO {
  id: string
  name: string
  member_count: number
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

export interface ConversationDetailDTO extends ConversationDTO {
  in_window: boolean
  assignee_name?: string
  team_id?: string
  team_name?: string
}

export interface MessagesResponse {
  items: MessageDTO[]
}

export interface TeamMemberDTO {
  id: string
  name: string
  email: string
  role: string
  team_id?: string
  team_name?: string
}

export interface CreateTeamMemberResult {
  member: TeamMemberDTO
  generated_password: string
}

export interface WaAccountDTO {
  id: string
  display_name: string
  phone_number_id: string
  waba_id?: string
  status: string
  team_id?: string
  team_name?: string
}

export interface WaAccountInput {
  display_name?: string
  phone_number_id?: string
  access_token?: string
  verify_token?: string
  waba_id?: string
  status?: string
  team_id?: string
}

export interface PhoneMetaResult {
  ok: boolean
  error?: string
  info?: {
    id: string
    display_phone_number: string
    verified_name: string
    quality_rating: string
    messaging_limit_tier: string
    code_verification_status: string
    status: string
  }
}

export interface AnalyticsTotals {
  conversations: number
  cost: number
}

export interface AnalyticsAccount {
  id: string
  display_name: string
  phone_number_id: string
  conversations: number
  cost: number
}

export interface AnalyticsCategory {
  category: string
  conversations: number
  cost: number
}

export interface AnalyticsResponse {
  range: string
  start: number
  end: number
  totals: AnalyticsTotals
  accounts: AnalyticsAccount[]
  categories: AnalyticsCategory[]
  errors: string[]
}

export interface ListResponse<T> {
  items: T[]
}

export interface NotificationDTO {
  id: string
  kind: string
  body: string
  read: boolean
  conversation_id: string
  contact_name: string
  created: string
}

export interface NotificationsResponse {
  items: NotificationDTO[]
}

export interface InviteDTO {
  id: string
  email: string
  role: string
  team_id?: string
  team_name?: string
  status: string
  expires_at: string
  created_at: string
  token?: string
}

export interface InviteInput {
  email: string
  role: string
  team_id?: string
}

export interface InviteInfo {
  email: string
  role: string
  status: string
  org_name: string
  team_name?: string
  expired: boolean
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
