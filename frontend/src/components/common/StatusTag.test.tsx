import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import StatusTag from '@/components/common/StatusTag';

describe('StatusTag', () => {
  it('renders asset status labels', () => {
    render(<StatusTag type="asset" value={1} />);
    expect(screen.getByText('在库')).toBeInTheDocument();
  });

  it('renders workflow type label', () => {
    render(<StatusTag type="workflowType" value={1} />);
    expect(screen.getByText('领用')).toBeInTheDocument();
  });

  it('renders inventory status label', () => {
    render(<StatusTag type="inventory" value={1} />);
    expect(screen.getByText('进行中')).toBeInTheDocument();
  });

  it('renders unknown asset status fallback', () => {
    render(<StatusTag type="asset" value={99} />);
    expect(screen.getByText('未知(99)')).toBeInTheDocument();
  });
});
