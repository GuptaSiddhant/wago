import type { ReactNode } from 'react'
import type { TemplateButton } from '../../api/types'

export interface TemplatePreviewProps {
  headerText?: string
  /** Media header info (type + filename); rendered as a placeholder tile. */
  headerMedia?: { media_type: string; filename?: string }
  body: string
  footer?: string
  buttons?: TemplateButton[]
  /** Sample values substituted for {{1}}, {{2}}, … in the preview. */
  values?: string[]
  /** Override the business name shown above the bubble. */
  businessName?: string
}

// renderBody splits on {{n}} placeholders and renders the substituted value
// with a highlight so the template variables are obvious in the preview.
function renderBody(body: string, values: string[]): ReactNode[] {
  const parts = body.split(/(\{\{\d+\}\})/)
  return parts.map((part, i) => {
    const m = part.match(/^\{\{(\d+)\}\}$/)
    if (m) {
      const n = Number.parseInt(m[1], 10)
      return (
        <span key={i} className="rounded bg-emerald-500/20 px-0.5 font-medium text-emerald-300">
          {values[n - 1] ?? `{{${n}}}`}
        </span>
      )
    }
    return <span key={i}>{part}</span>
  })
}

export function TemplatePreview({
  headerText,
  headerMedia,
  body,
  footer,
  buttons,
  values = [],
  businessName = 'WaGo',
}: TemplatePreviewProps) {
  const hasTextHeader = Boolean(headerText?.trim())
  const hasMediaHeader = Boolean(headerMedia)
  const hasFooter = Boolean(footer?.trim())
  const hasButtons = (buttons?.length ?? 0) > 0

  return (
    <div className="flex flex-col gap-2 rounded-2xl bg-canvas p-4 ring-1 ring-edge">
      <div className="flex items-center gap-2 text-xs text-ink-faint">
        <span className="flex h-5 w-5 items-center justify-center rounded-full bg-emerald-600 text-[10px] font-bold text-white">
          {businessName.charAt(0).toUpperCase()}
        </span>
        <span className="font-medium text-ink-muted">{businessName}</span>
        <span className="ml-auto">✓✓</span>
      </div>

      <div className="max-w-[280px] overflow-hidden rounded-xl rounded-tr-sm bg-emerald-600/15 ring-1 ring-emerald-500/20">
        <div className="px-3 py-2 text-sm text-ink">
          {hasMediaHeader ? (
            <>
              <div className="mb-2 flex items-center gap-2 rounded-lg bg-emerald-600/20 px-3 py-2 text-xs text-emerald-300 ring-1 ring-emerald-500/30">
                <span className="font-semibold uppercase">
                  {headerMedia!.media_type.toUpperCase()}
                </span>
                <span className="truncate text-ink-muted">{headerMedia!.filename ?? 'Media'}</span>
              </div>
              <div className="mb-2 h-px bg-emerald-500/30" />
            </>
          ) : null}

          {hasTextHeader ? (
            <>
              <p className="mb-1 whitespace-pre-wrap text-sm font-semibold text-ink">
                {headerText}
              </p>
              <div className="mb-2 h-px bg-emerald-500/30" />
            </>
          ) : null}

          <p className="whitespace-pre-wrap break-words text-sm leading-relaxed">
            {renderBody(body, values)}
          </p>

          {hasFooter ? (
            <>
              <div className="mt-2 h-px bg-emerald-500/30" />
              <p className="mt-1 text-xs text-ink-muted">{footer}</p>
            </>
          ) : null}
        </div>

        {hasButtons ? (
          <>
            <div className="h-px bg-emerald-500/30" />
            <div className="flex flex-col">
              {buttons!.map((b, i) => (
                <button
                  key={i}
                  type="button"
                  tabIndex={-1}
                  className={`px-3 py-2 text-center text-sm font-medium text-emerald-300 ${
                    i < buttons!.length - 1 ? 'border-b border-emerald-500/20' : ''
                  }`}
                >
                  {b.text}
                </button>
              ))}
            </div>
          </>
        ) : null}
      </div>

      <p className="text-xs text-ink-faint">Template preview · variables are highlighted</p>
    </div>
  )
}
