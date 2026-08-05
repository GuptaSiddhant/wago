import { getPushConfig, pushSubscribe, pushUnsubscribe } from '../api/client'

// Stored key mirrors the endpoint we last told the backend, so we avoid
// re-registering the same subscription on every load.
const STORED_ENDPOINT_KEY = 'wago:push-endpoint'

export function registerServiceWorker(): void {
  if (!('serviceWorker' in navigator)) return
  if (import.meta.env.PROD) {
    window.addEventListener('load', () => {
      void navigator.serviceWorker.register('/sw.js')
    })
    return
  }
  // Also register in dev so push works against a local HTTPS tunnel, but don't
  // block the boot in the process.
  void navigator.serviceWorker.register('/sw.js').catch(() => {})
}

function urlBase64ToUint8Array(base64: string): Uint8Array<ArrayBuffer> {
  const padding = '='.repeat((4 - (base64.length % 4)) % 4)
  const b64 = (base64 + padding).replace(/-/g, '+').replace(/_/g, '/')
  const raw = atob(b64)
  const bytes = new Uint8Array(new ArrayBuffer(raw.length))
  for (let i = 0; i < raw.length; i++) bytes[i] = raw.charCodeAt(i)
  return bytes
}

async function saveSubscription(sub: PushSubscription): Promise<void> {
  const json = sub.toJSON()
  await pushSubscribe({
    endpoint: sub.endpoint,
    keys: { p256dh: json.keys?.p256dh ?? '', auth: json.keys?.auth ?? '' },
  })
  try {
    localStorage.setItem(STORED_ENDPOINT_KEY, sub.endpoint)
  } catch {
    /* ignore quota errors */
  }
}

async function submitSubscription(): Promise<void> {
  if (!isSecureContext) return
  if (!('serviceWorker' in navigator) || !('PushManager' in window)) return
  if (Notification.permission !== 'granted') return

  const reg = await navigator.serviceWorker.ready
  const existing = await reg.pushManager.getSubscription()
  if (existing && existing.endpoint === localStorage.getItem(STORED_ENDPOINT_KEY)) {
    return // already registered with the backend
  }
  if (existing) {
    await saveSubscription(existing)
    return
  }

  const config = await getPushConfig()
  if (!config.vapid_public_key) return
  const sub = await reg.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: urlBase64ToUint8Array(config.vapid_public_key),
  })
  await saveSubscription(sub)
}

// Syncs the browser's push subscription with the backend. Call after login and
// whenever notification permission is granted. Best-effort — never throws.
export async function syncPushSubscription(): Promise<void> {
  if (!isSecureContext) return
  try {
    await submitSubscription()
  } catch (err) {
    console.warn('Push subscription sync failed', err)
  }
}

// Wire up the service worker's pushsubscriptionchange message so a freshly
// issued subscription is re-registered without a reload.
export function listenForPushSubscriptionChanges(): void {
  if (!('serviceWorker' in navigator)) return
  navigator.serviceWorker.addEventListener('message', (event) => {
    if (event.data?.type !== 'push-subscription-changed') return
    const sub = event.data.subscription
    if (sub?.endpoint) {
      void saveSubscription(sub as unknown as PushSubscription)
    } else {
      try {
        localStorage.removeItem(STORED_ENDPOINT_KEY)
      } catch {
        /* ignore */
      }
    }
  })
}

export function canPush(): boolean {
  return (
    isSecureContext &&
    'serviceWorker' in navigator &&
    'PushManager' in window &&
    'Notification' in window &&
    Notification.permission === 'granted'
  )
}

export async function unsubscribePush(): Promise<void> {
  if (!isSecureContext || !('serviceWorker' in navigator)) return
  try {
    const reg = await navigator.serviceWorker.ready
    const sub = await reg.pushManager.getSubscription()
    if (sub) {
      await pushUnsubscribe(sub.endpoint).catch(() => {})
      await sub.unsubscribe()
    }
  } catch {
    /* best-effort */
  }
  try {
    localStorage.removeItem(STORED_ENDPOINT_KEY)
  } catch {
    /* ignore */
  }
}