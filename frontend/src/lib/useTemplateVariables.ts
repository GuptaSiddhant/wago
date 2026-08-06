import { useMemo, useState } from 'react'
import type { Dispatch, SetStateAction } from 'react'

// Returns the highest {{n}} variable index found in a message-template body
// (e.g. "Hi {{1}}!" → 1). Returns 0 when the body has no variables.
export function maxTemplateVariableIndex(body: string): number {
  let max = 0
  for (const m of body.match(/\{\{(\d+)\}\}/g) ?? []) {
    const n = Number.parseInt(m.replace(/\D/g, ''), 10)
    if (!Number.isNaN(n) && n > max) max = n
  }
  return max
}

/**
 * Holds the fill-in sample value for every {{N}} variable in `body`.
 *
 * The returned array is kept exactly `maxVariableIndex(body)` long: slots are
 * appended when the body gains variables and truncated when they shrink. These
 * values drive both the live preview and the `example_values` payload Meta
 * requires when a template body contains variables.
 */
export function useTemplateVariables(body: string): {
  variableCount: number
  values: string[]
  setValues: Dispatch<SetStateAction<string[]>>
} {
  const variableCount = useMemo(() => maxTemplateVariableIndex(body), [body])

  // State is grown/shrunk synchronously on render so the array length always
  // matches `variableCount` even before an effect runs. This mirrors React's
  // "adjust state during render" pattern and is guarded to avoid infinite loops.
  const [values, setValues] = useState<string[]>([])
  const [prevCount, setPrevCount] = useState(0)
  if (variableCount !== prevCount) {
    setPrevCount(variableCount)
    setValues((prev) => {
      const next = [...prev]
      while (next.length < variableCount) next.push('')
      return next.slice(0, variableCount)
    })
  }

  return { variableCount, values, setValues }
}