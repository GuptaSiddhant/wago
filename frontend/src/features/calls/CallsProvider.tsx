import type { ReactNode } from 'react'
import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from 'react'
import { endCall, signalCall, startCall, subscribeCallEvents } from '../../api/client'
import type { CallEventDTO } from '../../api/types'
import { useSession } from '../../lib/session'

/** The WebRTC media session backing any call the agent is involved in. */
interface ActiveSession {
  callId: string
  peer: RTCPeerConnection
  audio: HTMLAudioElement
  isAnswering: boolean
  direction: CallEventDTO['direction']
  phone: string
  name?: string
}

interface CallsContextValue {
  /** Inbound call currently ringing and awaiting an answer, if any. */
  ringing: CallEventDTO | null
  /** The call in progress (connected). */
  active: CallEventDTO | null
  /** True while a WebRTC session is being established. */
  connecting: boolean
  error: string | null
  answerCurrent: () => void
  declineCurrent: () => void
  hangup: () => void
  startOutbound: (conversationId: string) => Promise<void>
}

const CallsContext = createContext<CallsContextValue | null>(null)

export function useCalls(): CallsContextValue {
  const ctx = useContext(CallsContext)
  if (!ctx) throw new Error('useCalls must be used within a CallsProvider')
  return ctx
}

const RTC_CONFIG = { iceServers: [{ urls: 'stun:stun.l.google.com:19302' }] }

export function CallsProvider({ children }: { children: ReactNode }) {
  const { org } = useSession()
  const orgId = org?.id ?? ''

  const [ringing, setRinging] = useState<CallEventDTO | null>(null)
  const [active, setActive] = useState<CallEventDTO | null>(null)
  const [connecting, setConnecting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Refs hold the live peer/audio so start/end callbacks stay stable.
  const sessionRef = useRef<ActiveSession | null>(null)
  const attemptedRef = useRef<string | null>(null)
  const activeRef = useRef<CallEventDTO | null>(null)
  useEffect(() => {
    activeRef.current = active
  }, [active])

  function clearSession() {
    const s = sessionRef.current
    if (!s) return
    try {
      s.peer.getSenders().forEach((sender) => sender.track?.stop())
      s.peer.close()
    } catch {
      // already closed
    }
    s.audio.pause()
    s.audio.srcObject = null
    sessionRef.current = null
    attemptedRef.current = null
  }

  const startNegotiation = 
    (call: CallEventDTO) => {
      attemptedRef.current = call.id
      setError(null)
      setConnecting(true)
      const peer = new RTCPeerConnection(RTC_CONFIG)
      const audio = new Audio()
      audio.autoplay = true
      peer.ontrack = (e) => {
        audio.srcObject = e.streams[0]
        void audio.play().catch(() => {})
      }
      const session: ActiveSession = {
        callId: call.id,
        peer,
        audio,
        isAnswering: true,
        direction: call.direction,
        phone: call.phone ?? '',
        name: call.name,
      }
      sessionRef.current = session

      void (async () => {
        try {
          const offer = await peer.createOffer()
          await peer.setLocalDescription(offer)
          const answer = await signalCall(call.id, offer.sdp ?? '')
          await peer.setRemoteDescription({ type: 'answer', sdp: answer.sdp })
          setConnecting(false)
          setActive(call)
        } catch (err) {
          setConnecting(false)
          setError(err instanceof Error ? err.message : 'Failed to establish the call')
          clearSession()
        }
      })()
    }

  // Listen for live call events; negotiate inbound calls immediately.
  useEffect(() => {
    if (!orgId) return
    const controller = new AbortController()
    subscribeCallEvents((_event, payload) => {
      if (payload.direction === 'inbound' && payload.state === 'ringing') {
        setRinging(payload)
        // Only auto-negotiate when no session is already active.
        if (!sessionRef.current) {
          startNegotiation(payload)
        } else {
          // A session is running; keep the banner so a second agent can't
          // answer twice. The event defines its own state.
        }
      } else if (payload.state === 'ended' || payload.state === 'failed') {
        if (activeRef.current?.id === payload.id || sessionRef.current?.callId === payload.id) {
          clearSession()
          setActive(null)
        }
        setRinging((r) => (r?.id === payload.id ? null : r))
      }
    }, controller.signal).catch(() => {
      // Stream dropped; retried on next org change.
    })
    return () => {controller.abort()}
  }, [orgId])

  const answerCurrent = useCallback(() => {
    // The session is already negotiating as soon as an inbound event arrives;
    // answering simply keeps it (the offer was already sent). Surfaced for
    // parity with the UI so re-answering is a no-op.
    setRinging(null)
  }, [])

  const declineCurrent = useCallback(() => {
    const s = sessionRef.current
    if (s) void endCall(s.callId).catch(() => {})
    clearSession()
    setRinging(null)
  }, [])

  const hangup = useCallback(() => {
    const s = sessionRef.current
    if (s) void endCall(s.callId).catch(() => {})
    clearSession()
    setActive(null)
  }, [])

  const startOutbound = useCallback(
    async (conversationId: string) => {
      if (sessionRef.current) return
      setError(null)
      try {
        // Ask the server to create the ringing call, then negotiate like an
        // inbound one from the returned record.
        const call = await startCall(conversationId)
        const ev: CallEventDTO = {
          id: call.id,
          direction: call.direction,
          state: call.status,
          phone: call.phone,
          name: call.name,
        }
        startNegotiation(ev)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to start the call')
      }
    },
    [startNegotiation],
  )

  const value = useMemo<CallsContextValue>(
    () => ({
      ringing,
      active,
      connecting,
      error,
      answerCurrent,
      declineCurrent,
      hangup,
      startOutbound,
    }),
    [ringing, active, connecting, error, answerCurrent, declineCurrent, hangup, startOutbound],
  )

  return <CallsContext.Provider value={value}>{children}</CallsContext.Provider>
}