import { Button, Label, ListBox, ListBoxItem, Popover, Select, SelectValue } from 'react-aria-components'
import type { Key, ListBoxItemRenderProps, SelectValueRenderProps } from 'react-aria-components'
import { Building2, Check, ChevronsUpDown } from 'lucide-react'
import { useSession } from '../lib/session'
import type { OrgSummary } from '../api/types'

export function OrgSwitcher() {
  const { session, org, selectOrg } = useSession()
  if (!session) return null

  if ((session.orgs?.length ?? 0) === 0) {
    return (
      <div className="flex items-center gap-2">
        <Building2 size={16} className="shrink-0 text-emerald-500" />
        <span className="text-sm font-medium text-zinc-200">No orgs</span>
      </div>
    )
  }

  return (
    <Select
      selectedKey={org?.id ?? null}
      onSelectionChange={(key: Key | null) => {
        if (typeof key === 'string') selectOrg(key)
      }}
      aria-label="Organization"
      className="w-full"
    >
      <Label className="sr-only">Organization</Label>
      <Button className="flex h-10 w-full items-center gap-2 rounded-xl border border-zinc-800 bg-zinc-900 px-3 text-sm text-zinc-100 outline-none transition hover:border-zinc-700 focus-visible:ring-2 focus-visible:ring-emerald-500/50">
        <Building2 size={16} className="shrink-0 text-emerald-500" />
        <SelectValue className="truncate text-left font-medium">
          {({ selectedText }: SelectValueRenderProps<unknown>) =>
            selectedText ?? 'Select org'
          }
        </SelectValue>
        <ChevronsUpDown size={14} className="ml-auto shrink-0 text-zinc-500" />
      </Button>
      <Popover className="w-56 rounded-xl border border-zinc-700 bg-zinc-900 p-1 shadow-xl shadow-black/40">
        <ListBox className="max-h-72 overflow-auto outline-none" items={session.orgs ?? []}>
          {(o: OrgSummary) => (
            <ListBoxItem
              id={o.id}
              textValue={o.name}
              className={({ isFocused, isSelected }: ListBoxItemRenderProps) =>
                `flex cursor-pointer items-center justify-between gap-2 rounded-lg px-3 py-2 text-sm outline-none ${
                  isFocused ? 'bg-zinc-800' : ''
                } ${isSelected ? 'text-emerald-400' : 'text-zinc-200'}`
              }
            >
              {({ isSelected }: ListBoxItemRenderProps) => (
                <>
                  <span className="flex min-w-0 items-center gap-2">
                    <span className="truncate">{o.name}</span>
                    <span className="text-[10px] uppercase tracking-wide text-zinc-500">
                      {o.role}
                    </span>
                  </span>
                  {isSelected ? <Check size={14} className="shrink-0" /> : null}
                </>
              )}
            </ListBoxItem>
          )}
        </ListBox>
      </Popover>
    </Select>
  )
}
