import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/lib/queryKeys'
import { demosApi } from './demos.api'

export const demosQueries = {
  list: () =>
    queryOptions({
      queryKey: qk.demos.list(),
      queryFn: ({ signal }) => demosApi.list(signal),
      staleTime: Infinity,
    }),
}

export function useRunDemo() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (id: string) => demosApi.run(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.security.all })
    },
  })
}
