import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import MetricCards from '../components/ui/MetricCards';

// Mock framer-motion to avoid animation issues in tests
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
  AnimatePresence: ({ children }: any) => <>{children}</>,
}));

describe('MetricCards', () => {
  const defaultProps = {
    totalPII: 150,
    highRiskFindings: 12,
    assetsHit: 8,
    actionsRequired: 25,
  };

  it('renders all metric cards', () => {
    render(<MetricCards {...defaultProps} />);
    expect(screen.getByText('PII Instances Found')).toBeInTheDocument();
    expect(screen.getByText('Critical Findings')).toBeInTheDocument();
    expect(screen.getByText('Data Sources Impacted')).toBeInTheDocument();
    expect(screen.getByText('Remediation Tasks')).toBeInTheDocument();
  });

  it('displays metric values', () => {
    render(<MetricCards {...defaultProps} />);
    expect(screen.getByText('150')).toBeInTheDocument();
    expect(screen.getByText('12')).toBeInTheDocument();
    expect(screen.getByText('8')).toBeInTheDocument();
    expect(screen.getByText('25')).toBeInTheDocument();
  });

  it('renders zero values correctly', () => {
    render(<MetricCards totalPII={0} highRiskFindings={0} assetsHit={0} actionsRequired={0} />);
    const zeros = screen.getAllByText('0');
    expect(zeros.length).toBe(4);
  });

  it('renders loading state', () => {
    const { container } = render(<MetricCards {...defaultProps} loading={true} />);
    const pulseElements = container.querySelectorAll('.animate-pulse');
    expect(pulseElements.length).toBeGreaterThan(0);
  });

  it('renders large numbers with locale formatting', () => {
    render(<MetricCards totalPII={1500000} highRiskFindings={0} assetsHit={0} actionsRequired={0} />);
    expect(screen.getByText('1,500,000')).toBeInTheDocument();
  });
});