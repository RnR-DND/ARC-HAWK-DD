import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import RiskChart from '../components/ui/RiskChart';

// Mock framer-motion
jest.mock('framer-motion', () => ({
  motion: {
    div: ({ children, ...props }: any) => <div {...props}>{children}</div>,
  },
}));

// Mock recharts to avoid SVG rendering issues in jsdom
jest.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: any) => <div data-testid="responsive-container">{children}</div>,
  PieChart: ({ children }: any) => <div data-testid="pie-chart">{children}</div>,
  Pie: () => null,
  Cell: () => null,
  BarChart: ({ children }: any) => <div data-testid="bar-chart">{children}</div>,
  Bar: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  Legend: () => null,
}));

describe('RiskChart', () => {
  const defaultProps = {
    byPiiType: { 'IN_AADHAAR': 45, 'EMAIL_ADDRESS': 30, 'IN_PAN': 15 },
    byAsset: { 'users_db': 50, 'logs_bucket': 20, 'config_files': 10 },
    byConfidence: { '> 90%': 40, '70-90%': 25, '< 70%': 15 },
  };

  it('renders chart sections', () => {
    render(<RiskChart {...defaultProps} />);
    expect(screen.getByText('PII Type Distribution')).toBeInTheDocument();
    expect(screen.getByText('Asset Risk Overview')).toBeInTheDocument();
    expect(screen.getByText('Confidence Distribution')).toBeInTheDocument();
  });

  it('displays summary cards', () => {
    render(<RiskChart {...defaultProps} />);
    expect(screen.getByText('High Risk')).toBeInTheDocument();
    expect(screen.getByText('Total Findings')).toBeInTheDocument();
    expect(screen.getByText('Data Sources')).toBeInTheDocument();
  });

  it('displays correct data source count', () => {
    render(<RiskChart {...defaultProps} />);
    expect(screen.getByText('3')).toBeInTheDocument(); // 3 assets
  });

  it('renders PII type legend items', () => {
    render(<RiskChart {...defaultProps} />);
    expect(screen.getByText('IN_AADHAAR')).toBeInTheDocument();
    expect(screen.getByText('EMAIL_ADDRESS')).toBeInTheDocument();
    expect(screen.getByText('IN_PAN')).toBeInTheDocument();
  });

  it('renders loading state', () => {
    const { container } = render(<RiskChart {...defaultProps} loading={true} />);
    const pulseElements = container.querySelectorAll('.animate-pulse');
    expect(pulseElements.length).toBeGreaterThan(0);
  });

  it('handles empty data', () => {
    render(<RiskChart byPiiType={{}} byAsset={{}} byConfidence={{}} />);
    expect(screen.getByText('PII Type Distribution')).toBeInTheDocument();
  });

  it('renders confidence bars', () => {
    render(<RiskChart {...defaultProps} />);
    expect(screen.getByText('> 90%')).toBeInTheDocument();
    expect(screen.getByText('70-90%')).toBeInTheDocument();
    expect(screen.getByText('< 70%')).toBeInTheDocument();
  });
});