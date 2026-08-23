import { useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { ImagePlus, Save, X } from 'lucide-react'
import { updateOrg } from '../../api/client'
import { useSession } from '../../lib/session'
import { Button } from '../../components/ui/Button'
import { FormError } from '../../components/ui/FormError'
import { OrgAvatar } from '../../components/OrgAvatar'
import { SelectField } from '../../components/ui/Select'
import { TextField } from '../../components/ui/TextField'

// WhatsApp Business verticals mirror the Meta Graph API `vertical` enum
// (excluding UNDEFINED / NOT_A_BIZ which aren't user-selectable).
const verticals = [
  'OTHER',
  'AUTO',
  'BEAUTY',
  'APPAREL',
  'EDU',
  'ENTERTAIN',
  'EVENT_PLAN',
  'FINANCE',
  'GROCERY',
  'GOVT',
  'HOTEL',
  'HEALTH',
  'NONPROFIT',
  'PROF_SERVICES',
  'RETAIL',
  'TRAVEL',
  'RESTAURANT',
]

const verticalLabels: Record<string, string> = {
  OTHER: 'Other',
  AUTO: 'Automotive',
  BEAUTY: 'Beauty',
  APPAREL: 'Apparel & Fashion',
  EDU: 'Education',
  ENTERTAIN: 'Entertainment',
  EVENT_PLAN: 'Event Planning',
  FINANCE: 'Finance',
  GROCERY: 'Grocery',
  GOVT: 'Government',
  HOTEL: 'Hotel & Lodging',
  HEALTH: 'Health & Wellness',
  NONPROFIT: 'Nonprofit',
  PROF_SERVICES: 'Professional Services',
  RETAIL: 'Retail',
  TRAVEL: 'Travel',
  RESTAURANT: 'Restaurant',
}

export function OrgPage() {
  const { org, refresh } = useSession()
  const [name, setName] = useState(org?.name ?? '')
  const [about, setAbout] = useState(org?.about ?? '')
  const [address, setAddress] = useState(org?.address ?? '')
  const [description, setDescription] = useState(org?.description ?? '')
  const [email, setEmail] = useState(org?.email ?? '')
  const [website1, setWebsite1] = useState(org?.websites?.[0] ?? '')
  const [website2, setWebsite2] = useState(org?.websites?.[1] ?? '')
  const [vertical, setVertical] = useState<string | null>(org?.vertical ?? null)
  const [picture, setPicture] = useState<File | null>(null)
  const [picturePreview, setPicturePreview] = useState<string | null>(null)
  const [removePicture, setRemovePicture] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  const canEdit = org?.role === 'owner' || org?.role === 'admin'

  const mutation = useMutation({
    mutationFn: () =>
      updateOrg({
        name: name.trim() || undefined,
        about: about.trim() ?? '',
        address: address.trim() ?? '',
        description: description.trim() ?? '',
        email: email.trim() ?? '',
        websites: [website1, website2].map((w) => w.trim()).filter(Boolean),
        vertical: vertical ?? undefined,
        profile_picture: picture ?? undefined,
        remove_picture: removePicture,
      }),
    onSuccess: async () => {
      setSaved(true)
      setPicture(null)
      setRemovePicture(false)
      if (picturePreview) {
        URL.revokeObjectURL(picturePreview)
        setPicturePreview(null)
      }
      await refresh()
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to update organization')
    },
  })

  function handlePicture(file: File | undefined) {
    setPicture(file ?? null)
    if (picturePreview) URL.revokeObjectURL(picturePreview)
    setPicturePreview(file ? URL.createObjectURL(file) : null)
    setRemovePicture(false)
    setSaved(false)
    setError(null)
  }

  function handleRemovePicture() {
    setPicture(null)
    if (picturePreview) URL.revokeObjectURL(picturePreview)
    setPicturePreview(null)
    setRemovePicture(true)
    setSaved(false)
    if (fileInputRef.current) fileInputRef.current.value = ''
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) return setError('Organization name is required')
    setError(null)
    setSaved(false)
    mutation.mutate()
  }

  const section = 'flex flex-col gap-3 rounded-xl border border-zinc-800 bg-zinc-900/50 p-3'

  if (!org) return null

  return (
    <div className="flex h-full min-h-0 flex-1 flex-col">
      <div className="flex-1 overflow-y-auto px-6 py-6">
        <div className="mx-auto flex max-w-2xl flex-col gap-6">
          <div>
            <h1 className="text-xl font-semibold text-zinc-100">Organization</h1>
            <p className="mt-1 text-sm text-zinc-500">
              Business details shown to WhatsApp contacts and used when syncing
              your number's business profile. {canEdit ? '' : 'Only owners and admins can edit these.'}
            </p>
          </div>

          <form onSubmit={handleSubmit} className="flex flex-col gap-5">
            <div className={section}>
              <TextField
                label="Organization name"
                value={name}
                onChange={(v) => {
                  setName(v)
                  setSaved(false)
                }}
                placeholder="Acme Inc."
                isRequired
                isDisabled={!canEdit}
              />

              <div>
                <span className="mb-1.5 block text-sm font-medium text-zinc-300">
                  Profile picture
                </span>
                {picturePreview || (!removePicture && org.profile_picture_url) ? (
                  <div className="flex items-center gap-3">
                    {picturePreview ? (
                      <img
                        src={picturePreview}
                        alt="Profile preview"
                        className="h-16 w-16 rounded-xl object-cover ring-1 ring-zinc-700"
                      />
                    ) : (
                      <OrgAvatar org={org} size={64} />
                    )}
                    <div className="flex flex-col gap-2">
                      <span className="text-xs text-zinc-400">
                        {picture?.name ?? 'Current picture'}
                      </span>
                      {canEdit ? (
                        <div className="flex gap-2">
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onPress={() => fileInputRef.current?.click()}
                          >
                            Replace
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            onPress={handleRemovePicture}
                          >
                            <X size={14} />
                            Remove
                          </Button>
                        </div>
                      ) : null}
                    </div>
                  </div>
                ) : canEdit ? (
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    className="flex h-16 w-full items-center justify-center gap-2 rounded-xl border border-dashed border-zinc-700 bg-zinc-900 text-sm text-zinc-500 transition hover:border-zinc-600 hover:text-zinc-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
                  >
                    <ImagePlus size={18} />
                    Upload picture (JPEG/PNG/WebP)
                  </button>
                ) : (
                  <p className="text-sm text-zinc-500">No profile picture set.</p>
                )}
                <input
                  ref={fileInputRef}
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  className="hidden"
                  disabled={!canEdit}
                  onChange={(e) => handlePicture(e.target.files?.[0])}
                />
              </div>
            </div>

            <div className={section}>
              <TextField
                label="About"
                value={about}
                onChange={(v) => {
                  setAbout(v)
                  setSaved(false)
                }}
                placeholder="Quick blurb shown in the chat header"
                maxLength={139}
                description="Up to 139 characters"
                isDisabled={!canEdit}
              />
              <TextField
                label="Description"
                value={description}
                onChange={(v) => {
                  setDescription(v)
                  setSaved(false)
                }}
                placeholder="What does your business do?"
                maxLength={512}
                description="Up to 512 characters"
                isDisabled={!canEdit}
              />
            </div>

            <div className={section}>
              <TextField
                label="Address"
                value={address}
                onChange={(v) => {
                  setAddress(v)
                  setSaved(false)
                }}
                placeholder="1 Hacker Way, Menlo Park, CA"
                maxLength={256}
                isDisabled={!canEdit}
              />
              <TextField
                label="Contact email"
                value={email}
                onChange={(v) => {
                  setEmail(v)
                  setSaved(false)
                }}
                type="email"
                placeholder="hello@acme.com"
                maxLength={128}
                isDisabled={!canEdit}
              />
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <TextField
                  label="Website 1"
                  value={website1}
                  onChange={(v) => {
                    setWebsite1(v)
                    setSaved(false)
                  }}
                  placeholder="https://acme.com"
                  type="url"
                  isDisabled={!canEdit}
                />
                <TextField
                  label="Website 2"
                  value={website2}
                  onChange={(v) => {
                    setWebsite2(v)
                    setSaved(false)
                  }}
                  placeholder="https://shop.acme.com"
                  type="url"
                  isDisabled={!canEdit}
                />
              </div>
            </div>

            <div className={section}>
              <SelectField
                label="Industry"
                ariaLabel="Industry"
                placeholder="Select industry…"
                options={verticals.map((v) => ({ id: v, label: verticalLabels[v] ?? v }))}
                selectedKey={vertical}
                isDisabled={!canEdit}
                onSelectionChange={(key) => {
                  setVertical(typeof key === 'string' ? key : null)
                  setSaved(false)
                }}
              />
            </div>

            {canEdit ? (
              <>
                <FormError message={error} />
                <div className="flex items-center gap-3">
                  <Button type="submit" isDisabled={mutation.isPending}>
                    <Save size={16} />
                    {mutation.isPending ? 'Saving…' : 'Save changes'}
                  </Button>
                  {saved ? (
                    <span className="text-sm text-emerald-400">Saved</span>
                  ) : null}
                </div>
              </>
            ) : (
              <p className="text-sm text-zinc-600">
                Your role ({org.role}) is read-only here. Ask an owner or admin
                to update the organization details.
              </p>
            )}
          </form>
        </div>
      </div>
    </div>
  )
}