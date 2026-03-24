'use client';

import React, { useState, useEffect } from 'react';
import { maskingApi, MaskingStatusResponse } from '@/services/masking.api';
import MaskingConfirmationModal from './MaskingConfirmationModal';

interface MaskingButtonProps {
    assetId: string;
    assetName: string;
    findingsCount: number;
    onMaskingComplete?: () => void;
}

export default function MaskingButton({
    assetId,
    assetName,
    findingsCount,
    onMaskingComplete,
}: MaskingButtonProps) {
    const [showModal, setShowModal] = useState(false);
    const [maskingStatus, setMaskingStatus] = useState<MaskingStatusResponse | null>(null);
    const [loading, setLoading] = useState(true);
    const [showToast, setShowToast] = useState(false);

    useEffect(() => {
        fetchMaskingStatus();
    }, [assetId]);

    const fetchMaskingStatus = async () => {
        try {
            setLoading(true);
            const status = await maskingApi.getMaskingStatus(assetId);
            setMaskingStatus(status);
        } catch (err) {
            console.error('Failed to fetch masking status:', err);
        } finally {
            setLoading(false);
        }
    };

    const handleMaskingSuccess = () => {
        setShowToast(true);
        fetchMaskingStatus();
        if (onMaskingComplete) {
            onMaskingComplete();
        }

        // Hide toast after 3 seconds
        setTimeout(() => {
            setShowToast(false);
        }, 3000);
    };

    if (loading) {
        return (
            <button className="masking-btn loading" disabled>
                <span className="spinner"></span>
                Loading...
            </button>
        );
    }

    const isMasked = maskingStatus?.is_masked || false;

    return (
        <>
            <button
                className={`masking-btn ${isMasked ? 'masked' : 'unmask'}`}
                onClick={() => setShowModal(true)}
                disabled={isMasked}
                title={isMasked ? 'Asset is already masked' : 'Mask sensitive data in this asset'}
            >
                {isMasked ? (
                    <>
                        <span className="icon">🔒</span>
                        Masked ({maskingStatus?.masking_strategy})
                    </>
                ) : (
                    <>
                        <span className="icon">🔓</span>
                        Mask Asset
                    </>
                )}
            </button>

            {showModal && !isMasked && (
                <MaskingConfirmationModal
                    assetId={assetId}
                    assetName={assetName}
                    findingsCount={findingsCount}
                    onClose={() => setShowModal(false)}
                    onSuccess={handleMaskingSuccess}
                />
            )}

            {showToast && (
                <div className="toast success">
                    ✅ Asset masked successfully!
                </div>
            )}

            <style jsx>{`
        .masking-btn {
          display: inline-flex;
          align-items: center;
          gap: 8px;
          padding: 8px 16px;
          border-radius: 6px;
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s;
          border: none;
        }

        .masking-btn.unmask {
          background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
          color: white;
        }

        .masking-btn.unmask:hover:not(:disabled) {
          background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
          transform: translateY(-1px);
          box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
        }

        .masking-btn.masked {
          background: rgba(34, 197, 94, 0.1);
          color: #86efac;
          border: 1px solid rgba(34, 197, 94, 0.3);
          cursor: not-allowed;
        }

        .masking-btn.loading {
          background: rgba(255, 255, 255, 0.1);
          color: var(--color-text-muted, #94a3b8);
          cursor: wait;
        }

        .masking-btn:disabled {
          opacity: 0.7;
          cursor: not-allowed;
        }

        .masking-btn .icon {
          font-size: 16px;
        }

        .spinner {
          width: 14px;
          height: 14px;
          border: 2px solid rgba(255, 255, 255, 0.3);
          border-top-color: white;
          border-radius: 50%;
          animation: spin 0.6s linear infinite;
        }

        @keyframes spin {
          to {
            transform: rotate(360deg);
          }
        }

        .toast {
          position: fixed;
          top: 20px;
          right: 20px;
          padding: 12px 20px;
          border-radius: 8px;
          font-size: 14px;
          font-weight: 500;
          z-index: 3000;
          animation: slideIn 0.3s ease-out;
        }

        .toast.success {
          background: rgba(34, 197, 94, 0.9);
          color: white;
          box-shadow: 0 4px 12px rgba(34, 197, 94, 0.3);
        }

        @keyframes slideIn {
          from {
            transform: translateX(400px);
            opacity: 0;
          }
          to {
            transform: translateX(0);
            opacity: 1;
          }
        }
      `}</style>
        </>
    );
}
