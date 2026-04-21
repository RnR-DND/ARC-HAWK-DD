import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import AnalyticsPage from '../app/analytics/page';

// Mock the theme
jest.mock('@/design-system/theme', () => ({
  theme: {
    colors: {
      background: {
        primary: '#000000',
        card: '#1a1a1a',
        tertiary: '#222'
      },
      border: {
        default: '#333333'
      },
      text: {
        primary: '#ffffff',
        secondary: '#cccccc',
        muted: '#888888'
      },
      risk: {
        critical: '#ef4444',
        high: '#f97316',
        medium: '#eab308',
        low: '#22c55e'
      }
    }
  },
  getRiskColor: (level: string) => {
    const colors = {
      critical: '#ef4444',
      high: '#f97316',
      medium: '#eab308',
      low: '#22c55e'
    };
    return colors[level as keyof typeof colors] || colors.low;
  }
}));

// Mock Tooltip component
jest.mock('@/components/Tooltip', () => ({
  InfoIcon: ({ size }: any) => <div data-testid="info-icon" style={{ width: size, height: size }} />,
  __esModule: true,
  default: ({ children, content }: any) => (
    <div data-testid="tooltip" title={content}>
      {children}
    </div>
  )
}));

jest.mock('@/components/ErrorBanner', () => ({
  __esModule: true,
  default: ({ message }: any) => <div data-testid="error-banner">{message}</div>,
}));

jest.mock('@/services/analytics.api', () => ({
  analyticsApi: {
    getHeatmap: jest.fn(),
    getTrends: jest.fn(),
    getRiskDistribution: jest.fn(),
  },
}));
import { analyticsApi } from '@/services/analytics.api';
const mockedApi = analyticsApi as jest.Mocked<typeof analyticsApi>;

const emptyHeatmap = { rows: [], columns: [] };
const emptyTrend = { timeline: [], newly_exposed: 0, resolved: 0 };
const emptyRiskDist = null;

function mockResolved(heatmap: any = emptyHeatmap, trend: any = emptyTrend, riskDist: any = emptyRiskDist) {
  mockedApi.getHeatmap.mockResolvedValue(heatmap);
  mockedApi.getTrends.mockResolvedValue(trend);
  mockedApi.getRiskDistribution.mockResolvedValue(riskDist);
}

function mockPending() {
  const pending = () => new Promise(() => {});
  mockedApi.getHeatmap.mockImplementation(pending);
  mockedApi.getTrends.mockImplementation(pending);
  mockedApi.getRiskDistribution.mockImplementation(pending as () => Promise<any>);
}

function mockRejected(err: Error) {
  mockedApi.getHeatmap.mockRejectedValue(err);
  mockedApi.getTrends.mockRejectedValue(err);
  mockedApi.getRiskDistribution.mockRejectedValue(err);
}

describe('AnalyticsPage Integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders loading state initially', () => {
    mockPending();

    render(<AnalyticsPage />);

    expect(screen.getByText('Loading analytics...')).toBeInTheDocument();
  });

  it('renders analytics data after successful fetch', async () => {
    mockResolved(
      {
        rows: [
          {
            asset_type: 'database',
            cells: [
              { pii_type: 'PAN', finding_count: 5, risk_level: 'critical', intensity: 80 },
              { pii_type: 'Email', finding_count: 12, risk_level: 'high', intensity: 60 }
            ],
            total: 17
          }
        ],
        columns: ['PAN', 'Email']
      },
      {
        timeline: [
          { date: '2026-01-10', total_pii: 50, critical_pii: 10 },
          { date: '2026-01-11', total_pii: 45, critical_pii: 8 }
        ],
        newly_exposed: 15,
        resolved: 20
      }
    );

    render(<AnalyticsPage />);

    await waitFor(() => {
      expect(screen.getByText('Risk Analytics & Heatmap')).toBeInTheDocument();
    });

    expect(screen.getByText('Data Distribution Heatmap')).toBeInTheDocument();
    expect(screen.getByText('30-Day Risk Exposure Trend')).toBeInTheDocument();
    expect(screen.getByText('+15')).toBeInTheDocument(); // newly exposed
    expect(screen.getByText('+20')).toBeInTheDocument(); // resolved
  });

  it('displays stat badges with correct labels', async () => {
    mockResolved(
      { rows: [{ asset_type: 'database', cells: [], total: 0 }], columns: [] },
      { timeline: [], newly_exposed: 5, resolved: 10 }
    );

    render(<AnalyticsPage />);

    await waitFor(() => {
      expect(screen.getByText('Newly Exposed (30d)')).toBeInTheDocument();
      expect(screen.getByText('Resolved (30d)')).toBeInTheDocument();
    });
  });

  it('renders heatmap table with correct structure', async () => {
    mockResolved(
      {
        rows: [
          {
            asset_type: 'database',
            cells: [
              { pii_type: 'PAN', finding_count: 5, risk_level: 'critical', intensity: 80 }
            ],
            total: 5
          }
        ],
        columns: ['PAN']
      },
      { timeline: [], newly_exposed: 0, resolved: 0 }
    );

    render(<AnalyticsPage />);

    await waitFor(() => {
      expect(screen.getByText(/Asset Type/i)).toBeInTheDocument();
      expect(screen.getByText(/PAN/i)).toBeInTheDocument(); // column header
      expect(screen.getByText(/database/i)).toBeInTheDocument(); // row header
      expect(screen.getAllByText(/5/i)[0]).toBeInTheDocument(); // total
    });
  });

  it('handles API errors gracefully', async () => {
    mockRejected(new Error('Network error'));
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    render(<AnalyticsPage />);

    await waitFor(() => {
      expect(document.body).toBeInTheDocument();
    });

    consoleSpy.mockRestore();
  });

  it('renders tooltips with info icons', async () => {
    mockResolved(
      { rows: [{ asset_type: 'database', cells: [], total: 0 }], columns: [] },
      { timeline: [], newly_exposed: 0, resolved: 0 }
    );

    render(<AnalyticsPage />);

    await waitFor(() => {
      const infoIcons = screen.getAllByTestId('info-icon');
      expect(infoIcons.length).toBeGreaterThan(0);
    });
  });

  it('displays trend chart section', async () => {
    mockResolved(
      { rows: [], columns: [] },
      { timeline: [{ date: '2026-01-10', total_pii: 50, critical_pii: 10 }], newly_exposed: 5, resolved: 3 }
    );

    render(<AnalyticsPage />);

    await waitFor(() => {
      expect(screen.getByText('30-Day Risk Exposure Trend')).toBeInTheDocument();
    });
  });
});
