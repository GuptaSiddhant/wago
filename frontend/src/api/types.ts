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
  is_admin: boolean
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

export interface MessageMedia {
  media_id: string
  mime_type?: string
  filename?: string
  caption?: string
  url?: string
}

export interface MessageDTO {
  id: string
  wamid: string
  body: string
  direction: 'inbound' | 'outbound'
  status: string
  created: string
  kind?: string
  media?: MessageMedia
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

export interface WebhookStatusResult {
  ok: boolean
  callback_url?: string
  verify_token?: boolean
}

export interface WebhookConnectResult {
  ok: boolean
  error?: string
  callback_url?: string
  message?: string
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

export interface TemplateButton {
  type: 'QUICK_REPLY' | 'URL' | 'PHONE_NUMBER'
  text: string
  url?: string
  phone_number?: string
}

export interface MessageTemplateDTO {
  id: string
  account_id: string
  account_name?: string
  meta_id?: string
  name: string
  language: string
  category: string
  header_type: string
  header_text?: string
  header_media_type?: string
  header_media_id?: string
  header_media_name?: string
  body: string
  footer?: string
  buttons?: TemplateButton[]
  status: string
  meta_error?: string
  created?: string
}

export interface TemplateInput {
  account_id: string
  name: string
  language: string
  category: string
  header_type: string
  header_text?: string
  header_media_type?: string
  header_media_id?: string
  header_media_name?: string
  body: string
  footer?: string
  buttons?: TemplateButton[]
  example_values?: string[]
}

export interface TemplatesResponse {
  items: MessageTemplateDTO[]
  errors?: string[]
}

export interface BroadcastDTO {
  id: string
  name: string
  status: string
  account_id: string
  account_name?: string
  template_id: string
  template_name?: string
  header_media_type?: string
  header_media_id?: string
  header_media_name?: string
  rate_per_minute: number
  batch_size: number
  recipient_count: number
  sent_count: number
  failed_count: number
  pending: number
  sending: number
  created?: string
  started_at?: string
  finished_at?: string
}

export interface BroadcastCreateInput {
  name: string
  account_id: string
  template_id: string
  params?: Record<string, unknown>[]
  rate_per_minute: number
  batch_size: number
  contact_ids?: string[]
  all_contacts?: boolean
  header_media_type?: string
  header_media_id?: string
  header_media_name?: string
}

export interface BroadcastRecipient {
  id: string
  name: string
  phone: string
  status: string
  wamid?: string
  error?: string
  available?: string
}

export interface BroadcastDetail {
  broadcast: BroadcastDTO
  recipients: BroadcastRecipient[]
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

export interface MediaMessageResult {
  id: string
  wamid: string
  status: string
  kind: string
  in_window: boolean
}

export interface MediaUploadResult {
  media_id: string
  media_type: string // IMAGE | VIDEO | DOCUMENT
  filename: string
}

export interface TemplateSendInput {
  contact_id: string
  account_id: string
  template_id: string
  parameters?: Record<string, unknown>[]
}

export interface TemplateSendResult {
  id: string
  wamid: string
  status: string
  conversation_id: string
}

export type CallDirection = 'inbound' | 'outbound'
export type CallState = 'missed' | 'ringing' | 'active' | 'ended' | 'failed'

export interface CallDTO {
  id: string
  conversation_id: string
  contact_id: string
  account_id: string
  direction: CallDirection
  status: CallState
  phone: string
  name?: string
  duration: number
  started_at?: string
  ended_at?: string
  created: string
}

/** Live call event pushed over the /calls/events SSE stream. */
export interface CallEventDTO {
  id: string
  direction: CallDirection
  state: string
  caller_id?: string
  phone?: string
  name?: string
}
