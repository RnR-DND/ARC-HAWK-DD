import React from 'react';
import { colors } from '@/design-system/colors';
import { theme } from '@/design-system/themes';

interface EmptyStateProps {
    icon?: string;
    title: string;
    description?: string;
    action?: {
        label: string;
        onClick: () => void;
    };
}

export default function EmptyState({
    icon = '📭',
    title,
    description,
    action,
}: EmptyStateProps) {
    return (
        <div
            style={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                justifyContent: 'center',
                padding: '64px 32px',
                minHeight: '400px',
                background: `radial-gradient(circle at center, ${colors.background.surface} 0%, ${colors.background.primary} 100%)`,
                borderRadius: '16px',
                border: `1px dashed ${colors.border.strong}`,
            }}
        >
            {/* Icon */}
            <div
                style={{
                    fontSize: '64px',
                    marginBottom: '24px',
                    opacity: 0.8,
                    filter: 'grayscale(0.2)',
                }}
            >
                {icon}
            </div>

            {/* Title */}
            <h3
                style={{
                    fontSize: theme.fontSize.lg,
                    fontWeight: 800,
                    color: colors.text.primary,
                    marginBottom: '12px',
                    letterSpacing: '-0.02em',
                }}
            >
                {title}
            </h3>

            {/* Description */}
            {description && (
                <p
                    style={{
                        fontSize: '15px',
                        color: colors.text.secondary,
                        textAlign: 'center',
                        maxWidth: '440px',
                        lineHeight: 1.6,
                        marginBottom: action ? '32px' : '0',
                    }}
                >
                    {description}
                </p>
            )}

            {/* Action Button */}
            {action && (
                <button
                    onClick={action.onClick}
                    style={{
                        padding: '12px 28px',
                        backgroundColor: colors.nodeColors.system,
                        color: '#ffffff',
                        border: 'none',
                        borderRadius: '12px',
                        fontSize: '14px',
                        fontWeight: 700,
                        cursor: 'pointer',
                        transition: 'all 0.2s ease',
                        boxShadow: '0 4px 6px -1px rgba(79, 70, 229, 0.2)',
                    }}
                    onMouseEnter={(e) => {
                        e.currentTarget.style.transform = 'translateY(-2px)';
                        e.currentTarget.style.boxShadow = '0 10px 15px -3px rgba(79, 70, 229, 0.3)';
                    }}
                    onMouseLeave={(e) => {
                        e.currentTarget.style.transform = 'translateY(0)';
                        e.currentTarget.style.boxShadow = '0 4px 6px -1px rgba(79, 70, 229, 0.2)';
                    }}
                >
                    {action.label}
                </button>
            )}
        </div>
    );
}
