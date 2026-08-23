import { useNavigate } from '@tanstack/react-router'
import { OrgCreateForm } from './OrgCreateForm'
import { ModalDialog } from './ui/Modal'

/**
 * Dialog wrapper around the shared org-create form. After creation the new
 * org is selected automatically (handled inside OrgCreateForm).
 */
export function NewOrgDialog({ onClose }: { onClose: () => void }) {
  const navigate = useNavigate()
  return (
    <ModalDialog isOpen onOpenChange={(open) => !open && onClose()} title="New organization">
      <OrgCreateForm
        onCancel={onClose}
        onCreated={() => {
          // The form already refreshed memberships and selected the new org;
          // if the user had no orgs before, land them in the app properly.
          void navigate({ to: '/inbox' })
          onClose()
        }}
      />
    </ModalDialog>
  )
}
