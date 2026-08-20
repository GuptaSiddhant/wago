import { orgPictureUrl } from '../lib/orgPictureUrl'
import type { OrgSummary } from '../api/types'

/** Renders an org's profile picture or a fallback monogram. */
export function OrgAvatar({ org, size = 32 }: { org: OrgSummary; size?: number }) {
  const pic = orgPictureUrl(org)
  if (pic) {
    return (
      <img
        src={pic}
        alt=""
        style={{ width: size, height: size }}
        className="shrink-0 rounded-lg object-cover"
      />
    )
  }
  return (
    <div
      style={{ width: size, height: size }}
      className="flex shrink-0 items-center justify-center rounded-lg bg-emerald-600 text-sm font-bold text-white"
    >
      W
    </div>
  )
}