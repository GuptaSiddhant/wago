import { getStoredSession } from './authStore'
import type { OrgSummary } from '../api/types'

/** Builds an org profile picture URL that authenticates via query param token. */
export function orgPictureUrl(org: OrgSummary): string | undefined {
  if (!org.profile_picture_url) return undefined
  const token = getStoredSession()?.token
  return token ? `${org.profile_picture_url}?authorization=${encodeURIComponent(token)}` : undefined
}