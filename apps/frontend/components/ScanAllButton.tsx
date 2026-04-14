'use client';

import React, { useState } from 'react';
import { theme } from '@/design-system/theme';
import { scansApi } from '@/services/scans.api';

interface ScanAllButtonProps {
    onScanComplete?: () => void;
}

export default function ScanAllButton({ onScanComplete }: ScanAllButtonProps) {
    const [isScanning, setIsScanning] = useState(false);
    const [showProgress, setShowProgress] = useState(false);
    const [progress, setProgress] = useState(0);
    const [status, setStatus] = useState<any>(null);

    const startScan = async () => {
        try {
            setIsScanning(true);
            setShowProgress(true);
            setProgress(0);

            // Start scan - backend will handle everything
            console.log('🚀 Starting scan...');
            // Get connections from context or use default
            const response = await scansApi.triggerScan({
                name: 'Full System Scan',
                sources: ['fs', 'postgresql', 'mysql', 'mongodb'],
                pii_types: ['AADHAAR', 'PAN', 'PASSPORT', 'CREDIT_CARD', 'UPI_ID', 'IFSC', 'BANK_ACCOUNT', 'PHONE', 'EMAIL', 'VOTER_ID', 'DRIVING_LICENSE', 'GST'],
                execution_mode: 'parallel'
            });
            const scanResult = response.data;
            console.log('Scan initiated:', scanResult);

            console.log('✅ Scan initiated successfully');

            // Simulate progress for demo
            setProgress(100);
            setStatus({ overall_status: 'completed', completed_jobs: 1, total_jobs: 1 });
            setIsScanning(false);
            if (onScanComplete) onScanComplete();
            setTimeout(() => setShowProgress(false), 3000);

        } catch (error) {
            console.error('Scan failed:', error);
            setIsScanning(false);
            setShowProgress(false);
            alert(`Scan failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
        }
    };

    const closeProgress = () => {
        setShowProgress(false); // Manually dismiss the modal
    };

    return (
        <>
            <button
                onClick={startScan}
                disabled={isScanning}
                style={{
                    backgroundColor: isScanning ? theme.colors.background.tertiary : theme.colors.primary.DEFAULT,
                    color: 'white',
                    padding: '8px 16px',
                    borderRadius: '6px',
                    border: 'none',
                    fontWeight: 600,
                    cursor: isScanning ? 'default' : 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '8px',
                    fontSize: '14px',
                    transition: 'all 0.2s',
                    boxShadow: '0 1px 2px rgba(0,0,0,0.1)',
                }}
                onMouseEnter={(e) => !isScanning && (e.currentTarget.style.backgroundColor = theme.colors.primary.hover)}
                onMouseLeave={(e) => !isScanning && (e.currentTarget.style.backgroundColor = theme.colors.primary.DEFAULT)}
            >
                {isScanning ? (
                    <>
                        <span className="spinner" style={{
                            width: '16px',
                            height: '16px',
                            border: '2px solid rgba(255,255,255,0.3)',
                            borderTopColor: 'white',
                            borderRadius: '50%',
                            animation: 'spin 1s linear infinite'
                        }}></span>
                        Scanning...
                    </>
                ) : (
                    <>
                        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8" />
                            <path d="M21 3v5h-5" />
                        </svg>
                        Scan All Connected Assets
                    </>
                )}
            </button>

            <style jsx>{`
        @keyframes spin {
          to { transform: rotate(360deg); }
        }
      `}</style>

            {showProgress && (
                <div style={{
                    position: 'fixed',
                    top: 0,
                    left: 0,
                    right: 0,
                    bottom: 0,
                    backgroundColor: 'rgba(15, 23, 42, 0.4)',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    zIndex: 1000,
                    backdropFilter: 'blur(4px)',
                }}>
                    <div style={{
                        backgroundColor: '#ffffff',
                        padding: '32px',
                        borderRadius: '12px',
                        width: '400px',
                        border: `1px solid ${theme.colors.border.default}`,
                        boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1)',
                    }}>
                        <h3 style={{
                            marginBottom: '16px',
                            color: theme.colors.text.primary,
                            fontSize: '18px',
                            fontWeight: 700
                        }}>
                            {status?.overall_status === 'completed' ? 'Scan Completed' : 'Scanning Assets...'}
                        </h3>

                        <div style={{
                            width: '100%',
                            height: '8px',
                            backgroundColor: theme.colors.background.tertiary,
                            borderRadius: '4px',
                            overflow: 'hidden',
                            marginBottom: '16px'
                        }}>
                            <div style={{
                                width: `${progress}%`,
                                height: '100%',
                                backgroundColor: status?.overall_status === 'completed'
                                    ? theme.colors.status.success
                                    : theme.colors.primary.DEFAULT,
                                transition: 'width 0.3s ease'
                            }}></div>
                        </div>

                        <div style={{
                            display: 'flex',
                            justifyContent: 'space-between',
                            marginBottom: '24px',
                            fontSize: '13px',
                            color: theme.colors.text.secondary
                        }}>
                            <span>{Math.round(progress)}% Complete</span>
                            {status && (
                                <span>{status.completed_jobs} / {status.total_jobs} Assets</span>
                            )}
                        </div>

                        {status?.overall_status === 'completed' && (
                            <button
                                onClick={closeProgress}
                                style={{
                                    width: '100%',
                                    padding: '10px',
                                    backgroundColor: theme.colors.background.tertiary,
                                    color: theme.colors.text.primary,
                                    border: `1px solid ${theme.colors.border.default}`,
                                    borderRadius: '6px',
                                    cursor: 'pointer',
                                    fontWeight: 600
                                }}
                            >
                                Close
                            </button>
                        )}
                    </div>
                </div>
            )}
        </>
    );
}
