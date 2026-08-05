const CACHE_NAME = 'wago-shell-v1'

self.addEventListener('install', () => {
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      const keys = await caches.keys()
      await Promise.all(keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k)))
      await self.clients.claim()
    })(),
  )
})

// Network-first with cache fallback so the SPA still loads offline once visited.
self.addEventListener('fetch', (event) => {
  const { request } = event
  if (request.method !== 'GET' || !request.url.startsWith(self.location.origin)) return

  event.respondWith(
    (async () => {
      try {
        const response = await fetch(request)
        if (response && response.status === 200 && response.type === 'basic') {
          const cache = await caches.open(CACHE_NAME)
          void cache.put(request, response.clone())
        }
        return response
      } catch {
        const cached = await caches.match(request)
        if (cached) return cached
        if (request.mode === 'navigate') {
          const index = await caches.match('/')
          if (index) return index
        }
        throw new Error('offline')
      }
    })(),
  )
})

self.addEventListener('push', (event) => {
  let data = {}
  try {
    data = event.data ? event.data.json() : {}
  } catch {
    data = { body: event.data ? event.data.text() : '' }
  }

  event.waitUntil(
    (async () => {
      // If the user is actively looking at the app, the page shows the
      // notification in-app — don't duplicate it as an OS banner.
      const clients = await self.clients.matchAll({ type: 'window', includeUncontrolled: true })
      if (clients.some((c) => 'focus' in c)) return

      await self.registration.showNotification(data.title || 'WaGo', {
        body: data.body || '',
        icon: '/icon-192.png',
        badge: '/icon-192.png',
        tag: data.conversation_id ? `conv-${data.conversation_id}` : undefined,
        renotify: true,
        data: { conversation_id: data.conversation_id, url: `/inbox?conv=${data.conversation_id || ''}` },
      })
    })(),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const target = event.notification.data?.url || '/'
  event.waitUntil(
    (async () => {
      const clients = await self.clients.matchAll({ type: 'window', includeUncontrolled: true })
      for (const client of clients) {
        if ('focus' in client) {
          await client.focus()
          await client.navigate(target)
          return
        }
      }
      await self.clients.openWindow(target)
    })(),
  )
})

// Push subscription changes (e.g. after an expiration). The page re-registers
// and re-saves the new subscription via the backend, so here we just surface
// the fresh subscription to any open client.
self.addEventListener('pushsubscriptionchange', () => {
  const notifyClient = async () => {
    const registration = await self.registration
    const subscription = await registration.pushManager.getSubscription()
    const clients = await self.clients.matchAll({ type: 'window', includeUncontrolled: true })
    for (const client of clients) {
      client.postMessage({ type: 'push-subscription-changed', subscription: subscription ? subscription.toJSON() : null })
    }
  }
  event.waitUntil(notifyClient())
})
