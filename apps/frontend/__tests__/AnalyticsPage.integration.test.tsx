import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import AnalyticsPage from '../app/analytics/page';

// Mock the theme
jest.mock('@/design-system/theme', () => ({
  theme: {
    colors: {
      background: {
        primary: '#000000',
        card: '#1a1a1a'
      },
      border: {
        default: '#333333'
      },
      text: {
        primary: '#ffffff',
        secondary: '#cccccc'
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

// Mock Topbar component
jest.mock('@/components/Topbar', () => {
  return function MockTopbar() {
    return <div data-testid="topbar">Topbar</div>;
  };
});

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

// Mock fetch API
global.fetch = jest.fn();

describe('AnalyticsPage Integration', () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('renders loading state initially', () => {
    // Mock fetch to never resolve (loading state)
    (global.fetch as jest.Mock).mockImplementation(() => new Promise(() => { }));

    render(<AnalyticsPage />);

    expect(screen.getByText('Loading analytics...')).toBeInTheDocument();
  });

  it('renders analytics data after successful fetch', async () => {
    const mockHeatmapData = {
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
    };

    const mockTrendData = {
      timeline: [
        { date: '2026-01-10', total_pii: 50, critical_pii: 10 },
        { date: '2026-01-11', total_pii: 45, critical_pii: 8 }
      ],
      newly_exposed: 15,
      resolved: 20
    };

    (global.fetch as jest.Mock).mockImplementation((url) => {
      if (url.includes('heatmap')) return Promise.resolve({ ok: true, json: () => Promise.resolve(mockHeatmapData) });
      return Promise.resolve({ ok: true, json: () => Promise.resolve(mockTrendData) });
    });

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
    const mockHeatmapData = {
      rows: [{ asset_type: 'database', cells: [], total: 0 }],
      columns: []
    };

    const mockTrendData = {
      timeline: [],
      newly_exposed: 5,
      resolved: 10
    };

    (global.fetch as jest.Mock).mockImplementation((url) => {
      if (url.includes('heatmap')) return Promise.resolve({ ok: true, json: () => Promise.resolve(mockHeatmapData) });
      return Promise.resolve({ ok: true, json: () => Promise.resolve(mockTrendData) });
    });

    render(<AnalyticsPage />);

    await waitFor(() => {
      expect(screen.getByText('Newly Exposed (30d)')).toBeInTheDocument();
      expect(screen.getByText('Resolved (30d)')).toBeInTheDocument();
    });
  });

  it('renders heatmap table with correct structure', async () => {
    const mockHeatmapData = {
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
    };

    const mockTrendData = {
      timeline: [],
      newly_exposed: 0,
      resolved: 0
    };

    (global.fetch as jest.Mock).mockImplementation((input) => {
      const url = input?.toString() || '';
      if (url.includes('heatmap')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(mockHeatmapData) });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve(mockTrendData) });
    });

    render(<AnalyticsPage />);

    await waitFor(() => {
      expect(screen.getByText(/Asset Type/i)).toBeInTheDocument();
      expect(screen.getByText(/PAN/i)).toBeInTheDocument(); // column header
      expect(screen.getByText(/database/i)).toBeInTheDocument(); // row header
      expect(screen.getAllByText(/5/i)[0]).toBeInTheDocument(); // total
    });
  });

  it('handles API errors gracefully', async () => {
    (global.fetch as jest.Mock)
      .mockRejectedValueOnce(new Error('Network error'))
      .mockRejectedValueOnce(new Error('Network error'));

    // Mock console.error to avoid test output pollution
    const consoleSpy = jest.spyOn(console, 'error').mockImplementation(() => { });

    render(<AnalyticsPage />);

    await waitFor(() => {
      // Should still render the page structure even with errors
      expect(screen.getByTestId('topbar')).toBeInTheDocument();
    });

    consoleSpy.mockRestore();
  });

  it('renders tooltips with info icons', async () => {
    const mockHeatmapData = {
      rows: [{ asset_type: 'database', cells: [], total: 0 }],
      columns: []
    };

    const mockTrendData = {
      timeline: [],
      newly_exposed: 0,
      resolved: 0
    };

    (global.fetch as jest.Mock).mockImplementation((url) => {
      if (url.includes('heatmap')) return Promise.resolve({ ok: true, json: () => Promise.resolve(mockHeatmapData) });
      return Promise.resolve({ ok: true, json: () => Promise.resolve(mockTrendData) });
    });

    render(<AnalyticsPage />);

    await waitFor(() => {
      const infoIcons = screen.getAllByTestId('info-icon');
      expect(infoIcons.length).toBeGreaterThan(0);
    });
  });

  it('displays trend chart section', async () => {
    const mockHeatmapData = {
      rows: [],
      columns: []
    };

    const mockTrendData = {
      timeline: [
        { date: '2026-01-10', total_pii: 50, critical_pii: 10 }
      ],
      newly_exposed: 5,
      resolved: 3
    };

    (global.fetch as jest.Mock).mockImplementation((url) => {
      if (url.includes('heatmap')) return Promise.resolve({ ok: true, json: () => Promise.resolve(mockHeatmapData) });
      return Promise.resolve({ ok: true, json: () => Promise.resolve(mockTrendData) });
    });

    render(<AnalyticsPage />);

    await waitFor(() => {
      expect(screen.getByText('30-Day Risk Exposure Trend')).toBeInTheDocument();
    });
  });
});