import type { Session } from '../api/types'

const SESSION_KEY = 'wago.session'
const ORG_KEY = 'wago.org'

export class ApiError extends Error {
  readonly status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

export function getStoredSession(): Session | null {
  const raw = localStorage.getItem(SESSION_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as Session
  } catch {
    return null
  }
}

export function setStoredSession(session: Session | null): void {
  if (session) {
    localStorage.setItem(SESSION_KEY, JSON.stringify(session))
  } else {
    localStorage.removeItem(SESSION_KEY)
  }
}

export function getStoredOrgId(): string | null {
  return localStorage.getItem(ORG_KEY)
}

export function setStoredOrgId(orgId: string): void {
  localStorage.setItem(ORG_KEY, orgId)
}

export function clearStoredOrgId(): void {
  localStorage.removeItem(ORG_KEY)
}
