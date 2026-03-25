'use client';

import React, { useState } from 'react';
import { theme } from '@/design-system/theme';

interface AddDataSourceModalProps {
    isOpen: boolean;
    onClose: () => void;
}

type SourceType = 'fs' | 'postgresql' | 'mysql' | 'mongodb' | 's3' | 'gcs' | 'slack' | 'azure_blob';

interface SourceOption {
    id: SourceType;
    label: string;
    description: string;
    icon: string;
}

const SOURCE_OPTIONS: SourceOption[] = [
    { id: 'fs', label: 'Local Filesystem', description: 'Scan files on the server local storage', icon: '📁' },
    { id: 's3', label: 'Amazon S3', description: 'Scan AWS S3 buckets for PII', icon: '☁️' },
    { id: 'postgresql', label: 'PostgreSQL', description: 'Scan PostgreSQL databases', icon: '🐘' },
    { id: 'mysql', label: 'MySQL', description: 'Scan MySQL/MariaDB databases', icon: '🐬' },
    { id: 'mongodb', label: 'MongoDB', description: 'Scan MongoDB collections', icon: '🍃' },
    { id: 'gcs', label: 'Google Cloud Storage', description: 'Scan GCS buckets', icon: '📦' },
    { id: 'azure_blob', label: 'Azure Blob Storage', description: 'Scan Azure containers', icon: '🟦' },
    { id: 'slack', label: 'Slack', description: 'Scan Slack messages', icon: '💬' },
];

export default function AddDataSourceModal({ isOpen, onClose }: AddDataSourceModalProps) {
    const [step, setStep] = useState(1);
    const [selectedType, setSelectedType] = useState<SourceType | null>(null);
    const [profileName, setProfileName] = useState('');
    const [config, setConfig] = useState<Record<string, string>>({});
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    if (!isOpen) return null;

    const handleNext = () => {
        if (step === 1 && selectedType) {
            setStep(2);
        } else if (step === 2) {
            handleSubmit();
        }
    };

    const handleSubmit = async () => {
        setLoading(true);
        setError(null);

        try {
            const payload = {
                source_type: selectedType,
                profile_name: profileName || `${selectedType}_${Date.now()}`,
                config: config
            };

            const res = await fetch('/api/v1/connections', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload)
            });

            if (!res.ok) {
                const data = await res.json();
                throw new Error(data.error || 'Failed to add connection');
            }

            setStep(3); // Success
            setTimeout(() => {
                onClose();
                setStep(1);
                setSelectedType(null);
                setProfileName('');
                setConfig({});
            }, 2000);
        } catch (err: any) {
            setError(err.message);
        } finally {
            setLoading(false);
        }
    };

    const renderConfigForm = () => {
        switch (selectedType) {
            case 'fs':
                return (
                    <div style={styles.formGroup}>
                        <label style={styles.label}>Directory Path</label>
                        <input
                            style={styles.input}
                            placeholder="/path/to/scan"
                            value={config.path || ''}
                            onChange={(e) => setConfig({ ...config, path: e.target.value })}
                        />
                    </div>
                );
            case 'postgresql':
            case 'mysql':
                return (
                    <>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>Host</label>
                            <input style={styles.input} placeholder="localhost" value={config.host || ''} onChange={(e) => setConfig({ ...config, host: e.target.value })} />
                        </div>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>Port</label>
                            <input style={styles.input} placeholder="5432" value={config.port || ''} onChange={(e) => setConfig({ ...config, port: e.target.value })} />
                        </div>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>Database</label>
                            <input style={styles.input} placeholder="mydb" value={config.database || ''} onChange={(e) => setConfig({ ...config, database: e.target.value })} />
                        </div>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>User</label>
                            <input style={styles.input} placeholder="user" value={config.user || ''} onChange={(e) => setConfig({ ...config, user: e.target.value })} />
                        </div>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>Password</label>
                            <input style={styles.input} type="password" placeholder="***" value={config.password || ''} onChange={(e) => setConfig({ ...config, password: e.target.value })} />
                        </div>
                    </>
                );
            case 's3':
                return (
                    <>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>Bucket Name</label>
                            <input style={styles.input} placeholder="my-bucket" value={config.bucket || ''} onChange={(e) => setConfig({ ...config, bucket: e.target.value })} />
                        </div>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>Region</label>
                            <input style={styles.input} placeholder="us-east-1" value={config.region || ''} onChange={(e) => setConfig({ ...config, region: e.target.value })} />
                        </div>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>AWS Access Key ID</label>
                            <input style={styles.input} placeholder="AKIA..." value={config.aws_access_key_id || ''} onChange={(e) => setConfig({ ...config, aws_access_key_id: e.target.value })} />
                        </div>
                        <div style={styles.formGroup}>
                            <label style={styles.label}>AWS Secret Access Key</label>
                            <input style={styles.input} type="password" placeholder="secret..." value={config.aws_secret_access_key || ''} onChange={(e) => setConfig({ ...config, aws_secret_access_key: e.target.value })} />
                        </div>
                    </>
                );
            default:
                return (
                    <div style={styles.formGroup}>
                        <div style={{ color: theme.colors.text.secondary }}>Configuration for {selectedType} is generic. Please check docs.</div>
                        <label style={styles.label}>JSON Config (Optional)</label>
                        <textarea
                            style={{ ...styles.input, height: '100px' }}
                            placeholder="{}"
                            onChange={(e) => {
                                try {
                                    setConfig(JSON.parse(e.target.value));
                                } catch (err) { }
                            }}
                        />
                    </div>
                );
        }
    };

    return (
        <div style={{
            position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
            backgroundColor: 'rgba(0,0,0,0.4)', zIndex: 1000,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            backdropFilter: 'blur(4px)',
        }}>
            <div style={{
                backgroundColor: theme.colors.background.card,
                width: '800px', maxWidth: '90%', maxHeight: '90vh',
                borderRadius: '12px', border: `1px solid ${theme.colors.border.default}`,
                display: 'flex', flexDirection: 'column', overflow: 'hidden'
            }}>
                {/* Header */}
                <div style={{
                    padding: '24px', borderBottom: `1px solid ${theme.colors.border.default}`,
                    display: 'flex', justifyContent: 'space-between', alignItems: 'center'
                }}>
                    <div>
                        <h2 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, marginBottom: '4px' }}>
                            Add Data Source
                        </h2>
                        <p style={{ fontSize: '13px', color: theme.colors.text.secondary }}>
                            Step {step} of 3: {step === 1 ? 'Select Source Type' : step === 2 ? 'Configure Connection' : 'Complete'}
                        </p>
                    </div>
                    <button onClick={onClose} style={{ background: 'none', border: 'none', fontSize: '20px', color: theme.colors.text.muted, cursor: 'pointer' }}>×</button>
                </div>

                {/* Content */}
                <div style={{ padding: '24px', overflowY: 'auto', flex: 1 }}>
                    {step === 1 && (
                        <>
                            <div style={{ marginBottom: '24px' }}>
                                <label style={styles.label}>Source Name (Friendly Name)</label>
                                <input
                                    style={styles.input}
                                    placeholder="e.g. Production Database, Customer Data Warehouse"
                                    value={profileName}
                                    onChange={(e) => setProfileName(e.target.value)}
                                />
                            </div>
                            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px' }}>
                                {SOURCE_OPTIONS.map(opt => (
                                    <div
                                        key={opt.id}
                                        onClick={() => setSelectedType(opt.id)}
                                        style={{
                                            padding: '16px',
                                            borderRadius: '8px',
                                            border: `1px solid ${selectedType === opt.id ? theme.colors.primary.DEFAULT : theme.colors.border.default}`,
                                            backgroundColor: selectedType === opt.id ? `${theme.colors.primary.DEFAULT}10` : 'transparent',
                                            cursor: 'pointer',
                                            transition: 'all 0.2s',
                                        }}
                                    >
                                        <div style={{ fontSize: '24px', marginBottom: '8px' }}>{opt.icon}</div>
                                        <div style={{ fontSize: '14px', fontWeight: 600, color: theme.colors.text.primary, marginBottom: '4px' }}>{opt.label}</div>
                                        <div style={{ fontSize: '12px', color: theme.colors.text.secondary }}>{opt.description}</div>
                                    </div>
                                ))}
                            </div>
                        </>
                    )}

                    {step === 2 && (
                        <div style={{ maxWidth: '600px', margin: '0 auto' }}>
                            {renderConfigForm()}
                            {error && <div style={{ marginTop: '16px', padding: '12px', backgroundColor: '#ef444420', color: '#ef4444', borderRadius: '6px', fontSize: '13px' }}>{error}</div>}
                        </div>
                    )}

                    {step === 3 && (
                        <div style={{ textAlign: 'center', padding: '40px' }}>
                            <div style={{ fontSize: '48px', marginBottom: '16px' }}>✅</div>
                            <h3 style={{ color: theme.colors.text.primary, marginBottom: '8px' }}>Data Source Added!</h3>
                            <p style={{ color: theme.colors.text.secondary }}>
                                Configuration has been saved to connection.yml. <br />
                                Review your new assets in the Asset Inventory or trigger a Scan All.
                            </p>
                        </div>
                    )}
                </div>

                {/* Footer */}
                {step < 3 && (
                    <div style={{
                        padding: '16px 24px', borderTop: `1px solid ${theme.colors.border.default}`,
                        display: 'flex', justifyContent: 'flex-end', gap: '12px'
                    }}>
                        <button onClick={onClose} style={styles.secondaryButton}>Cancel</button>
                        <button
                            onClick={handleNext}
                            disabled={step === 1 && !selectedType || loading}
                            style={{ ...styles.primaryButton, opacity: (step === 1 && !selectedType) ? 0.5 : 1 }}
                        >
                            {loading ? 'Saving...' : 'Continue'}
                        </button>
                    </div>
                )}
            </div>
        </div>
    );
}

const styles = {
    formGroup: {
        marginBottom: '16px',
    },
    label: {
        display: 'block',
        fontSize: '13px',
        fontWeight: 600,
        color: theme.colors.text.primary,
        marginBottom: '8px',
    },
    input: {
        width: '100%',
        padding: '10px 12px',
        fontSize: '14px',
        backgroundColor: theme.colors.background.tertiary,
        border: `1px solid ${theme.colors.border.default}`,
        borderRadius: '6px',
        color: theme.colors.text.primary,
        outline: 'none',
    },
    primaryButton: {
        padding: '8px 16px',
        backgroundColor: theme.colors.primary.DEFAULT,
        color: '#fff',
        border: 'none',
        borderRadius: '6px',
        fontWeight: 600,
        cursor: 'pointer',
    },
    secondaryButton: {
        padding: '8px 16px',
        backgroundColor: 'transparent',
        color: theme.colors.text.primary,
        border: `1px solid ${theme.colors.border.default}`,
        borderRadius: '6px',
        fontWeight: 600,
        cursor: 'pointer',
    }
};
