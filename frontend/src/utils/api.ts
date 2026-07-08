import type { ApiResponse } from '@/types/common';

export function unwrapApiResponse<T>(response: ApiResponse<T>): T {
  if (response.code !== 0) {
    const err = new Error(response.message) as Error & { code?: number };
    err.code = response.code;
    throw err;
  }
  return response.data;
}
