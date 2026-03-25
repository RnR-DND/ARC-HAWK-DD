import React from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';

interface LoadingStateProps {
    message?: string;
    fullScreen?: boolean;
}

export default function LoadingState({
    message = 'Loading...',
    fullScreen = false,
}: LoadingStateProps) {
    const containerStyle: React.CSSProperties = fullScreen
        ? {
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: 'rgba(255, 255, 255, 0.8)',
            zIndex: theme.zIndex.modal,
            backdropFilter: 'blur(8px)',
        }
        : {
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '64px 32px',
            minHeight: '300px',
        };

    return (
        <div style={containerStyle}>
            {/* Spinner */}
            <div
                style={{
                    width: '48px',
                    height: '48px',
                    border: `4px solid ${fullScreen ? colors.neutral[100] : colors.neutral[200]}`,
                    borderTop: `4px solid ${colors.blue[600]}`,
                    borderRadius: '50%',
                    animation: 'spin 1s linear infinite',
                    marginBottom: '16px',
                }}
            />

            {/* Message */}
            <p
                style={{
                    fontSize: theme.fontSize.base,
                    fontWeight: theme.fontWeight.medium,
                    color: fullScreen ? colors.text.secondary : colors.text.muted,
                }}
            >
                {message}
            </p>

            {/* CSS for spin animation */}
            <style jsx>{`
        @keyframes spin {
          0% {
            transform: rotate(0deg);
          }
          100% {
            transform: rotate(360deg);
          }
        }
      `}</style>
        </div>
    );
}
