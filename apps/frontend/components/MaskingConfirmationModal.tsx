'use client';

import React, { useState } from 'react';
import { maskingApi } from '@/services/masking.api';

interface MaskingConfirmationModalProps {
    assetId: string;
    assetName: string;
    findingsCount: number;
    onClose: () => void;
    onSuccess: () => void;
}

export default function MaskingConfirmationModal({
    assetId,
    assetName,
    findingsCount,
    onClose,
    onSuccess,
}: MaskingConfirmationModalProps) {
    const [strategy, setStrategy] = useState<'REDACT' | 'PARTIAL' | 'TOKENIZE'>('REDACT');
    const [understood, setUnderstood] = useState(false);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleConfirm = async () => {
        if (!understood) {
            setError('Please confirm that you understand this action cannot be undone');
            return;
        }

        setLoading(true);
        setError(null);

        try {
            await maskingApi.maskAsset({
                asset_id: assetId,
                strategy,
                masked_by: 'user', // Could be replaced with actual user info
            });

            onSuccess();
            onClose();
        } catch (err: any) {
            setError(err.message || 'Failed to mask asset');
        } finally {
            setLoading(false);
        }
    };

    const getStrategyDescription = (strat: string) => {
        switch (strat) {
            case 'REDACT':
                return 'Replace all PII with [REDACTED]';
            case 'PARTIAL':
                return 'Show partial data (e.g., XXXX-XXXX-1234 for Aadhaar)';
            case 'TOKENIZE':
                return 'Replace with consistent tokens (e.g., TOKEN_ABC123)';
            default:
                return '';
        }
    };

    const getStrategyExample = (strat: string) => {
        switch (strat) {
            case 'REDACT':
                return '1234-5678-9012 → [REDACTED]';
            case 'PARTIAL':
                return '1234-5678-9012 → XXXX-XXXX-9012';
            case 'TOKENIZE':
                return '1234-5678-9012 → TOKEN_A1B2C3D4E5F6';
            default:
                return '';
        }
    };

    return (
        <div className="modal-overlay" onClick={onClose}>
            <div className="modal-content" onClick={(e) => e.stopPropagation()}>
                <div className="modal-header">
                    <h2>⚠️ Mask Asset Data</h2>
                    <button onClick={onClose} className="close-btn">×</button>
                </div>

                <div className="modal-body">
                    <div className="warning-banner">
                        <strong>Warning:</strong> This action is irreversible. Original PII values cannot be recovered after masking.
                    </div>

                    <div className="asset-info">
                        <p><strong>Asset:</strong> {assetName}</p>
                        <p><strong>Findings to mask:</strong> {findingsCount}</p>
                    </div>

                    <div className="strategy-selector">
                        <label>Masking Strategy:</label>
                        <select
                            value={strategy}
                            onChange={(e) => setStrategy(e.target.value as any)}
                            disabled={loading}
                        >
                            <option value="REDACT">REDACT - Maximum Security</option>
                            <option value="PARTIAL">PARTIAL - Partial Visibility</option>
                            <option value="TOKENIZE">TOKENIZE - Consistent Tokens</option>
                        </select>

                        <div className="strategy-description">
                            <p>{getStrategyDescription(strategy)}</p>
                            <p className="example"><em>Example:</em> {getStrategyExample(strategy)}</p>
                        </div>
                    </div>

                    <div className="confirmation-checkbox">
                        <label>
                            <input
                                type="checkbox"
                                checked={understood}
                                onChange={(e) => setUnderstood(e.target.checked)}
                                disabled={loading}
                            />
                            <span>I understand this action cannot be undone</span>
                        </label>
                    </div>

                    {error && (
                        <div className="error-message">
                            {error}
                        </div>
                    )}
                </div>

                <div className="modal-footer">
                    <button onClick={onClose} disabled={loading} className="btn-secondary">
                        Cancel
                    </button>
                    <button
                        onClick={handleConfirm}
                        disabled={!understood || loading}
                        className="btn-danger"
                    >
                        {loading ? 'Masking...' : 'Confirm Masking'}
                    </button>
                </div>
            </div>

            <style jsx>{`
        .modal-overlay {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(15, 23, 42, 0.3);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 2000;
          backdrop-filter: blur(4px);
        }

        .modal-content {
          background: #ffffff;
          border-radius: 12px;
          width: 90%;
          max-width: 500px;
          box-shadow: 0 20px 40px rgba(0, 0, 0, 0.08);
          border: 1px solid #e2e8f0;
        }

        .modal-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 20px 24px;
          border-bottom: 1px solid #e2e8f0;
        }

        .modal-header h2 {
          margin: 0;
          font-size: 20px;
          color: var(--color-text-primary, #0f172a);
        }

        .close-btn {
          background: none;
          border: none;
          font-size: 28px;
          color: #94a3b8;
          cursor: pointer;
          padding: 0;
          width: 32px;
          height: 32px;
          display: flex;
          align-items: center;
          justify-content: center;
        }

        .close-btn:hover {
          color: #0f172a;
        }

        .modal-body {
          padding: 24px;
        }

        .warning-banner {
          background: #fef2f2;
          border: 1px solid #fecaca;
          border-radius: 8px;
          padding: 12px 16px;
          margin-bottom: 20px;
          color: #b91c1c;
        }

        .asset-info {
          margin-bottom: 20px;
          padding: 12px;
          background: #f1f5f9;
          border-radius: 6px;
          border: 1px solid #e2e8f0;
        }

        .asset-info p {
          margin: 6px 0;
          color: #475569;
        }

        .strategy-selector {
          margin-bottom: 20px;
        }

        .strategy-selector label {
          display: block;
          font-size: 14px;
          font-weight: 500;
          margin-bottom: 8px;
          color: #475569;
        }

        .strategy-selector select {
          width: 100%;
          padding: 10px 12px;
          background: #ffffff;
          border: 1px solid #cbd5e1;
          border-radius: 6px;
          color: #0f172a;
          font-size: 14px;
          cursor: pointer;
        }

        .strategy-selector select:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .strategy-description {
          margin-top: 12px;
          padding: 12px;
          background: #eff6ff;
          border: 1px solid #dbeafe;
          border-radius: 6px;
        }

        .strategy-description p {
          margin: 4px 0;
          color: #1e40af;
          font-size: 13px;
        }

        .strategy-description .example {
          font-family: monospace;
          color: #2563eb;
        }

        .confirmation-checkbox {
          margin-bottom: 16px;
        }

        .confirmation-checkbox label {
          display: flex;
          align-items: center;
          cursor: pointer;
        }

        .confirmation-checkbox input[type="checkbox"] {
          margin-right: 10px;
          width: 18px;
          height: 18px;
          cursor: pointer;
        }

        .confirmation-checkbox input[type="checkbox"]:disabled {
          cursor: not-allowed;
        }

        .confirmation-checkbox span {
          color: #0f172a;
          font-size: 14px;
        }

        .error-message {
          background: #fef2f2;
          border: 1px solid #fecaca;
          border-radius: 6px;
          padding: 10px 12px;
          color: #b91c1c;
          font-size: 13px;
          margin-top: 12px;
        }

        .modal-footer {
          display: flex;
          justify-content: flex-end;
          gap: 12px;
          padding: 16px 24px;
          border-top: 1px solid #e2e8f0;
        }

        .btn-secondary,
        .btn-danger {
          padding: 10px 20px;
          border-radius: 6px;
          font-size: 14px;
          font-weight: 500;
          cursor: pointer;
          transition: all 0.2s;
          border: none;
        }

        .btn-secondary {
          background: #f1f5f9;
          color: #475569;
        }

        .btn-secondary:hover:not(:disabled) {
          background: #e2e8f0;
        }

        .btn-danger {
          background: #dc2626;
          color: white;
        }

        .btn-danger:hover:not(:disabled) {
          background: #b91c1c;
        }

        .btn-secondary:disabled,
        .btn-danger:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }
      `}</style>
        </div>
    );
}
