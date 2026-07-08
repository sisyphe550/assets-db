import { describe, it, expect } from 'vitest';
import { unwrapApiResponse } from '@/utils/api';

describe('unwrapApiResponse', () => {
  it('returns data when code is 0', () => {
    const result = unwrapApiResponse({ code: 0, message: 'ok', data: { id: 1 } });
    expect(result).toEqual({ id: 1 });
  });

  it('throws with code when code is non-zero', () => {
    expect(() =>
      unwrapApiResponse({ code: 40301, message: '账户已禁用', data: null }),
    ).toThrow('账户已禁用');

    try {
      unwrapApiResponse({ code: 40301, message: '账户已禁用', data: null });
    } catch (err) {
      expect((err as Error & { code?: number }).code).toBe(40301);
    }
  });
});
