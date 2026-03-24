import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import ScanStatusCard from '../components/ui/ScanStatusCard';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
    button: ({ children, ...props }: any) => <button {...props}>{children}</button>,
  },
}));

// Mock fetch for API calls
global.fetch = jest.fn();

describe('ScanStatusCard', () => {
  beforeEach(() => {
    (global.fetch as jest.Mock).mockReset();
  });

  it('renders idle state when no scanId', () => {
    render(<ScanStatusCard scanId={null} />);
    expect(screen.getAllByText('Scan Status')[0]).toBeInTheDocument();
  });

  it('renders with a scanId', () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        status: 'completed',
        created_at: '2026-01-01T00:00:00Z',
        completed_at: '2026-01-01T00:05:00Z',
        progress: 100,
      }),
    });

    render(<ScanStatusCard scanId="scan-123" />);
    expect(screen.getByText('Scan Status')).toBeInTheDocument();
  });

  it('shows scan ID when provided', () => {
    (global.fetch as jest.Mock).mockResolvedValueOnce({
      ok: true,
      json: async () => ({ status: 'idle' }),
    });

    render(<ScanStatusCard scanId="test-scan-456" />);
    expect(screen.getByText('test-scan-456')).toBeInTheDocument();
  });

  it('renders progress bar', () => {
    render(<ScanStatusCard scanId={null} />);
    expect(screen.getByText('Progress')).toBeInTheDocument();
  });

  it('renders recommended actions section', () => {
    render(<ScanStatusCard scanId={null} />);
    expect(screen.getByText('Recommended Next Steps')).toBeInTheDocument();
  });

  it('shows start scan button in idle state', () => {
    render(<ScanStatusCard scanId={null} />);
    expect(screen.getAllByText('Start Scan')[0]).toBeInTheDocument();
  });
});