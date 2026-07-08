import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import StatusTag from '@/components/common/StatusTag';

describe('StatusTag', () => {
  it('renders asset status labels', () => {
    render(<StatusTag type="asset" value={1} />);
    expect(screen.getByText('在库')).toBeInTheDocument();
  });

  it('renders unknown status fallback', () => {
    render(<StatusTag type="asset" value={99} />);
    expect(screen.getByText('未知(99)')).toBeInTheDocument();
  });
});
