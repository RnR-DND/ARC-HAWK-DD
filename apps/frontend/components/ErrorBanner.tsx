'use client';

import React, { useEffect, useState } from 'react';
import { colors } from '@/design-system/colors';

export type ErrorSeverity = 'error' | 'warning' | 'info';

interface ErrorBannerProps {
    message: string;
    severity?: ErrorSeverity;
    onRetry?: () => void;
    onDismiss?: () => void;
    autoDismiss?: boolean;
    dismissAfterMs?: number;
}

export default function ErrorBanner({
    message,
    severity = 'error',
    onRetry,
    onDismiss,
    autoDismiss = true,
    dismissAfterMs = 10000,
}: ErrorBannerProps) {
    const [visible, setVisible] = useState(true);

    useEffect(() => {
        if (autoDismiss) {
            const timer = setTimeout(() => {
                setVisible(false);
                onDismiss?.();
            }, dismissAfterMs);

            return () => clearTimeout(timer);
        }
    }, [autoDismiss, dismissAfterMs, onDismiss]);

    if (!visible) return null;

    const getSeverityStyles = () => {
        switch (severity) {
            case 'error':
                return {
                    bg: '#FEF2F2',
                    border: '#FECACA',
                    text: '#B91C1C',
                    icon: '⚠️',
                };
            case 'warning':
                return {
                    bg: '#FEF9C3',
                    border: '#FDE047',
                    text: '#854D0E',
                    icon: '⚡',
                };
            case 'info':
                return {
                    bg: '#DBEAFE',
                    border: '#93C5FD',
                    text: '#1E40AF',
                    icon: 'ℹ️',
                };
        }
    };

    const styles = getSeverityStyles();

    const handleDismiss = () => {
        setVisible(false);
        onDismiss?.();
    };

    return (
        <div
            style={{
                padding: '16px 24px',
                backgroundColor: styles.bg,
                border: `1px solid ${styles.border}`,
                borderRadius: '12px',
                color: styles.text,
                marginBottom: '24px',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
                animation: 'slideDown 0.3s ease-out',
            }}
        >
            <div style={{ display: 'flex', alignItems: 'center', gap: '12px', flex: 1 }}>
                <span style={{ fontSize: '20px' }}>{styles.icon}</span>
                <span style={{ fontWeight: 600, fontSize: '14px' }}>{message}</span>
            </div>

            <div style={{ display: 'flex', gap: '8px' }}>
                {onRetry && (
                    <button
                        onClick={onRetry}
                        style={{
                            padding: '6px 16px',
                            borderRadius: '6px',
                            border: `1px solid ${styles.text}`,
                            backgroundColor: 'transparent',
                            color: styles.text,
                            fontSize: '13px',
                            fontWeight: 700,
                            cursor: 'pointer',
                            transition: 'all 0.2s',
                        }}
                        onMouseEnter={(e) => {
                            e.currentTarget.style.backgroundColor = styles.text;
                            e.currentTarget.style.color = '#FFFFFF';
                        }}
                        onMouseLeave={(e) => {
                            e.currentTarget.style.backgroundColor = 'transparent';
                            e.currentTarget.style.color = styles.text;
                        }}
                    >
                        Retry
                    </button>
                )}
                <button
                    onClick={handleDismiss}
                    style={{
                        padding: '6px 12px',
                        borderRadius: '6px',
                        border: 'none',
                        backgroundColor: 'transparent',
                        color: styles.text,
                        fontSize: '18px',
                        fontWeight: 700,
                        cursor: 'pointer',
                        opacity: 0.7,
                        transition: 'opacity 0.2s',
                    }}
                    onMouseEnter={(e) => (e.currentTarget.style.opacity = '1')}
                    onMouseLeave={(e) => (e.currentTarget.style.opacity = '0.7')}
                    title="Dismiss"
                >
                    ×
                </button>
            </div>

            <style jsx global>{`
                @keyframes slideDown {
                    from {
                        opacity: 0;
                        transform: translateY(-10px);
                    }
                    to {
                        opacity: 1;
                        transform: translateY(0);
                    }
                }
            `}</style>
        </div>
    );
}
