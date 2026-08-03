import PocketBase from 'pocketbase'
import { getStoredSession } from '../lib/authStore'

export const pb = new PocketBase('/')

export function syncPocketBaseAuth(): void {
  const token = getStoredSession()?.token
  if (token) {
    pb.authStore.save(token, null)
  } else {
    pb.authStore.clear()
  }
}

export function currentOrgId(): string | null {
  return localStorage.getItem('wago.org')
}
