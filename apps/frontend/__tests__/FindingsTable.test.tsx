import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import '@testing-library/jest-dom';
import FindingsTable from '../components/ui/FindingsTable';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
    tr: ({ children, ...props }: any) => <tr {...props}>{children}</tr>,
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}));

// Mock RemediationConfirmationModal
jest.mock('../components/remediation/RemediationConfirmationModal', () => ({
  RemediationConfirmationModal: () => null,
}));

// Mock remediation API
jest.mock('../services/remediation.api', () => ({
  remediationApi: { executeRemediation: jest.fn() },
}));

const mockFindings = [
  {
    id: 'finding-1',
    assetName: 'users_table',
    assetPath: 'postgres://main/public/users',
    field: 'email',
    piiType: 'EMAIL_ADDRESS',
    confidence: 0.95,
    risk: 'High' as const,
    sourceType: 'Database' as const,
  },
  {
    id: 'finding-2',
    assetName: 'config_file',
    assetPath: '/etc/app/config.yml',
    field: 'api_key',
    piiType: 'IN_AADHAAR',
    confidence: 0.88,
    risk: 'Critical' as const,
    sourceType: 'File' as const,
  },
];

describe('FindingsTable', () => {
  it('renders table with findings', () => {
    render(<FindingsTable findings={mockFindings} />);
    expect(screen.getByText('PII Findings')).toBeInTheDocument();
    expect(screen.getByText('users_table')).toBeInTheDocument();
    expect(screen.getByText('config_file')).toBeInTheDocument();
  });

  it('shows finding count', () => {
    render(<FindingsTable findings={mockFindings} />);
    expect(screen.getByText(/2 of 2 findings/)).toBeInTheDocument();
  });

  it('renders empty state', () => {
    render(<FindingsTable findings={[]} />);
    expect(screen.getByText('No Findings Yet')).toBeInTheDocument();
  });

  it('displays risk badges', () => {
    render(<FindingsTable findings={mockFindings} />);
    expect(screen.getAllByText('High')[0]).toBeInTheDocument();
    expect(screen.getAllByText('Critical')[0]).toBeInTheDocument();
  });

  it('displays PII types', () => {
    render(<FindingsTable findings={mockFindings} />);
    expect(screen.getByText('EMAIL_ADDRESS')).toBeInTheDocument();
    expect(screen.getByText('IN_AADHAAR')).toBeInTheDocument();
  });

  it('displays confidence as percentage', () => {
    render(<FindingsTable findings={mockFindings} />);
    expect(screen.getByText('95%')).toBeInTheDocument();
    expect(screen.getByText('88%')).toBeInTheDocument();
  });

  it('renders loading state', () => {
    const { container } = render(<FindingsTable findings={[]} loading={true} />);
    const pulseElements = container.querySelectorAll('.animate-pulse');
    expect(pulseElements.length).toBeGreaterThan(0);
  });

  it('shows column headers', () => {
    render(<FindingsTable findings={mockFindings} />);
    expect(screen.getByText('Asset')).toBeInTheDocument();
    expect(screen.getByText('PII Type')).toBeInTheDocument();
    expect(screen.getByText('Confidence')).toBeInTheDocument();
    expect(screen.getByText('Risk Level')).toBeInTheDocument();
    expect(screen.getByText('Source')).toBeInTheDocument();
  });

  it('shows critical and high count badges', () => {
    render(<FindingsTable findings={mockFindings} />);
    expect(screen.getByText('1 Critical')).toBeInTheDocument();
    expect(screen.getByText('1 High')).toBeInTheDocument();
  });

  it('renders search input', () => {
    render(<FindingsTable findings={mockFindings} />);
    expect(screen.getByPlaceholderText(/Search findings/)).toBeInTheDocument();
  });
});