import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { listAccounts, listTemplates, sendTemplateToContact } from '../../api/client'
import type { ContactDTO } from '../../api/types'
import { useSession } from '../../lib/session'
import { useTemplateVariables } from '../../lib/useTemplateVariables'
import { Button } from '../../components/ui/Button'
import { FormError } from '../../components/ui/FormError'
import { ModalDialog } from '../../components/ui/Modal'
import { SelectField } from '../../components/ui/Select'
import { TextField } from '../../components/ui/TextField'
import { TemplatePreview } from '../templates/TemplatePreview'

export function TemplateSendDialog({
  contact,
  onClose,
}: {
  contact: ContactDTO
  onClose: () => void
}) {
  const { org } = useSession()
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const orgId = org?.id ?? ''

  const accountsQuery = useQuery({
    queryKey: ['accounts', orgId],
    queryFn: () => listAccounts(),
    enabled: orgId !== '',
  })
  const templatesQuery = useQuery({
    queryKey: ['templates', orgId],
    queryFn: () => listTemplates(),
    enabled: orgId !== '',
  })

  const accounts = accountsQuery.data?.items ?? []
  const approved = (templatesQuery.data?.items ?? []).filter((t) => t.status === 'APPROVED')

  const [accountId, setAccountId] = useState<string | null>(null)
  const [templateId, setTemplateId] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const effectiveAccountId = accountId ?? accounts[0]?.id ?? null

  const accountTemplates = useMemo(
    () => approved.filter((t) => t.account_id === effectiveAccountId),
    [approved, effectiveAccountId],
  )
  const selectedTemplate = approved.find((t) => t.id === templateId) ?? null

  const { variableCount, values, setValues } = useTemplateVariables(selectedTemplate?.body ?? '')

  const mutation = useMutation({
    mutationFn: () =>
      sendTemplateToContact({
        contact_id: contact.id,
        account_id: effectiveAccountId ?? '',
        template_id: templateId ?? '',
        parameters: values.map((v) => ({ type: 'text', text: v })),
      }),
    onSuccess: (result) => {
      void queryClient.invalidateQueries({ queryKey: ['conversations', orgId] })
      onClose()
      void navigate({ to: '/inbox', search: { conv: result.conversation_id } })
    },
    onError: (err) => {
      setError(err instanceof Error ? err.message : 'Failed to send template')
    },
  })

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!effectiveAccountId) return setError('Select a WhatsApp number')
    if (!templateId) return setError('Select an approved template')
    setError(null)
    mutation.mutate()
  }

  return (
    <ModalDialog
      isOpen
      onOpenChange={(open) => !open && onClose()}
      title={`Send template to ${contact.name || contact.phone}`}
    >
      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <SelectField
            label="WhatsApp number"
            options={accounts.map((a) => ({ id: a.id, label: a.display_name || a.phone_number_id }))}
            selectedKey={effectiveAccountId}
            onSelectionChange={(key) => {
              setAccountId(String(key))
              setTemplateId(null)
            }}
          />
          <SelectField
            label="Approved template"
            options={accountTemplates.map((t) => ({ id: t.id, label: t.name }))}
            selectedKey={templateId}
            onSelectionChange={(key) => setTemplateId(String(key))}
            placeholder={accountTemplates.length ? 'Select…' : 'No approved templates'}
            isDisabled={!accountId || accountTemplates.length === 0}
          />
        </div>

        {selectedTemplate ? (
          <div>
            <TemplatePreview
              headerText={
                selectedTemplate.header_type === 'TEXT' ? selectedTemplate.header_text : undefined
              }
              headerMedia={
                selectedTemplate.header_type === 'MEDIA' && selectedTemplate.header_media_type
                  ? {
                      media_type: selectedTemplate.header_media_type,
                      filename: selectedTemplate.header_media_name,
                    }
                  : undefined
              }
              body={selectedTemplate.body}
              footer={selectedTemplate.footer}
              buttons={selectedTemplate.buttons}
              values={values}
            />
            {variableCount > 0 ? (
              <div className="mt-3 grid gap-2 sm:grid-cols-2">
                {Array.from({ length: variableCount }, (_, i) => (
                  <TextField
                    key={i}
                    label={`{{${i + 1}}}`}
                    value={values[i] ?? ''}
                    onChange={(v) => setValues((prev) => prev.map((p, j) => (j === i ? v : p)))}
                    placeholder={`Value ${i + 1}`}
                  />
                ))}
              </div>
            ) : null}
          </div>
        ) : (
          <p className="rounded-lg border border-amber-500/20 bg-amber-500/5 px-3 py-2 text-sm text-amber-300">
            {approved.length === 0
              ? 'No approved templates yet — submit one and wait for Meta approval before messaging.'
              : 'Select an approved template to preview it.'}
          </p>
        )}

        <FormError message={error} />

        <div className="flex justify-end gap-2 pt-1">
          <Button type="button" variant="ghost" onPress={onClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            isDisabled={mutation.isPending || !templateId}
          >
            {mutation.isPending ? 'Sending…' : 'Send template'}
          </Button>
        </div>
      </form>
    </ModalDialog>
  )
}
