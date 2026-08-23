import { useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Image, Paperclip, Plus, Trash2, X } from 'lucide-react'
import { createTemplate, uploadMedia } from '../../api/client'
import type { TemplateButton, WaAccountDTO } from '../../api/types'
import { useSession } from '../../lib/session'
import { useTemplateVariables } from '../../lib/useTemplateVariables'
import { Button } from '../../components/ui/Button'
import { FormError } from '../../components/ui/FormError'
import { ModalDialog } from '../../components/ui/Modal'
import { SelectField } from '../../components/ui/Select'
import { TextField } from '../../components/ui/TextField'
import { TemplatePreview } from './TemplatePreview'

const LANGUAGES = [
  'en_US',
  'en_GB',
  'es_ES',
  'es_MX',
  'pt_BR',
  'fr_FR',
  'de_DE',
  'it_IT',
  'hi_IN',
  'id_ID',
  'ar_AR',
  'tr_TR',
]

const CATEGORIES = ['MARKETING', 'UTILITY', 'AUTHENTICATION']

const inputBase =
  'w-full px-3 py-2 rounded-xl bg-zinc-900 border border-zinc-700 text-zinc-100 text-sm placeholder:text-zinc-500 outline-none transition focus:border-emerald-500 focus:ring-2 focus:ring-emerald-500/30'

export function TemplateForm({
  accounts,
  onDone,
}: {
  accounts: WaAccountDTO[]
  onDone: () => void
}) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const orgId = org?.id ?? ''

  const [accountId, setAccountId] = useState<string | null>(accounts[0]?.id ?? null)
  const [name, setName] = useState('')
  const [language, setLanguage] = useState('en_US')
  const [category, setCategory] = useState('MARKETING')
  const [headerType, setHeaderType] = useState('NONE')
  const [headerText, setHeaderText] = useState('')
  const [headerMediaType, setHeaderMediaType] = useState('')
  const [headerMediaId, setHeaderMediaId] = useState('')
  const [headerMediaName, setHeaderMediaName] = useState('')
  const [body, setBody] = useState('')
  const [footer, setFooter] = useState('')
  const [buttons, setButtons] = useState<TemplateButton[]>([])
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const { variableCount, values, setValues } = useTemplateVariables(body)

  const mutation = useMutation({
    mutationFn: () =>
      createTemplate({
        account_id: accountId ?? '',
        name,
        language,
        category,
        header_type: headerType,
        header_text: headerText,
        header_media_type: headerType === 'MEDIA' ? headerMediaType : undefined,
        header_media_id: headerType === 'MEDIA' ? headerMediaId : undefined,
        header_media_name: headerType === 'MEDIA' ? headerMediaName : undefined,
        body,
        footer,
        buttons: buttons.filter((b) => b.text.trim() !== ''),
        example_values: values,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['templates', orgId] })
      onDone()
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to submit template')
    },
  })

  const uploadMutation = useMutation({
    mutationFn: (file: File) => uploadMedia({ accountId: accountId ?? '', file }),
    onSuccess: (result) => {
      setHeaderMediaType(result.media_type)
      setHeaderMediaId(result.media_id)
      setHeaderMediaName(result.filename)
      setError(null)
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to upload media')
    },
  })

  function handleFile(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file || !accountId) {
      setError(accountId ? 'Choose a file to attach' : 'Select the WhatsApp number first')
      return
    }
    uploadMutation.mutate(file)
  }

  function handleSubmit(e: FormEvent) {
    e.preventDefault()
    if (!accountId) {
      setError('Select the WhatsApp number the template belongs to')
      return
    }
    if (!name.trim()) {
      setError('Template name is required')
      return
    }
    if (!body.trim()) {
      setError('Body text is required')
      return
    }
    if (headerType === 'TEXT' && !headerText.trim()) {
      setError('Header text is required when the header type is Text')
      return
    }
    if (headerType === 'MEDIA') {
      if (!headerMediaId) {
        setError('Attach a media file when the header type is Media')
        return
      }
      if (uploadMutation.isPending) {
        setError('Still uploading the media file, please wait')
        return
      }
    }
    setError(null)
    mutation.mutate()
  }

  const previewButtons = buttons.filter((b) => b.text.trim() !== '')

  return (
    <ModalDialog
      isOpen
      onOpenChange={(open) => !open && onDone()}
      title="Create message template"
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <SelectField
            label="WhatsApp number"
            options={accounts.map((a) => ({
              id: a.id,
              label: a.display_name || a.phone_number_id,
            }))}
            selectedKey={accountId}
            onSelectionChange={(key) => setAccountId(String(key))}
          />
          <SelectField
            label="Category"
            options={CATEGORIES.map((c) => ({ id: c, label: c }))}
            selectedKey={category}
            onSelectionChange={(key) => setCategory(String(key))}
          />
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <TextField
            label="Template name"
            value={name}
            onChange={setName}
            placeholder="flash_sale_announcement"
            description="Lowercase letters, numbers and underscores"
          />
          <SelectField
            label="Language"
            options={LANGUAGES.map((l) => ({ id: l, label: l }))}
            selectedKey={language}
            onSelectionChange={(key) => setLanguage(String(key))}
          />
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <SelectField
            label="Header type"
            options={[
              { id: 'NONE', label: 'No header' },
              { id: 'TEXT', label: 'Text' },
              { id: 'MEDIA', label: 'Media (image, video or document)' },
            ]}
            selectedKey={headerType}
            onSelectionChange={(key) => setHeaderType(String(key))}
          />
          {headerType === 'TEXT' ? (
            <TextField
              label="Header text"
              value={headerText}
              onChange={setHeaderText}
              placeholder="Hi {{1}}"
              description="Max 60 characters"
            />
          ) : headerType === 'MEDIA' ? (
            <div className="flex flex-col justify-end gap-1.5">
              {headerMediaId ? (
                <div className="flex items-center gap-2 rounded-xl bg-zinc-900 border border-zinc-700 px-3 py-2">
                  <Image size={15} className="shrink-0 text-emerald-400" />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate text-sm text-zinc-200">{headerMediaName}</span>
                    <span className="block text-xs text-zinc-500">
                      {headerMediaType} header
                    </span>
                  </span>
                  <button
                    type="button"
                    aria-label="Remove media"
                    className="shrink-0 rounded-full p-1 text-zinc-500 hover:bg-zinc-800 hover:text-red-400"
                    onClick={() => {
                      setHeaderMediaId('')
                      setHeaderMediaType('')
                      setHeaderMediaName('')
                    }}
                  >
                    <X size={14} />
                  </button>
                </div>
              ) : (
                <>
                  <button
                    type="button"
                    onClick={() => fileInputRef.current?.click()}
                    disabled={uploadMutation.isPending}
                    className="flex items-center justify-center gap-2 rounded-xl bg-zinc-900 border border-dashed border-zinc-600 px-3 py-2 text-sm text-zinc-400 transition hover:border-emerald-500 hover:text-emerald-400 disabled:opacity-60 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-emerald-500/60"
                  >
                    <Paperclip size={15} />
                    {uploadMutation.isPending ? 'Uploading…' : 'Attach image, video or document'}
                  </button>
                  <span className="text-xs text-zinc-500">
                    Uploaded once here and attached on every send.
                  </span>
                </>
              )}
              <input
                ref={fileInputRef}
                type="file"
                className="hidden"
                accept="image/*,video/mp4,video/3gpp,video/quicktime,application/pdf"
                onChange={handleFile}
              />
            </div>
          ) : (
            <div />
          )}
        </div>

        <label className="flex flex-col gap-1.5">
          <span className="text-sm font-medium text-zinc-300">Body</span>
          <textarea
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={4}
            placeholder={'Hi {{1}}, our flash sale starts Friday!\nUse code {{2}} for 20% off.'}
            className={inputBase}
          />
          <span className="text-xs text-zinc-500">
            Use {'{{1}}'}, {'{{2}}'}… for variables. Sample values below power the preview and Meta
            example payload.
          </span>
        </label>

        {variableCount > 0 ? (
          <div className="flex flex-col gap-2 rounded-xl border border-zinc-800 bg-zinc-900/40 p-3">
            <span className="text-sm font-medium text-zinc-300">Sample variable values</span>
            <div className="grid gap-2 sm:grid-cols-2">
              {Array.from({ length: variableCount }, (_, i) => (
                <TextField
                  key={i}
                  label={`{{${i + 1}}}`}
                  value={values[i] ?? ''}
                  onChange={(v) =>
                    setValues((prev) => prev.map((p, j) => (j === i ? v : p)))
                  }
                  placeholder={`Sample value ${i + 1}`}
                />
              ))}
            </div>
          </div>
        ) : null}

        <TextField
          label="Footer"
          value={footer}
          onChange={setFooter}
          placeholder="Tap to view full terms and conditions"
          description="Optional · max 60 characters"
        />

        <div className="flex flex-col gap-2">
          <div className="flex items-center justify-between">
            <span className="text-sm font-medium text-zinc-300">Buttons</span>
            <div className="flex gap-1">
              <Button
                size="sm"
                variant="secondary"
                onPress={() => setButtons((b) => [...b, { type: 'QUICK_REPLY', text: '' }])}
              >
                <Plus size={13} /> Quick reply
              </Button>
              <Button
                size="sm"
                variant="secondary"
                onPress={() => setButtons((b) => [...b, { type: 'URL', text: '', url: '' }])}
              >
                <Plus size={13} /> Link
              </Button>
            </div>
          </div>
          {buttons.map((b, i) => (
            <div key={i} className="flex items-start gap-2 rounded-xl border border-zinc-800 bg-zinc-900/40 p-3">
              <div className="grid flex-1 gap-2 sm:grid-cols-2">
                <TextField
                  label="Button text"
                  value={b.text}
                  onChange={(v) => setButtons((arr) => arr.map((x, j) => (j === i ? { ...x, text: v } : x)))}
                  placeholder={b.type === 'QUICK_REPLY' ? 'Yes, please' : 'Visit us'}
                  isRequired
                />
                {b.type === 'URL' ? (
                  <TextField
                    label="URL"
                    value={b.url ?? ''}
                    onChange={(v) => setButtons((arr) => arr.map((x, j) => (j === i ? { ...x, url: v } : x)))}
                    placeholder="https://example.com"
                  />
                ) : (
                  <div className="hidden sm:block" />
                )}
              </div>
              <Button
                size="icon"
                variant="ghost"
                aria-label="Remove button"
                onPress={() => setButtons((arr) => arr.filter((_, j) => j !== i))}
                className="mt-6 shrink-0"
              >
                <Trash2 size={15} className="text-zinc-500 hover:text-red-400" />
              </Button>
            </div>
          ))}
          {buttons.length === 0 ? (
            <p className="text-xs text-zinc-600">
              Optional — add quick reply or call-to-action buttons.
            </p>
          ) : null}
        </div>

        <div>
          <span className="text-sm font-medium text-zinc-300">Preview</span>
          <div className="mt-2">
            <TemplatePreview
              headerText={headerType === 'TEXT' ? headerText : undefined}
              headerMedia={
                headerType === 'MEDIA' && headerMediaType
                  ? { media_type: headerMediaType, filename: headerMediaName }
                  : undefined
              }
              body={body || 'Your template body will appear here…'}
              footer={footer}
              buttons={previewButtons}
              values={values}
            />
          </div>
        </div>

        <FormError message={error} />

        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" variant="ghost" onPress={onDone}>
            Cancel
          </Button>
          <Button type="submit" isDisabled={mutation.isPending}>
            {mutation.isPending ? 'Submitting…' : 'Submit for review'}
          </Button>
        </div>
      </form>
    </ModalDialog>
  )
}
