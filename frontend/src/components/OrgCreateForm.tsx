import { useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation } from '@tanstack/react-query'
import { ImagePlus, X } from 'lucide-react'
import { createOrg } from '../api/client'
import type { OrgSummary } from '../api/types'
import { useSession } from '../lib/session'
import { Button } from './ui/Button'
import { FormError } from './ui/FormError'
import { SelectField } from './ui/Select'
import { TextField } from './ui/TextField'

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

interface OrgCreateFormProps {
  /** Called after the org exists, memberships refreshed, and it is selected. */
  onCreated: (org: OrgSummary) => void | Promise<void>
  onCancel?: () => void
  submitLabel?: string
}

/**
 * Shared organization creation form (superadmin-only endpoint). Used by the
 * first-run onboarding picker and any "new org" dialog.
 */
export function OrgCreateForm({ onCreated, onCancel, submitLabel }: OrgCreateFormProps) {
  const { refresh, selectOrg } = useSession()
  const [name, setName] = useState('')
  const [about, setAbout] = useState('')
  const [address, setAddress] = useState('')
  const [description, setDescription] = useState('')
  const [email, setEmail] = useState('')
  const [website1, setWebsite1] = useState('')
  const [website2, setWebsite2] = useState('')
  const [vertical, setVertical] = useState<string | null>(null)
  const [picture, setPicture] = useState<File | null>(null)
  const [picturePreview, setPicturePreview] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [error, setError] = useState<string | null>(null)

  const mutation = useMutation({
    mutationFn: () =>
      createOrg({
        name: name.trim(),
        about: about.trim() || undefined,
        address: address.trim() || undefined,
        description: description.trim() || undefined,
        email: email.trim() || undefined,
        websites: [website1, website2].map((w) => w.trim()).filter(Boolean),
        vertical: vertical ?? undefined,
        profile_picture: picture ?? undefined,
      }),
    onSuccess: async (org: OrgSummary) => {
      // Refresh memberships so the new org appears in the switcher, then jump
      // into it so it becomes the active org on first use.
      await refresh()
      selectOrg(org.id)
      await onCreated(org)
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to create organization')
    },
  })

  function handlePicture(file: File | undefined) {
    setPicture(file ?? null)
    if (picturePreview) URL.revokeObjectURL(picturePreview)
    setPicturePreview(file ? URL.createObjectURL(file) : null)
    setError(null)
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!name.trim()) return setError('Organization name is required')
    setError(null)
    mutation.mutate()
  }

  const section =
    'flex flex-col gap-3 rounded-xl border border-zinc-800 bg-zinc-950/40 p-3'

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <div className={section}>
        <TextField
          label="Organization name"
          value={name}
          onChange={setName}
          placeholder="Acme Inc."
          isRequired
          autoFocus
        />

        <div>
          <span className="mb-1.5 block text-sm font-medium text-zinc-300">
            Profile picture
          </span>
          {picturePreview ? (
            <div className="flex items-center gap-3">
              <img
                src={picturePreview}
                alt="Profile preview"
                className="h-16 w-16 rounded-xl object-cover ring-1 ring-zinc-700"
              />
              <div className="flex flex-col gap-2">
                <span className="text-xs text-zinc-400">
                  {picture?.name} ({Math.max(1, Math.round((picture?.size ?? 0) / 1024))} KB)
                </span>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onPress={() => {
                    handlePicture(undefined)
                    if (fileInputRef.current) fileInputRef.current.value = ''
                  }}
                >
                  <X size={14} />
                  Remove
                </Button>
              </div>
            </div>
          ) : (
            <button
              type="button"
              onClick={() => fileInputRef.current?.click()}
              className="flex h-16 w-full items-center justify-center gap-2 rounded-xl border border-dashed border-zinc-700 bg-zinc-900 text-sm text-zinc-500 transition hover:border-zinc-600 hover:text-zinc-300"
            >
              <ImagePlus size={18} />
              Upload picture (JPEG/PNG/WebP)
            </button>
          )}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="hidden"
            onChange={(e) => handlePicture(e.target.files?.[0])}
          />
        </div>
      </div>

      <div className={section}>
        <TextField
          label="About"
          value={about}
          onChange={setAbout}
          placeholder="Quick blurb shown in the chat header"
          maxLength={139}
          description="Up to 139 characters"
        />
        <TextField
          label="Description"
          value={description}
          onChange={setDescription}
          placeholder="What does your business do?"
          maxLength={512}
          description="Up to 512 characters"
        />
      </div>

      <div className={section}>
        <TextField
          label="Address"
          value={address}
          onChange={setAddress}
          placeholder="1 Hacker Way, Menlo Park, CA"
          maxLength={256}
        />
        <TextField
          label="Contact email"
          value={email}
          onChange={setEmail}
          type="email"
          placeholder="hello@acme.com"
          maxLength={128}
        />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <TextField
            label="Website 1"
            value={website1}
            onChange={setWebsite1}
            placeholder="https://acme.com"
            type="url"
          />
          <TextField
            label="Website 2"
            value={website2}
            onChange={setWebsite2}
            placeholder="https://shop.acme.com"
            type="url"
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
          onSelectionChange={(key) => setVertical(typeof key === 'string' ? key : null)}
        />
      </div>

      <FormError message={error} />
      <div className="flex justify-end gap-2 pt-1">
        {onCancel ? (
          <Button type="button" variant="ghost" onPress={onCancel}>
            Cancel
          </Button>
        ) : null}
        <Button type="submit" isDisabled={mutation.isPending}>
          {mutation.isPending ? 'Creating…' : (submitLabel ?? 'Create organization')}
        </Button>
      </div>
    </form>
  )
}
