import { QueryClient } from '@tanstack/react-query'

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 1,
      // Cached results are reused for a while and kept around, making
      // navigation between pages instant (data is shown immediately while
      // background refetches bring it up to date) instead of flashing a blank
      // page. Note: keepPreviousData is NOT set globally because switching the
      // active org changes query keys and would briefly flash the previous
      // org's data.
      staleTime: 30_000,
      gcTime: 10 * 60_000,
      refetchOnWindowFocus: false,
    },
  },
})
