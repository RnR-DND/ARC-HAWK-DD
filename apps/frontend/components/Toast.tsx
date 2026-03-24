'use client';

import React, { useEffect } from 'react';
import { colors } from '@/design-system/colors';

export type ToastType = 'error' | 'success' | 'info';

interface ToastProps {
    message: string;
    type: ToastType;
    onClose: () => void;
}

export default function Toast({ message, type, onClose }: ToastProps) {
    useEffect(() => {
        const timer = setTimeout(onClose, 5000);
        return () => clearTimeout(timer);
    }, [onClose]);

    const getBackgroundColor = () => {
        switch (type) {
            case 'error':
                return colors.state.risk;
            case 'success':
                return colors.state.success;
            case 'info':
                return colors.state.info;
            default:
                return colors.background.card;
        }
    };

    return (
        <div
            style={{
                position: 'fixed',
                bottom: '24px',
                right: '24px',
                background: getBackgroundColor(),
                color: colors.text.primary,
                padding: '16px 24px',
                borderRadius: '8px',
                boxShadow: '0 8px 24px rgba(0, 0, 0, 0.4)',
                zIndex: 9999,
                maxWidth: '400px',
                animation: 'slideUp 0.3s ease-out',
                display: 'flex',
                alignItems: 'center',
                gap: '12px',
                fontSize: '14px',
                fontWeight: 500,
            }}
        >
            <span>{type === 'error' ? '❌' : type === 'success' ? '✅' : 'ℹ️'}</span>
            <span style={{ flex: 1 }}>{message}</span>
            <button
                onClick={onClose}
                style={{
                    background: 'transparent',
                    border: 'none',
                    color: colors.text.primary,
                    cursor: 'pointer',
                    fontSize: '18px',
                    padding: '0 4px',
                }}
            >
                ×
            </button>

            <style jsx>{`
        @keyframes slideUp {
          from {
            transform: translateY(100px);
            opacity: 0;
          }
          to {
            transform: translateY(0);
            opacity: 1;
          }
        }
      `}</style>
        </div>
    );
}
