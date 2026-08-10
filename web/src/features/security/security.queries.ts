import { queryOptions, useMutation, useQueryClient } from '@tanstack/react-query'
import { qk } from '@/lib/queryKeys'
import { securityApi } from './security.api'
import type { ClientIPMode, EventListParams, LimiterPolicy } from './security.types'

export const securityQueries = {
  events: (params: EventListParams) =>
    queryOptions({
      queryKey: qk.security.events(params),
      queryFn: ({ signal }) => securityApi.listEvents(params, signal),
    }),

  stats: (window: string) =>
    queryOptions({
      queryKey: qk.security.stats(window),
      queryFn: ({ signal }) => securityApi.stats(window, signal),
      // The live stream drives most updates; this is a slow backstop for the
      // aggregate tiles.
      refetchInterval: 15_000,
    }),

  config: () =>
    queryOptions({
      queryKey: qk.security.config(),
      queryFn: ({ signal }) => securityApi.config(signal),
    }),
}

export function useSetClientIPMode() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (mode: ClientIPMode) => securityApi.setClientIPMode(mode),
    onSuccess: (config) => {
      queryClient.setQueryData(qk.security.config(), config)
      queryClient.invalidateQueries({ queryKey: qk.security.all })
    },
  })
}

export function useSetLimiterPolicy() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (policy: LimiterPolicy) => securityApi.setLimiterPolicy(policy),
    onSuccess: (config) => {
      queryClient.setQueryData(qk.security.config(), config)
      queryClient.invalidateQueries({ queryKey: qk.security.all })
    },
  })
}

export function useResetEvents() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: () => securityApi.resetEvents(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: qk.security.all })
    },
  })
}
