import { Phone, PhoneOff, PhoneIncoming, PhoneOutgoing } from 'lucide-react'
import { useCalls } from './CallsProvider'
import { Avatar } from '../../components/ui/Avatar'
import { Button } from '../../components/ui/Button'

/**
 * Overlay for the calls feature: an incoming-call banner while an inbound call
 * is ringing, and a compact active-call window while a session is connected.
 * Rendered from the app shell so it is visible across every page.
 */
export function CallOverlay() {
  const { ringing, active, connecting, error, answerCurrent, declineCurrent, hangup } = useCalls()

  return (
    <>
      {ringing && !active ? <IncomingBanner onAnswer={answerCurrent} onDecline={declineCurrent} /> : null}
      {active ? <ActiveCallWindow connecting={connecting} onHang={hangup} error={error} /> : null}
    </>
  )
}

function IncomingBanner({
  onAnswer,
  onDecline,
}: {
  onAnswer: () => void
  onDecline: () => void
}) {
  const { ringing } = useCalls()
  const contactName = ringing?.name || ringing?.phone || 'Unknown'

  return (
    <div className="fixed inset-x-0 top-4 z-50 flex justify-center px-4">
      <div className="w-full max-w-sm rounded-2xl border border-zinc-800 bg-zinc-900/95 p-4 shadow-2xl backdrop-blur">
        <div className="flex items-center gap-4">
          <Avatar name={contactName} size={44} />
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 text-sm font-semibold text-zinc-100">
              <PhoneIncoming size={14} className="text-emerald-400" />
              <span className="truncate">{contactName}</span>
            </div>
            <p className="truncate text-xs text-zinc-500">{ringing?.phone}</p>
            <p className="mt-0.5 text-[11px] text-emerald-400/80">Incoming call…</p>
          </div>
          <div className="flex shrink-0 gap-2">
            <Button
              variant="danger"
              size="icon"
              aria-label="Decline call"
              onPress={onDecline}
            >
              <PhoneOff size={18} />
            </Button>
            <Button size="icon" aria-label="Answer call" onPress={onAnswer}>
              <Phone size={18} />
            </Button>
          </div>
        </div>
      </div>
    </div>
  )
}

function ActiveCallWindow({
  connecting,
  error,
  onHang,
}: {
  connecting: boolean
  error: string | null
  onHang: () => void
}) {
  const { active } = useCalls()
  const contactName = active?.name || active?.phone || 'Unknown'
  const outbound = active?.direction === 'outbound'
  const CallIcon = outbound ? PhoneOutgoing : PhoneIncoming

  return (
    <div className="fixed right-4 bottom-4 z-50 w-72 rounded-2xl border border-zinc-800 bg-zinc-900/95 p-4 shadow-2xl backdrop-blur">
      <div className="flex items-center gap-3">
        <Avatar name={contactName} size={40} />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2 text-sm font-semibold text-zinc-100">
            <CallIcon size={14} className={outbound ? 'text-sky-400' : 'text-emerald-400'} />
            <span className="truncate">{contactName}</span>
          </div>
          <p className="truncate text-xs text-zinc-500">{active?.phone}</p>
        </div>
      </div>

      <div className="mt-3 rounded-xl bg-zinc-950/60 px-3 py-2 text-center text-xs text-zinc-400">
        {connecting ? 'Connecting…' : error ?? 'Call connected'}
      </div>

      <div className="mt-3 flex justify-center">
        <Button variant="danger" onPress={onHang}>
          <PhoneOff size={16} />
          End call
        </Button>
      </div>
    </div>
  )
}