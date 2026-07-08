import { describe, it, expect } from 'vitest';
import { formatPrice, formatDate } from '@/utils/format';

describe('format utils', () => {
  it('formats price in CNY', () => {
    expect(formatPrice(150000)).toContain('150');
    expect(formatPrice(150000)).toContain('¥');
  });

  it('formats ISO date', () => {
    expect(formatDate('2025-09-01T00:00:00+08:00')).toBe('2025-09-01');
  });
});
