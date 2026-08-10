import { api, type ListResponse } from '@/lib/http/client'
import type { DemoMeta, DemoResult } from './demos.types'

export const demosApi = {
  list: (signal?: AbortSignal) => api.get<ListResponse<DemoMeta>>('/ops/demos', undefined, signal),

  run: (id: string, signal?: AbortSignal) =>
    api.post<DemoResult>(`/ops/demos/${id}/run`, undefined, signal),
}
