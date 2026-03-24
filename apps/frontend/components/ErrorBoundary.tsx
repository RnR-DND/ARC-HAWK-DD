'use client';

import React, { Component, ReactNode } from 'react';
import { theme } from '@/design-system/theme';

interface Props {
  children: ReactNode;
  fallback?: ReactNode;
}

interface State {
  hasError: boolean;
  error?: Error;
  errorInfo?: React.ErrorInfo;
}

class ErrorBoundary extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(error: Error): State {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    this.setState({
      error,
      errorInfo
    });

    // Log error to monitoring service
    console.error('Error Boundary caught an error:', error, errorInfo);

    // In production, you would send this to your error monitoring service
    // Example: Sentry.captureException(error, { contexts: { react: errorInfo } });
  }

  handleRetry = () => {
    this.setState({ hasError: false, error: undefined, errorInfo: undefined });
  };

  handleReload = () => {
    window.location.reload();
  };

  render() {
    if (this.state.hasError) {
      if (this.props.fallback) {
        return this.props.fallback;
      }

      return (
        <div style={{
          minHeight: '100vh',
          backgroundColor: theme.colors.background.primary,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          padding: '20px'
        }}>
          <div style={{
            maxWidth: '600px',
            backgroundColor: theme.colors.background.card,
            borderRadius: '12px',
            border: `1px solid ${theme.colors.border.default}`,
            padding: '32px',
            textAlign: 'center',
            boxShadow: '0 10px 25px rgba(0,0,0,0.3)'
          }}>
            <div style={{
              fontSize: '48px',
              marginBottom: '16px',
              color: theme.colors.status.error
            }}>
              ⚠️
            </div>

            <h1 style={{
              fontSize: '24px',
              fontWeight: 700,
              color: theme.colors.text.primary,
              marginBottom: '8px'
            }}>
              Something went wrong
            </h1>

            <p style={{
              fontSize: '16px',
              color: theme.colors.text.secondary,
              marginBottom: '24px',
              lineHeight: '1.5'
            }}>
              We encountered an unexpected error. Our team has been notified and is working to fix this issue.
            </p>

            <div style={{
              backgroundColor: theme.colors.background.tertiary,
              borderRadius: '8px',
              padding: '16px',
              marginBottom: '24px',
              textAlign: 'left'
            }}>
              <div style={{
                fontSize: '14px',
                fontWeight: 600,
                color: theme.colors.text.secondary,
                marginBottom: '8px'
              }}>
                Error Details:
              </div>
              <div style={{
                fontSize: '13px',
                color: theme.colors.text.muted,
                fontFamily: 'monospace',
                wordBreak: 'break-word'
              }}>
                {this.state.error?.message || 'Unknown error'}
              </div>
            </div>

            <div style={{ display: 'flex', gap: '12px', justifyContent: 'center' }}>
              <button
                onClick={this.handleRetry}
                style={{
                  padding: '12px 24px',
                  borderRadius: '8px',
                  border: `1px solid ${theme.colors.primary.DEFAULT}`,
                  backgroundColor: 'transparent',
                  color: theme.colors.primary.DEFAULT,
                  fontSize: '14px',
                  fontWeight: 600,
                  cursor: 'pointer',
                  transition: 'all 0.2s'
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.backgroundColor = `${theme.colors.primary.DEFAULT}10`;
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.backgroundColor = 'transparent';
                }}
              >
                Try Again
              </button>

              <button
                onClick={this.handleReload}
                style={{
                  padding: '12px 24px',
                  borderRadius: '8px',
                  border: 'none',
                  backgroundColor: theme.colors.primary.DEFAULT,
                  color: '#fff',
                  fontSize: '14px',
                  fontWeight: 600,
                  cursor: 'pointer',
                  transition: 'all 0.2s'
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.opacity = '0.9';
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.opacity = '1';
                }}
              >
                Reload Page
              </button>
            </div>

            <div style={{
              marginTop: '24px',
              paddingTop: '24px',
              borderTop: `1px solid ${theme.colors.border.default}`,
              fontSize: '12px',
              color: theme.colors.text.muted
            }}>
              If this problem persists, please contact support with the error details above.
            </div>
          </div>
        </div>
      );
    }

    return this.props.children;
  }
}

export default ErrorBoundary;