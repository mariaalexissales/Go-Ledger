import { api, apiUrl, type ListResponse } from '@/lib/http/client'
import type {
  ClientIPMode,
  EventListParams,
  LimiterPolicy,
  OpsConfig,
  SecurityEvent,
  SecurityStats,
} from './security.types'

export const securityApi = {
  listEvents: (params: EventListParams, signal?: AbortSignal) =>
    api.get<ListResponse<SecurityEvent>>('/ops/events', params, signal),

  stats: (window: string, signal?: AbortSignal) =>
    api.get<SecurityStats>('/ops/stats', { window }, signal),

  config: (signal?: AbortSignal) => api.get<OpsConfig>('/ops/config', undefined, signal),

  setClientIPMode: (mode: ClientIPMode) =>
    api.put<OpsConfig>('/ops/config/client-ip-mode', { mode }),

  setLimiterPolicy: (policy: LimiterPolicy) =>
    api.put<OpsConfig>('/ops/config/limiter-policy', policy),

  resetEvents: () => api.post<void>('/ops/events/reset'),

  /** URL for the EventSource subscription; the browser opens it directly. */
  streamUrl: (params?: EventListParams) => apiUrl('/ops/events/stream', params),
}
