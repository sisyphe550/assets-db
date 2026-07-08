import type { ApiResponse } from '@/types/common';

export function unwrapApiResponse<T>(response: ApiResponse<T>): T {
  if (response.code !== 0) {
    const err = new Error(response.message) as Error & { code?: number };
    err.code = response.code;
    throw err;
  }
  return response.data;
}

export function getApiErrorCode(error: unknown): number | undefined {
  if (!error || typeof error !== 'object') return undefined;
  const e = error as { data?: { code?: number }; code?: number };
  return e.data?.code ?? e.code;
}
