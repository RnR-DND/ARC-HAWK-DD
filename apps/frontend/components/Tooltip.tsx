'use client';

import React, { useState } from 'react';
import { theme } from '@/design-system/theme';

interface TooltipProps {
    content: string | React.ReactNode;
    children: React.ReactNode;
    placement?: 'top' | 'bottom';
}

export default function Tooltip({ content, children, placement = 'top' }: TooltipProps) {
    const [isVisible, setIsVisible] = useState(false);

    return (
        <div
            style={{ position: 'relative', display: 'inline-block' }}
            onMouseEnter={() => setIsVisible(true)}
            onMouseLeave={() => setIsVisible(false)}
        >
            {children}
            {isVisible && (
                <div style={{
                    position: 'absolute',
                    bottom: placement === 'top' ? '100%' : 'auto',
                    top: placement === 'bottom' ? '100%' : 'auto',
                    left: '50%',
                    transform: 'translateX(-50%)',
                    marginBottom: placement === 'top' ? '8px' : 0,
                    marginTop: placement === 'bottom' ? '8px' : 0,
                    backgroundColor: theme.colors.background.tertiary,
                    color: theme.colors.text.primary,
                    padding: '8px 12px',
                    borderRadius: '6px',
                    fontSize: '12px',
                    fontWeight: 500,
                    lineHeight: '1.4',
                    whiteSpace: 'nowrap',
                    boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.5)',
                    border: `1px solid ${theme.colors.border.active}`,
                    zIndex: 50,
                    minWidth: '200px',
                    textAlign: 'center',
                }}>
                    {content}
                    {/* Arrow */}
                    <div style={{
                        position: 'absolute',
                        top: placement === 'top' ? '100%' : 'auto',
                        bottom: placement === 'bottom' ? '100%' : 'auto',
                        left: '50%',
                        marginLeft: '-4px',
                        borderWidth: '4px',
                        borderStyle: 'solid',
                        borderColor: placement === 'top'
                            ? `${theme.colors.border.active} transparent transparent transparent`
                            : `transparent transparent ${theme.colors.border.active} transparent`,
                    }} />
                </div>
            )}
        </div>
    );
}

export function InfoIcon({ size = 14, color }: { size?: number, color?: string }) {
    const iconColor = color || theme.colors.text.muted;
    return (
        <svg
            width={size}
            height={size}
            viewBox="0 0 24 24"
            fill="none"
            stroke={iconColor}
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
            style={{ cursor: 'help' }}
        >
            <circle cx="12" cy="12" r="10"></circle>
            <line x1="12" y1="16" x2="12" y2="12"></line>
            <line x1="12" y1="8" x2="12.01" y2="8"></line>
        </svg>
    );
}
