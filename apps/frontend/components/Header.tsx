import React from 'react';

interface HeaderProps {
    scanTime?: string;
    environment?: string;
    riskScore?: number;
}

export default function Header({ scanTime, environment, riskScore }: HeaderProps) {
    return (
        <div className="header">
            <div className="header-content">
                <div>
                    <h1 className="header-title">ARC Platform - Data Lineage & PII Classification</h1>
                    {scanTime && (
                        <div className="header-meta">
                            <span>Last Scan: {new Date(scanTime).toLocaleString()}</span>
                            {environment && <span>Environment: {environment}</span>}
                        </div>
                    )}
                </div>
                {riskScore !== undefined && (
                    <div style={{ textAlign: 'right' }}>
                        <div className="card-label">Overall Risk Score</div>
                        <div className="card-value" style={{
                            fontSize: '24px',
                            color: riskScore >= 80 ? 'var(--risk-critical)' :
                                riskScore >= 60 ? 'var(--risk-high)' :
                                    riskScore >= 40 ? 'var(--risk-medium)' : 'var(--risk-low)'
                        }}>
                            {riskScore}
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}
