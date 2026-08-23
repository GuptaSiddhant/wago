import type { AnyRouteMatch } from '@tanstack/react-router'

declare module '@tanstack/react-router' {
  interface StaticDataRouteOption {
    title?: string
  }
}

export function pageTitleFromMatches(
  matches: ReadonlyArray<AnyRouteMatch>,
): string | undefined {
  for (let i = matches.length - 1; i >= 0; i--) {
    const data = matches[i]?.staticData as { title?: string } | undefined
    if (data?.title) return data.title
  }
  return undefined
}
