import { fetchServerSentEvents } from '@tanstack/ai-react'
import { getStoredOrgId, getStoredSession } from '../lib/authStore'

/**
 * SSE connection to the Wago AI chat endpoint. Auth headers are resolved per
 * request so a token refresh or org switch is picked up automatically. The
 * active conversation is passed per message via the `data` option of
 * `sendMessage`, which the backend surfaces as the transcript context.
 */
export const aiChatConnection = fetchServerSentEvents('/api/wa/ai/chat', async () => {
  const session = getStoredSession()
  return {
    headers: {
      Authorization: session?.token ? `Bearer ${session.token}` : '',
      'X-Org-Id': getStoredOrgId() ?? '',
    },
  }
})