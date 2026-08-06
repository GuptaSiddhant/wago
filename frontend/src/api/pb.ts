import PocketBase from 'pocketbase'
import { getStoredSession } from '../lib/authStore'

// Shared single PocketBase client. Only used by the SDK auth-store side-effect
// below so the user's token is kept fresh for PocketBase's own HTTP calls.
const pb = new PocketBase('/')

export function syncPocketBaseAuth(): void {
  const token = getStoredSession()?.token
  if (token) {
    pb.authStore.save(token, null)
  } else {
    pb.authStore.clear()
  }
}
