'use client';

import React, { useEffect, useState } from 'react';
import { theme, getRiskColor } from '@/design-system/theme';
import Tooltip, { InfoIcon } from '@/components/Tooltip';
import { complianceApi, type RetentionViolation, type DPDPAGapReport, type SectionSummary } from '@/services/compliance.api';
import { get } from '@/utils/api-client';
import type { ComplianceOverview } from '@/types/api';

interface ConsentRecord {
    id: string;
    asset_id: string;
    data_subject_id: string;
    consent_type: string;
    purpose: string;
    status: string;
}

interface DPRItem {
    id: string;
    request_type: string;
    status: string;
    data_principal_id: string;
    created_at: string;
    due_date?: string;
}

interface DPRStats {
    total?: number;
    pending?: number;
    in_progress?: number;
    completed?: number;
}

interface HealthComponent {
    name: string;
    status: string;
    message?: string;
}

interface HealthData {
    components: HealthComponent[];
    timestamp: string;
    status: string;
}

export default function CompliancePage() {
    const [data, setData] = useState<ComplianceOverview | null>(null);
    const [loading, setLoading] = useState(true);
    const [refreshing, setRefreshing] = useState(false);

    useEffect(() => {
        fetchComplianceData();

        // Poll compliance metrics every 60 seconds to catch updates from completed scans
        const interval = setInterval(() => {
            fetchComplianceData(/* silent */ true);
        }, 60000);
        return () => clearInterval(interval);
    }, []);

    const fetchComplianceData = async (silent = false) => {
        try {
            if (!silent) setLoading(true);
            else setRefreshing(true);
            const jsonData = await complianceApi.getOverview();
            setData(jsonData);
        } catch (error) {
            console.error('Failed to fetch compliance data', error);
        } finally {
            setLoading(false);
            setRefreshing(false);
        }
    };

    const handleManualRefresh = () => fetchComplianceData(true);

    if (loading) return <div style={{ padding: '32px', color: theme.colors.text.primary }}>Loading compliance posture...</div>;
    if (!data) return <div style={{ padding: '32px', color: theme.colors.text.primary }}>Failed to load compliance data.</div>;

    return (
        <div style={{ minHeight: '100vh', backgroundColor: theme.colors.background.primary }}>
            <div className="container max-w-screen-2xl mx-auto" style={{ padding: '32px' }}>
                {/* Header */}
                <div style={{ marginBottom: '32px', display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <div>
                        <h2 style={{ fontSize: '24px', fontWeight: 700, color: theme.colors.text.primary, marginBottom: '8px' }}>
                            DPDPA Compliance Posture
                        </h2>
                        <p style={{ color: theme.colors.text.secondary }}>
                            Real-time monitoring of Digital Personal Data Protection Act compliance
                        </p>
                    </div>
                    <button
                        onClick={handleManualRefresh}
                        disabled={refreshing}
                        style={{
                            padding: '8px 16px',
                            borderRadius: '8px',
                            border: `1px solid ${theme.colors.border.default}`,
                            backgroundColor: theme.colors.background.card,
                            color: theme.colors.text.secondary,
                            fontSize: '13px',
                            fontWeight: 600,
                            cursor: refreshing ? 'not-allowed' : 'pointer',
                            opacity: refreshing ? 0.6 : 1,
                            display: 'flex',
                            alignItems: 'center',
                            gap: '6px',
                        }}
                        title="Refresh compliance metrics (auto-refreshes every 60s)"
                    >
                        {refreshing ? 'Refreshing...' : 'Refresh'}
                    </button>
                </div>

                {/* KPI Cards */}
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                    <KPICard
                        title="Compliance Score"
                        value={`${Math.round(data.compliance_score)}%`}
                        subtitle={`${data.compliant_assets} / ${data.total_assets} Assets Compliant`}
                        trend={data.compliance_score > 90 ? 'positive' : 'negative'}
                        tooltip="Percentage of assets meeting all DPDPA requirements."
                    />
                    <KPICard
                        title="Critical Exposure"
                        value={data.critical_exposure.total_assets}
                        subtitle="Assets with Critical PII"
                        color={theme.colors.risk.critical}
                        tooltip="Assets containing Sensitive Personal Data (e.g., Finance, Health, Auth) that are publicly accessible or unencrypted."
                    />
                    <KPICard
                        title="Consent Violations"
                        value={data.consent_violations.missing_consent}
                        subtitle="Assets Missing Consent"
                        color={theme.colors.risk.high}
                        tooltip="Personal data assets that do not have a mapped Consent ID managed in the Consent Ledger."
                    />
                    <KPICard
                        title="Remediation Tasks"
                        value={data.remediation_queue.length}
                        subtitle="Pending Actions"
                        color={theme.colors.risk.medium}
                        tooltip="Tasks generated by policy violations or critical findings that require immediate attention."
                    />
                </div>

                {/* DPDPA Obligation Checklist */}
                <DPDPAObligationChecklist />

                {/* Retention Policy Violations */}
                <RetentionSection />

                {/* Consent Records */}
                <ConsentSection />

                {/* Data Principal Rights */}
                <DPRSection />

                {/* GRO Settings */}
                <GROSettingsSection />

                {/* Main Content Grid */}
                <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '24px' }}>
                    {/* Remediation Queue */}
                    <div style={{
                        backgroundColor: theme.colors.background.card,
                        borderRadius: '12px',
                        border: `1px solid ${theme.colors.border.default}`,
                        overflow: 'hidden'
                    }}>
                        <div style={{ padding: '24px', borderBottom: `1px solid ${theme.colors.border.default}` }}>
                            <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, margin: 0 }}>Priority Remediation Queue</h3>
                            <p style={{ fontSize: '13px', color: theme.colors.text.secondary, marginTop: '6px', lineHeight: 1.4 }}>
                                Assets showing highest risk exposure based on PII sensitivity and access controls.
                            </p>
                        </div>

                        <div className="overflow-x-auto">
                        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                            <thead style={{ backgroundColor: theme.colors.background.tertiary }}>
                                <tr>
                                    <th style={{ padding: '16px 24px', textAlign: 'left', fontSize: '12px', color: theme.colors.text.secondary, textTransform: 'uppercase' }}>Asset</th>
                                    <th style={{ padding: '16px 24px', textAlign: 'left', fontSize: '12px', color: theme.colors.text.secondary, textTransform: 'uppercase' }}>Msg</th>
                                    <th style={{ padding: '16px 24px', textAlign: 'left', fontSize: '12px', color: theme.colors.text.secondary, textTransform: 'uppercase' }}>Risk</th>
                                    <th style={{ padding: '16px 24px', textAlign: 'right', fontSize: '12px', color: theme.colors.text.secondary, textTransform: 'uppercase' }}>Action</th>
                                </tr>
                            </thead>
                            <tbody>
                                {data.remediation_queue.length > 0 ? (
                                    data.remediation_queue.map((item) => (
                                        <tr key={item.asset_id} style={{ borderBottom: `1px solid ${theme.colors.border.default}` }}>
                                            <td style={{ padding: '16px 24px' }}>
                                                <div style={{ fontWeight: 600, color: theme.colors.text.primary }}>{item.asset_name}</div>
                                                <div style={{ fontSize: '12px', color: theme.colors.text.muted, fontFamily: 'monospace' }}>{item.asset_path}</div>
                                            </td>
                                            <td style={{ padding: '16px 24px' }}>
                                                <div style={{ display: 'flex', gap: '6px', flexWrap: 'wrap' }}>
                                                    {item.pii_types.slice(0, 3).map(type => (
                                                        <span key={type} style={{
                                                            fontSize: '11px',
                                                            padding: '2px 8px',
                                                            borderRadius: '4px',
                                                            backgroundColor: theme.colors.background.tertiary,
                                                            color: theme.colors.text.secondary
                                                        }}>
                                                            {type}
                                                        </span>
                                                    ))}
                                                    {item.pii_types.length > 3 && (
                                                        <span style={{ fontSize: '11px', color: theme.colors.text.muted }}>+{item.pii_types.length - 3}</span>
                                                    )}
                                                </div>
                                            </td>
                                            <td style={{ padding: '16px 24px' }}>
                                                <RiskBadge level={item.risk_level} />
                                            </td>
                                            <td style={{ padding: '16px 24px', textAlign: 'right' }}>
                                                <button
                                                    onClick={() => window.location.href = '/findings'}
                                                    style={{
                                                        padding: '6px 12px',
                                                        fontSize: '13px',
                                                        fontWeight: 600,
                                                        color: theme.colors.primary.DEFAULT,
                                                        backgroundColor: 'transparent',
                                                        border: `1px solid ${theme.colors.primary.DEFAULT}`,
                                                        borderRadius: '6px',
                                                        cursor: 'pointer'
                                                    }}>
                                                    Investigate
                                                </button>
                                            </td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr>
                                        <td colSpan={4} style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>
                                            No pending remediation tasks. Excellent!
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                        </div>{/* overflow-x-auto */}
                    </div>

                    {/* Breakdown View */}
                    <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
                        <div style={{
                            backgroundColor: theme.colors.background.card,
                            borderRadius: '12px',
                            border: `1px solid ${theme.colors.border.default}`,
                            padding: '24px'
                        }}>
                            <h3 style={{ fontSize: '16px', fontWeight: 600, color: theme.colors.text.primary, marginBottom: '16px' }}>
                                DPDPA Categories
                            </h3>
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                                {(() => {
                                    const barMax = Math.max(data.critical_exposure.total_findings, data.remediation_queue.length, data.total_assets, 1);
                                    return (<>
                                        <CategoryBar label="Sensitive Personal Data" count={data.critical_exposure.total_findings} color={theme.colors.risk.critical} maxCount={barMax} />
                                        <CategoryBar label="Personal Data" count={data.remediation_queue.length} color={theme.colors.risk.high} maxCount={barMax} />
                                        <CategoryBar label="Non-Personal Data" count={data.total_assets} color={theme.colors.risk.low} maxCount={barMax} />
                                    </>);
                                })()}
                            </div>
                        </div>

                        <div style={{
                            backgroundColor: theme.colors.background.card,
                            borderRadius: '12px',
                            border: `1px solid ${theme.colors.border.default}`,
                            padding: '24px',
                            flex: 1
                        }}>
                            <h3 style={{ fontSize: '16px', fontWeight: 600, color: theme.colors.text.primary, marginBottom: '12px' }}>
                                System Health
                            </h3>
                            <SystemHealthStatus />
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}

function KPICard({ title, value, subtitle, color, trend, tooltip }: {
    title: string;
    value: React.ReactNode;
    subtitle?: React.ReactNode;
    color?: string;
    trend?: string | number;
    tooltip?: React.ReactNode;
}) {
    return (
        <div style={{
            backgroundColor: theme.colors.background.card,
            borderRadius: '12px',
            border: `1px solid ${theme.colors.border.default}`,
            padding: '24px',
        }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                <span style={{ fontSize: '14px', color: theme.colors.text.secondary, fontWeight: 600 }}>{title}</span>
                {tooltip && <Tooltip content={tooltip}><InfoIcon size={14} /></Tooltip>}
            </div>
            <div style={{ fontSize: '32px', color: color || theme.colors.text.primary, fontWeight: 800, marginBottom: '4px' }}>{value}</div>
            <div style={{ fontSize: '13px', color: theme.colors.text.muted }}>{subtitle}</div>
        </div>
    );
}

function RiskBadge({ level }: { level: string }) {
    const color = getRiskColor(level);
    return (
        <span style={{
            display: 'inline-block',
            padding: '4px 12px',
            borderRadius: '999px',
            fontSize: '12px',
            fontWeight: 700,
            backgroundColor: `${color}20`,
            color: color,
            border: `1px solid ${color}40`,
            textTransform: 'uppercase'
        }}>
            {level}
        </span>
    );
}

function CategoryBar({ label, count, color, maxCount }: { label: string; count: string | number; color: string; maxCount: number }) {
    const pct = maxCount > 0 ? Math.round((Number(count) / maxCount) * 100) : 0;
    return (
        <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '13px', marginBottom: '6px' }}>
                <span style={{ color: theme.colors.text.secondary }}>{label}</span>
                <span style={{ color: theme.colors.text.primary, fontWeight: 600 }}>{count}</span>
            </div>
            <div style={{ height: '6px', backgroundColor: theme.colors.background.tertiary, borderRadius: '3px', overflow: 'hidden' }}>
                <div style={{ height: '100%', width: `${pct}%`, backgroundColor: color, borderRadius: '3px' }} />
            </div>
        </div>
    );
}

function RetentionSection() {
    const [violations, setViolations] = useState<RetentionViolation[]>([]);
    const [loading, setLoading] = useState(true);
    const [showSetPolicy, setShowSetPolicy] = useState<string | null>(null);
    const [policyForm, setPolicyForm] = useState({ policy_days: 365, policy_name: 'DPDPA Default', policy_basis: 'DPDPA Section 8(7)' });
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        complianceApi.getRetentionViolations().then(v => {
            setViolations(v);
            setLoading(false);
        });
    }, []);

    const handleSetPolicy = async (assetId: string) => {
        try {
            setSaving(true);
            await complianceApi.setRetentionPolicy(assetId, policyForm);
            setShowSetPolicy(null);
            const updated = await complianceApi.getRetentionViolations();
            setViolations(updated);
        } catch (e) {
            console.error('Failed to set retention policy', e);
        } finally {
            setSaving(false);
        }
    };

    return (
        <div style={{
            backgroundColor: theme.colors.background.card,
            borderRadius: '12px',
            border: `1px solid ${theme.colors.border.default}`,
            overflow: 'hidden',
            marginBottom: '24px'
        }}>
            <div style={{ padding: '24px', borderBottom: `1px solid ${theme.colors.border.default}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                    <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, margin: 0 }}>Data Retention Policies</h3>
                    <p style={{ fontSize: '13px', color: theme.colors.text.secondary, marginTop: '6px' }}>
                        Findings that have exceeded their retention period and require deletion per DPDPA.
                    </p>
                </div>
                <span style={{ fontSize: '13px', color: violations.length > 0 ? theme.colors.risk.critical : theme.colors.risk.low, fontWeight: 700, backgroundColor: violations.length > 0 ? `${theme.colors.risk.critical}15` : `${theme.colors.risk.low}15`, padding: '4px 12px', borderRadius: '999px' }}>
                    {loading ? '…' : `${violations.length} Violations`}
                </span>
            </div>

            {loading ? (
                <div style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>Loading retention data…</div>
            ) : violations.length === 0 ? (
                <div style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>
                    ✅ No retention violations. All data is within policy limits.
                </div>
            ) : (
                <div className="overflow-x-auto">
                <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead style={{ backgroundColor: theme.colors.background.tertiary }}>
                        <tr>
                            {['Asset', 'PII Type', 'Detected', 'Policy (days)', 'Due Date', 'Overdue', 'Action'].map(h => (
                                <th key={h} style={{ padding: '12px 20px', textAlign: 'left', fontSize: '11px', color: theme.colors.text.secondary, textTransform: 'uppercase' }}>{h}</th>
                            ))}
                        </tr>
                    </thead>
                    <tbody>
                        {violations.map(v => (
                            <React.Fragment key={v.finding_id}>
                                <tr style={{ borderBottom: `1px solid ${theme.colors.border.default}` }}>
                                    <td style={{ padding: '14px 20px', fontWeight: 600, color: theme.colors.text.primary }}>{v.asset_name}</td>
                                    <td style={{ padding: '14px 20px' }}>
                                        <span style={{ fontSize: '12px', padding: '2px 8px', borderRadius: '4px', backgroundColor: theme.colors.background.tertiary, color: theme.colors.text.secondary }}>{v.pii_type}</span>
                                    </td>
                                    <td style={{ padding: '14px 20px', fontSize: '13px', color: theme.colors.text.muted }}>{new Date(v.first_detected_at).toLocaleDateString()}</td>
                                    <td style={{ padding: '14px 20px', fontSize: '13px', color: theme.colors.text.secondary }}>{v.retention_policy_days}d</td>
                                    <td style={{ padding: '14px 20px', fontSize: '13px', color: theme.colors.risk.high }}>{new Date(v.deletion_due_at).toLocaleDateString()}</td>
                                    <td style={{ padding: '14px 20px' }}>
                                        <span style={{ fontSize: '12px', fontWeight: 700, color: theme.colors.risk.critical }}>{v.days_overdue} days</span>
                                    </td>
                                    <td style={{ padding: '14px 20px' }}>
                                        <div style={{ display: 'flex', gap: '8px' }}>
                                            <button onClick={() => setShowSetPolicy(showSetPolicy === v.asset_id ? null : v.asset_id)}
                                                style={{ padding: '4px 10px', fontSize: '12px', fontWeight: 600, color: theme.colors.primary.DEFAULT, backgroundColor: 'transparent', border: `1px solid ${theme.colors.primary.DEFAULT}`, borderRadius: '6px', cursor: 'pointer' }}>
                                                Set Policy
                                            </button>
                                            <a href="/remediation" style={{ padding: '4px 10px', fontSize: '12px', fontWeight: 600, color: 'white', backgroundColor: theme.colors.risk.critical, border: 'none', borderRadius: '6px', textDecoration: 'none', display: 'inline-block' }}>
                                                Remediate
                                            </a>
                                        </div>
                                    </td>
                                </tr>
                                {showSetPolicy === v.asset_id && (
                                    <tr style={{ backgroundColor: `${theme.colors.primary.DEFAULT}08` }}>
                                        <td colSpan={7} style={{ padding: '16px 20px' }}>
                                            <div style={{ display: 'flex', gap: '12px', alignItems: 'flex-end', flexWrap: 'wrap' }}>
                                                <div>
                                                    <label style={{ fontSize: '11px', color: theme.colors.text.secondary, display: 'block', marginBottom: '4px' }}>Retention (days)</label>
                                                    <input type="number" value={policyForm.policy_days} min={1}
                                                        onChange={e => setPolicyForm(f => ({ ...f, policy_days: +e.target.value }))}
                                                        style={{ padding: '6px 10px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '6px', width: '100px', fontSize: '13px' }} />
                                                </div>
                                                <div>
                                                    <label style={{ fontSize: '11px', color: theme.colors.text.secondary, display: 'block', marginBottom: '4px' }}>Policy Name</label>
                                                    <input type="text" value={policyForm.policy_name}
                                                        onChange={e => setPolicyForm(f => ({ ...f, policy_name: e.target.value }))}
                                                        style={{ padding: '6px 10px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '6px', width: '180px', fontSize: '13px' }} />
                                                </div>
                                                <div>
                                                    <label style={{ fontSize: '11px', color: theme.colors.text.secondary, display: 'block', marginBottom: '4px' }}>Legal Basis</label>
                                                    <select value={policyForm.policy_basis}
                                                        onChange={e => setPolicyForm(f => ({ ...f, policy_basis: e.target.value }))}
                                                        style={{ padding: '6px 10px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '6px', fontSize: '13px' }}>
                                                        <option>DPDPA Section 8(7)</option>
                                                        <option>Consent – Withdraw After Period</option>
                                                        <option>Legal Hold</option>
                                                        <option>Business Requirement</option>
                                                    </select>
                                                </div>
                                                <button onClick={() => handleSetPolicy(v.asset_id)} disabled={saving}
                                                    style={{ padding: '7px 16px', backgroundColor: theme.colors.primary.DEFAULT, color: 'white', border: 'none', borderRadius: '6px', fontSize: '13px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer', opacity: saving ? 0.7 : 1 }}>
                                                    {saving ? 'Saving…' : 'Save Policy'}
                                                </button>
                                            </div>
                                        </td>
                                    </tr>
                                )}
                            </React.Fragment>
                        ))}
                    </tbody>
                </table>
                </div>
            )}
        </div>
    );
}

// Maps DPDPA section keys to human-readable titles and descriptions.
const DPDPA_SECTIONS: Record<string, { title: string; description: string }> = {
    Sec4:  { title: 'Sec 4 — Lawful Processing',          description: 'Personal data processed only for lawful purposes with valid consent or legitimate use.' },
    Sec5:  { title: 'Sec 5 — Purpose Limitation',         description: 'Data collected for a specified purpose; not used beyond declared scope.' },
    Sec6:  { title: 'Sec 6 — Consent',                    description: 'Consent obtained, recorded, and linked to each data asset.' },
    Sec7:  { title: 'Sec 7 — Data Principal Rights',      description: 'Right to access, correct, erase, and nominate data principal rights.' },
    Sec8:  { title: 'Sec 8 — Data Accuracy',              description: 'Data assets scanned within the last 90 days (stale = gap).' },
    Sec9:  { title: "Sec 9 — Children's Data",            description: 'Age-indicator fields handled under heightened protection.' },
    Sec10: { title: 'Sec 10 — Data Fiduciary',            description: 'High-risk assets (score > 60) must have a DPO assigned.' },
    Sec11: { title: 'Sec 11 — Grievance Redressal Officer', description: 'Appointment and contact details of Grievance Redressal Officer.' },
    Sec12: { title: 'Sec 12 — Cross-Border Data Transfer', description: 'Data transfer outside India must comply with government-approved countries list.' },
    Sec17: { title: 'Sec 17 — Retention',                 description: 'No findings violating their declared retention period.' },
};

function DPDPAObligationChecklist() {
    const [report, setReport] = useState<DPDPAGapReport | null>(null);
    const [loading, setLoading] = useState(true);
    const [expanded, setExpanded] = useState<string | null>(null);

    useEffect(() => {
        complianceApi.getDPDPAGaps().then(r => {
            setReport(r);
            setLoading(false);
        });
    }, []);

    const handleDownloadReport = () => {
        window.open(complianceApi.getDPDPAReportUrl(), '_blank');
    };

    const passCount = report?.sections.filter(s => s.gaps === 0).length ?? 0;
    const totalSections = report?.sections.length ?? 0;

    return (
        <div style={{
            backgroundColor: theme.colors.background.card,
            borderRadius: '12px',
            border: `1px solid ${theme.colors.border.default}`,
            overflow: 'hidden',
            marginBottom: '24px',
        }}>
            <div style={{
                padding: '24px',
                borderBottom: `1px solid ${theme.colors.border.default}`,
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
                flexWrap: 'wrap',
                gap: '12px',
            }}>
                <div>
                    <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, margin: 0 }}>
                        DPDPA 2023 Obligation Checklist
                    </h3>
                    <p style={{ fontSize: '13px', color: theme.colors.text.secondary, marginTop: '6px' }}>
                        {loading ? 'Loading…' : `${passCount} / ${totalSections} sections fully compliant · ${report?.total_gaps ?? 0} total gaps`}
                    </p>
                </div>
                <button
                    onClick={handleDownloadReport}
                    style={{
                        padding: '8px 16px',
                        backgroundColor: theme.colors.primary.DEFAULT,
                        color: 'white',
                        border: 'none',
                        borderRadius: '8px',
                        fontSize: '13px',
                        fontWeight: 600,
                        cursor: 'pointer',
                    }}>
                    Download Gap Report (PDF)
                </button>
            </div>

            {loading ? (
                <div style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>Loading obligation data…</div>
            ) : !report ? (
                <div style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>
                    Obligation data unavailable. Check backend connectivity.
                </div>
            ) : (
                <div style={{ padding: '16px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    {report.sections.map((section: SectionSummary) => {
                        const meta = DPDPA_SECTIONS[section.section] ?? { title: section.section, description: '' };
                        const isPass = section.gaps === 0;
                        const isOpen = expanded === section.section;
                        const gapItems = report.gaps.filter(g => g.section === section.section);

                        return (
                            <div
                                key={section.section}
                                style={{
                                    border: `1px solid ${isPass ? theme.colors.risk.low : theme.colors.risk.high}30`,
                                    borderRadius: '8px',
                                    overflow: 'hidden',
                                    backgroundColor: isPass ? `${theme.colors.risk.low}08` : `${theme.colors.risk.high}06`,
                                }}>
                                {/* Section header row */}
                                <button
                                    onClick={() => setExpanded(isOpen ? null : section.section)}
                                    style={{
                                        width: '100%',
                                        display: 'flex',
                                        alignItems: 'center',
                                        gap: '12px',
                                        padding: '14px 16px',
                                        background: 'none',
                                        border: 'none',
                                        cursor: 'pointer',
                                        textAlign: 'left',
                                    }}>
                                    {/* Status icon */}
                                    <span style={{ fontSize: '18px', flexShrink: 0 }}>
                                        {isPass ? '✅' : '⚠️'}
                                    </span>
                                    {/* Section name + desc */}
                                    <div style={{ flex: 1, minWidth: 0 }}>
                                        <div style={{ fontWeight: 600, fontSize: '14px', color: theme.colors.text.primary }}>
                                            {meta.title}
                                        </div>
                                        <div style={{ fontSize: '12px', color: theme.colors.text.secondary, marginTop: '2px' }}>
                                            {meta.description}
                                        </div>
                                    </div>
                                    {/* Stats */}
                                    <div style={{ display: 'flex', gap: '16px', flexShrink: 0, alignItems: 'center' }}>
                                        <span style={{ fontSize: '12px', color: theme.colors.text.muted }}>
                                            {section.pass} / {section.total_assets} assets
                                        </span>
                                        {!isPass && (
                                            <span style={{
                                                fontSize: '12px',
                                                fontWeight: 700,
                                                color: 'white',
                                                backgroundColor: theme.colors.risk.high,
                                                padding: '2px 8px',
                                                borderRadius: '999px',
                                            }}>
                                                {section.gaps} gaps
                                            </span>
                                        )}
                                        <span style={{ fontSize: '12px', color: theme.colors.text.muted }}>
                                            {isOpen ? '▲' : '▼'}
                                        </span>
                                    </div>
                                </button>

                                {/* Expanded gap list */}
                                {isOpen && gapItems.length > 0 && (
                                    <div style={{
                                        borderTop: `1px solid ${theme.colors.border.default}`,
                                        padding: '0',
                                    }}>
                                        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                                            <thead>
                                                <tr style={{ backgroundColor: theme.colors.background.tertiary }}>
                                                    <th style={{ padding: '8px 16px', textAlign: 'left', color: theme.colors.text.secondary, fontSize: '11px', textTransform: 'uppercase' }}>Asset</th>
                                                    <th style={{ padding: '8px 16px', textAlign: 'left', color: theme.colors.text.secondary, fontSize: '11px', textTransform: 'uppercase' }}>Status</th>
                                                    <th style={{ padding: '8px 16px', textAlign: 'left', color: theme.colors.text.secondary, fontSize: '11px', textTransform: 'uppercase' }}>Evidence</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                {gapItems.map(gap => (
                                                    <tr key={`${gap.asset_id}-${gap.section}`} style={{ borderTop: `1px solid ${theme.colors.border.default}` }}>
                                                        <td style={{ padding: '10px 16px', fontWeight: 500, color: theme.colors.text.primary }}>
                                                            {gap.asset_name}
                                                        </td>
                                                        <td style={{ padding: '10px 16px' }}>
                                                            <span style={{
                                                                fontSize: '11px',
                                                                padding: '2px 8px',
                                                                borderRadius: '4px',
                                                                backgroundColor: gap.status === 'pass'
                                                                    ? `${theme.colors.risk.low}20`
                                                                    : gap.status === 'NOT_ASSESSED'
                                                                        ? `${theme.colors.risk.medium}20`
                                                                        : `${theme.colors.risk.high}20`,
                                                                color: gap.status === 'pass'
                                                                    ? theme.colors.risk.low
                                                                    : gap.status === 'NOT_ASSESSED'
                                                                        ? theme.colors.risk.medium
                                                                        : theme.colors.risk.high,
                                                                fontWeight: 600,
                                                                textTransform: 'uppercase',
                                                            }}>
                                                                {gap.status === 'NOT_ASSESSED' ? '⚠ Not Assessed' : gap.status}
                                                            </span>
                                                            {gap.status === 'NOT_ASSESSED' && section.section === 'Sec9' && (
                                                                <div style={{ fontSize: '11px', color: theme.colors.risk.medium, marginTop: '4px' }}>
                                                                    Enable AGE_INDICATOR scanning
                                                                </div>
                                                            )}
                                                        </td>
                                                        <td style={{ padding: '10px 16px', color: theme.colors.text.secondary }}>
                                                            {gap.evidence}
                                                        </td>
                                                    </tr>
                                                ))}
                                            </tbody>
                                        </table>
                                    </div>
                                )}
                                {isOpen && gapItems.length === 0 && (
                                    <div style={{ padding: '12px 16px', borderTop: `1px solid ${theme.colors.border.default}`, fontSize: '13px', color: theme.colors.text.secondary }}>
                                        All assets compliant for this section.
                                    </div>
                                )}
                                {isOpen && section.section === 'Sec7' && (
                                    <div style={{ borderTop: `1px solid ${theme.colors.border.default}`, padding: '10px 16px', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '8px' }}>
                                        <a href="/compliance/dpr" style={{ color: theme.colors.primary.DEFAULT, fontWeight: 600, textDecoration: 'none' }}>
                                            Manage requests →
                                        </a>
                                        <span style={{ fontSize: '12px', color: theme.colors.text.muted }}>Data Principal Rights intake</span>
                                    </div>
                                )}
                            </div>
                        );
                    })}
                </div>
            )}
        </div>
    );
}

// ─── Status color helpers ─────────────────────────────────────────────────────
const DPR_STATUS_COLORS: Record<string, { bg: string; text: string }> = {
    PENDING:     { bg: '#fef9c3', text: '#854d0e' },
    IN_PROGRESS: { bg: '#dbeafe', text: '#1e40af' },
    COMPLETED:   { bg: '#dcfce7', text: '#166534' },
    REJECTED:    { bg: '#fee2e2', text: '#991b1b' },
};

function StatusBadge({ status }: { status: string }) {
    const c = DPR_STATUS_COLORS[status] ?? { bg: theme.colors.background.tertiary, text: theme.colors.text.secondary };
    return (
        <span style={{
            display: 'inline-block',
            padding: '2px 10px',
            borderRadius: '999px',
            fontSize: '11px',
            fontWeight: 700,
            backgroundColor: c.bg,
            color: c.text,
            textTransform: 'uppercase',
        }}>
            {status.replace('_', ' ')}
        </span>
    );
}

// ─── Consent Records Section ──────────────────────────────────────────────────
function ConsentSection() {
    const [records, setRecords] = useState<ConsentRecord[]>([]);
    const [loading, setLoading] = useState(true);
    const [showForm, setShowForm] = useState(false);
    const [form, setForm] = useState({ asset_id: '', data_subject_id: '', consent_type: 'EXPLICIT', purpose: '' });
    const [saving, setSaving] = useState(false);
    const [withdrawingId, setWithdrawingId] = useState<string | null>(null);

    useEffect(() => { load(); }, []);

    const load = async () => {
        setLoading(true);
        const data = await complianceApi.listConsentRecords();
        setRecords(data);
        setLoading(false);
    };

    const handleCreate = async () => {
        setSaving(true);
        try {
            await complianceApi.createConsentRecord(form);
            setShowForm(false);
            setForm({ asset_id: '', data_subject_id: '', consent_type: 'EXPLICIT', purpose: '' });
            await load();
        } catch (e) {
            console.error('Failed to create consent record', e);
        } finally {
            setSaving(false);
        }
    };

    const handleWithdraw = async (id: string) => {
        setWithdrawingId(id);
        try {
            await complianceApi.withdrawConsent(id);
            await load();
        } catch (e) {
            console.error('Failed to withdraw consent', e);
        } finally {
            setWithdrawingId(null);
        }
    };

    const getStatusColor = (status: string) => {
        if (status === 'active') return { bg: '#dcfce7', text: '#166534' };
        if (status === 'withdrawn') return { bg: '#fee2e2', text: '#991b1b' };
        return { bg: '#fef9c3', text: '#854d0e' }; // expired
    };

    return (
        <div style={{ backgroundColor: theme.colors.background.card, borderRadius: '12px', border: `1px solid ${theme.colors.border.default}`, overflow: 'hidden', marginBottom: '24px' }}>
            <div style={{ padding: '24px', borderBottom: `1px solid ${theme.colors.border.default}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                    <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, margin: 0 }}>Consent Records</h3>
                    <p style={{ fontSize: '13px', color: theme.colors.text.secondary, marginTop: '6px' }}>
                        DPDPA Sec 6 — Consent ledger for personal data assets.
                    </p>
                </div>
                <button
                    onClick={() => setShowForm(!showForm)}
                    style={{ padding: '8px 16px', backgroundColor: theme.colors.primary.DEFAULT, color: 'white', border: 'none', borderRadius: '8px', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
                    + New Consent Record
                </button>
            </div>

            {showForm && (
                <div style={{ padding: '20px 24px', borderBottom: `1px solid ${theme.colors.border.default}`, backgroundColor: `${theme.colors.primary.DEFAULT}06` }}>
                    <div style={{ display: 'flex', gap: '12px', flexWrap: 'wrap', alignItems: 'flex-end' }}>
                        {[
                            { label: 'Asset ID', key: 'asset_id', placeholder: 'asset-uuid' },
                            { label: 'Data Subject ID', key: 'data_subject_id', placeholder: 'subject-id' },
                            { label: 'Purpose', key: 'purpose', placeholder: 'e.g. Marketing, Analytics' },
                        ].map(({ label, key, placeholder }) => (
                            <div key={key}>
                                <label style={{ fontSize: '11px', color: theme.colors.text.secondary, display: 'block', marginBottom: '4px' }}>{label}</label>
                                <input
                                    type="text"
                                    placeholder={placeholder}
                                    value={(form as any)[key]}
                                    onChange={e => setForm(f => ({ ...f, [key]: e.target.value }))}
                                    style={{ padding: '6px 10px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '6px', fontSize: '13px', width: '180px' }}
                                />
                            </div>
                        ))}
                        <div>
                            <label style={{ fontSize: '11px', color: theme.colors.text.secondary, display: 'block', marginBottom: '4px' }}>Consent Type</label>
                            <select value={form.consent_type} onChange={e => setForm(f => ({ ...f, consent_type: e.target.value }))}
                                style={{ padding: '6px 10px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '6px', fontSize: '13px' }}>
                                {['EXPLICIT', 'IMPLICIT', 'LEGITIMATE_INTEREST'].map(t => <option key={t}>{t}</option>)}
                            </select>
                        </div>
                        <div style={{ display: 'flex', gap: '8px' }}>
                            <button onClick={handleCreate} disabled={saving || !form.asset_id || !form.data_subject_id}
                                style={{ padding: '7px 16px', backgroundColor: theme.colors.primary.DEFAULT, color: 'white', border: 'none', borderRadius: '6px', fontSize: '13px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer', opacity: saving ? 0.7 : 1 }}>
                                {saving ? 'Saving…' : 'Save'}
                            </button>
                            <button onClick={() => setShowForm(false)}
                                style={{ padding: '7px 16px', backgroundColor: 'transparent', color: theme.colors.text.secondary, border: `1px solid ${theme.colors.border.default}`, borderRadius: '6px', fontSize: '13px', cursor: 'pointer' }}>
                                Cancel
                            </button>
                        </div>
                    </div>
                </div>
            )}

            {loading ? (
                <div style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>Loading consent records…</div>
            ) : records.length === 0 ? (
                <div style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>No consent records found.</div>
            ) : (
                <div className="overflow-x-auto">
                    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                        <thead style={{ backgroundColor: theme.colors.background.tertiary }}>
                            <tr>
                                {['ID', 'Asset ID', 'Subject ID', 'Type', 'Purpose', 'Status', 'Action'].map(h => (
                                    <th key={h} style={{ padding: '12px 20px', textAlign: 'left', fontSize: '11px', color: theme.colors.text.secondary, textTransform: 'uppercase' }}>{h}</th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            {records.map((r) => {
                                const sc = getStatusColor(r.status);
                                return (
                                    <tr key={r.id} style={{ borderBottom: `1px solid ${theme.colors.border.default}` }}>
                                        <td style={{ padding: '12px 20px', fontFamily: 'monospace', fontSize: '12px', color: theme.colors.text.muted }}>{(r.id || '').slice(0, 8)}…</td>
                                        <td style={{ padding: '12px 20px', fontSize: '13px', color: theme.colors.text.secondary, fontFamily: 'monospace' }}>{(r.asset_id || '').slice(0, 12)}…</td>
                                        <td style={{ padding: '12px 20px', fontSize: '13px', color: theme.colors.text.secondary }}>{r.data_subject_id || '—'}</td>
                                        <td style={{ padding: '12px 20px' }}>
                                            <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '4px', backgroundColor: theme.colors.background.tertiary, color: theme.colors.text.secondary }}>{r.consent_type}</span>
                                        </td>
                                        <td style={{ padding: '12px 20px', fontSize: '13px', color: theme.colors.text.secondary }}>{r.purpose || '—'}</td>
                                        <td style={{ padding: '12px 20px' }}>
                                            <span style={{ fontSize: '11px', fontWeight: 700, padding: '2px 10px', borderRadius: '999px', backgroundColor: sc.bg, color: sc.text, textTransform: 'uppercase' }}>
                                                {r.status}
                                            </span>
                                        </td>
                                        <td style={{ padding: '12px 20px' }}>
                                            {r.status === 'active' && (
                                                <button
                                                    onClick={() => handleWithdraw(r.id)}
                                                    disabled={withdrawingId === r.id}
                                                    style={{ padding: '4px 10px', fontSize: '12px', fontWeight: 600, color: '#991b1b', backgroundColor: 'transparent', border: '1px solid #fca5a5', borderRadius: '6px', cursor: 'pointer' }}>
                                                    {withdrawingId === r.id ? 'Withdrawing…' : 'Withdraw'}
                                                </button>
                                            )}
                                        </td>
                                    </tr>
                                );
                            })}
                        </tbody>
                    </table>
                </div>
            )}
        </div>
    );
}

// ─── DPR Management Section ───────────────────────────────────────────────────
function DPRSection() {
    const [stats, setStats] = useState<DPRStats | null>(null);
    const [dprs, setDPRs] = useState<DPRItem[]>([]);
    const [loading, setLoading] = useState(true);
    const [showModal, setShowModal] = useState(false);
    const [submitForm, setSubmitForm] = useState({ request_type: 'ACCESS', data_principal_id: '', data_principal_email: '', request_details: '' });
    const [submitting, setSubmitting] = useState(false);
    const [updatingId, setUpdatingId] = useState<string | null>(null);

    useEffect(() => { loadData(); }, []);

    const loadData = async () => {
        setLoading(true);
        const [statsData, dprsData] = await Promise.all([
            complianceApi.getDPRStats(),
            complianceApi.listDPRs(),
        ]);
        setStats(statsData);
        setDPRs(dprsData);
        setLoading(false);
    };

    const handleSubmit = async () => {
        setSubmitting(true);
        try {
            await complianceApi.submitDPR({
                request_type: submitForm.request_type,
                data_principal_id: submitForm.data_principal_id,
                data_principal_email: submitForm.data_principal_email || undefined,
                request_details: submitForm.request_details ? { notes: submitForm.request_details } : undefined,
            });
            setShowModal(false);
            setSubmitForm({ request_type: 'ACCESS', data_principal_id: '', data_principal_email: '', request_details: '' });
            await loadData();
        } catch (e) {
            console.error('Failed to submit DPR', e);
        } finally {
            setSubmitting(false);
        }
    };

    const handleUpdateStatus = async (id: string, status: string) => {
        setUpdatingId(id);
        try {
            await complianceApi.updateDPRStatus(id, status);
            await loadData();
        } catch (e) {
            console.error('Failed to update DPR status', e);
        } finally {
            setUpdatingId(null);
        }
    };

    const overdueCount = dprs.filter(d => {
        if (d.status !== 'PENDING') return false;
        return (Date.now() - new Date(d.created_at).getTime()) > 30 * 24 * 60 * 60 * 1000;
    }).length;

    return (
        <div style={{ backgroundColor: theme.colors.background.card, borderRadius: '12px', border: `1px solid ${theme.colors.border.default}`, overflow: 'hidden', marginBottom: '24px' }}>
            <div style={{ padding: '24px', borderBottom: `1px solid ${theme.colors.border.default}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                    <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, margin: 0 }}>Data Principal Rights (DPR)</h3>
                    <p style={{ fontSize: '13px', color: theme.colors.text.secondary, marginTop: '6px' }}>
                        DPDPA Sec 7 — Manage access, correction, erasure, nomination, and grievance requests.
                    </p>
                </div>
                <button
                    onClick={() => setShowModal(true)}
                    style={{ padding: '8px 16px', backgroundColor: theme.colors.primary.DEFAULT, color: 'white', border: 'none', borderRadius: '8px', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
                    + Submit New Request
                </button>
            </div>

            {/* Overdue banner */}
            {overdueCount > 0 && (
                <div style={{ margin: '16px 24px 0', padding: '12px 16px', backgroundColor: '#fee2e2', border: '1px solid #fca5a5', borderRadius: '8px', color: '#991b1b', fontSize: '13px', fontWeight: 600 }}>
                    ⚠ {overdueCount} pending request{overdueCount > 1 ? 's' : ''} exceeded the 30-day response deadline — action required.
                </div>
            )}

            {/* Stats Cards */}
            {stats && (
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px', padding: '20px 24px' }}>
                    {[
                        { label: 'Total Requests', value: stats.total ?? dprs.length, color: theme.colors.text.primary },
                        { label: 'Pending', value: stats.pending ?? dprs.filter((d) => d.status === 'PENDING').length, color: '#854d0e' },
                        { label: 'In Progress', value: stats.in_progress ?? dprs.filter((d) => d.status === 'IN_PROGRESS').length, color: '#1e40af' },
                        { label: 'Completed', value: stats.completed ?? dprs.filter((d) => d.status === 'COMPLETED').length, color: '#166534' },
                    ].map(({ label, value, color }) => (
                        <div key={label} style={{ padding: '16px', backgroundColor: theme.colors.background.tertiary, borderRadius: '8px', textAlign: 'center' }}>
                            <div style={{ fontSize: '24px', fontWeight: 800, color }}>{value ?? '—'}</div>
                            <div style={{ fontSize: '12px', color: theme.colors.text.secondary, marginTop: '4px' }}>{label}</div>
                        </div>
                    ))}
                </div>
            )}

            {loading ? (
                <div style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>Loading requests…</div>
            ) : dprs.length === 0 ? (
                <div style={{ padding: '32px', textAlign: 'center', color: theme.colors.text.secondary }}>No DPR requests found.</div>
            ) : (
                <div className="overflow-x-auto">
                    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                        <thead style={{ backgroundColor: theme.colors.background.tertiary }}>
                            <tr>
                                {['ID', 'Type', 'Status', 'Principal ID', 'Due Date', 'Created', 'Action'].map(h => (
                                    <th key={h} style={{ padding: '12px 20px', textAlign: 'left', fontSize: '11px', color: theme.colors.text.secondary, textTransform: 'uppercase' }}>{h}</th>
                                ))}
                            </tr>
                        </thead>
                        <tbody>
                            {dprs.map((d) => (
                                <tr key={d.id} style={{ borderBottom: `1px solid ${theme.colors.border.default}` }}>
                                    <td style={{ padding: '12px 20px', fontFamily: 'monospace', fontSize: '12px', color: theme.colors.text.muted }}>{(d.id || '').slice(0, 8)}…</td>
                                    <td style={{ padding: '12px 20px' }}>
                                        <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '4px', backgroundColor: theme.colors.background.tertiary, color: theme.colors.text.secondary, fontWeight: 600 }}>{d.request_type}</span>
                                    </td>
                                    <td style={{ padding: '12px 20px' }}><StatusBadge status={d.status} /></td>
                                    <td style={{ padding: '12px 20px', fontSize: '13px', color: theme.colors.text.secondary }}>{d.data_principal_id || '—'}</td>
                                    <td style={{ padding: '12px 20px', fontSize: '13px', color: d.due_date && new Date(d.due_date) < new Date() ? '#991b1b' : theme.colors.text.secondary }}>
                                        {d.due_date ? new Date(d.due_date).toLocaleDateString() : '—'}
                                    </td>
                                    <td style={{ padding: '12px 20px', fontSize: '12px', color: theme.colors.text.muted }}>{d.created_at ? new Date(d.created_at).toLocaleDateString() : '—'}</td>
                                    <td style={{ padding: '12px 20px' }}>
                                        {(d.status === 'PENDING' || d.status === 'IN_PROGRESS') && (
                                            <select
                                                disabled={updatingId === d.id}
                                                defaultValue=""
                                                onChange={e => { if (e.target.value) handleUpdateStatus(d.id, e.target.value); }}
                                                style={{ padding: '4px 8px', fontSize: '12px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '6px', cursor: 'pointer', backgroundColor: theme.colors.background.card, color: theme.colors.text.primary }}>
                                                <option value="" disabled>Update status</option>
                                                <option value="IN_PROGRESS">In Progress</option>
                                                <option value="COMPLETED">Completed</option>
                                                <option value="REJECTED">Rejected</option>
                                            </select>
                                        )}
                                    </td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                </div>
            )}

            {/* Submit Modal */}
            {showModal && (
                <div style={{ position: 'fixed', inset: 0, backgroundColor: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000 }}>
                    <div style={{ backgroundColor: theme.colors.background.card, borderRadius: '12px', padding: '32px', width: '480px', maxWidth: '95vw', boxShadow: '0 20px 60px rgba(0,0,0,0.3)' }}>
                        <h3 style={{ fontSize: '18px', fontWeight: 700, color: theme.colors.text.primary, marginBottom: '24px' }}>Submit DPR Request</h3>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                            <div>
                                <label style={{ fontSize: '12px', color: theme.colors.text.secondary, display: 'block', marginBottom: '6px', fontWeight: 600 }}>Request Type</label>
                                <select value={submitForm.request_type} onChange={e => setSubmitForm(f => ({ ...f, request_type: e.target.value }))}
                                    style={{ width: '100%', padding: '8px 12px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', fontSize: '13px', backgroundColor: theme.colors.background.card, color: theme.colors.text.primary }}>
                                    {['ACCESS', 'CORRECTION', 'ERASURE', 'NOMINATION', 'GRIEVANCE'].map(t => <option key={t}>{t}</option>)}
                                </select>
                            </div>
                            <div>
                                <label style={{ fontSize: '12px', color: theme.colors.text.secondary, display: 'block', marginBottom: '6px', fontWeight: 600 }}>Data Principal ID <span style={{ color: '#ef4444' }}>*</span></label>
                                <input type="text" value={submitForm.data_principal_id} onChange={e => setSubmitForm(f => ({ ...f, data_principal_id: e.target.value }))}
                                    placeholder="user-id or identifier"
                                    style={{ width: '100%', padding: '8px 12px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', fontSize: '13px', boxSizing: 'border-box' }} />
                            </div>
                            <div>
                                <label style={{ fontSize: '12px', color: theme.colors.text.secondary, display: 'block', marginBottom: '6px', fontWeight: 600 }}>Email (optional)</label>
                                <input type="email" value={submitForm.data_principal_email} onChange={e => setSubmitForm(f => ({ ...f, data_principal_email: e.target.value }))}
                                    placeholder="principal@example.com"
                                    style={{ width: '100%', padding: '8px 12px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', fontSize: '13px', boxSizing: 'border-box' }} />
                            </div>
                            <div>
                                <label style={{ fontSize: '12px', color: theme.colors.text.secondary, display: 'block', marginBottom: '6px', fontWeight: 600 }}>Details (optional)</label>
                                <textarea value={submitForm.request_details} onChange={e => setSubmitForm(f => ({ ...f, request_details: e.target.value }))}
                                    rows={3} placeholder="Additional context or notes…"
                                    style={{ width: '100%', padding: '8px 12px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', fontSize: '13px', resize: 'vertical', boxSizing: 'border-box', fontFamily: 'inherit' }} />
                            </div>
                        </div>
                        <div style={{ display: 'flex', gap: '12px', marginTop: '24px', justifyContent: 'flex-end' }}>
                            <button onClick={() => setShowModal(false)}
                                style={{ padding: '8px 20px', backgroundColor: 'transparent', color: theme.colors.text.secondary, border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', fontSize: '13px', cursor: 'pointer' }}>
                                Cancel
                            </button>
                            <button onClick={handleSubmit} disabled={submitting || !submitForm.data_principal_id}
                                style={{ padding: '8px 20px', backgroundColor: theme.colors.primary.DEFAULT, color: 'white', border: 'none', borderRadius: '8px', fontSize: '13px', fontWeight: 600, cursor: submitting ? 'not-allowed' : 'pointer', opacity: submitting || !submitForm.data_principal_id ? 0.7 : 1 }}>
                                {submitting ? 'Submitting…' : 'Submit Request'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}

// ─── GRO Settings Section ─────────────────────────────────────────────────────
function GROSettingsSection() {
    const [form, setForm] = useState({ gro_name: '', gro_email: '', gro_phone: '', is_significant_data_fiduciary: false });
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [saved, setSaved] = useState(false);

    useEffect(() => {
        complianceApi.getGROSettings().then(data => {
            if (data) {
                setForm({
                    gro_name: data.gro_name ?? '',
                    gro_email: data.gro_email ?? '',
                    gro_phone: data.gro_phone ?? '',
                    is_significant_data_fiduciary: data.is_significant_data_fiduciary ?? false,
                });
            }
            setLoading(false);
        });
    }, []);

    const handleSave = async () => {
        setSaving(true);
        setSaved(false);
        try {
            await complianceApi.updateGROSettings(form);
            setSaved(true);
            setTimeout(() => setSaved(false), 3000);
        } catch (e) {
            console.error('Failed to save GRO settings', e);
        } finally {
            setSaving(false);
        }
    };

    return (
        <div style={{ backgroundColor: theme.colors.background.card, borderRadius: '12px', border: `1px solid ${theme.colors.border.default}`, overflow: 'hidden', marginBottom: '24px' }}>
            <div style={{ padding: '24px', borderBottom: `1px solid ${theme.colors.border.default}` }}>
                <h3 style={{ fontSize: '18px', fontWeight: 600, color: theme.colors.text.primary, margin: 0 }}>Grievance Redressal Officer (GRO) Settings</h3>
                <p style={{ fontSize: '13px', color: theme.colors.text.secondary, marginTop: '6px' }}>
                    DPDPA Sec 11 — Contact details for the appointed Grievance Redressal Officer.
                </p>
            </div>

            <div style={{ padding: '24px' }}>
                {loading ? (
                    <div style={{ color: theme.colors.text.secondary, fontSize: '13px' }}>Loading GRO settings…</div>
                ) : (
                    <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '20px', maxWidth: '640px' }}>
                        <div>
                            <label style={{ fontSize: '12px', color: theme.colors.text.secondary, display: 'block', marginBottom: '6px', fontWeight: 600 }}>GRO Name</label>
                            <input type="text" value={form.gro_name} onChange={e => setForm(f => ({ ...f, gro_name: e.target.value }))}
                                placeholder="Full name"
                                style={{ width: '100%', padding: '8px 12px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', fontSize: '13px', boxSizing: 'border-box' }} />
                        </div>
                        <div>
                            <label style={{ fontSize: '12px', color: theme.colors.text.secondary, display: 'block', marginBottom: '6px', fontWeight: 600 }}>GRO Email</label>
                            <input type="email" value={form.gro_email} onChange={e => setForm(f => ({ ...f, gro_email: e.target.value }))}
                                placeholder="gro@company.com"
                                style={{ width: '100%', padding: '8px 12px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', fontSize: '13px', boxSizing: 'border-box' }} />
                        </div>
                        <div>
                            <label style={{ fontSize: '12px', color: theme.colors.text.secondary, display: 'block', marginBottom: '6px', fontWeight: 600 }}>GRO Phone</label>
                            <input type="tel" value={form.gro_phone} onChange={e => setForm(f => ({ ...f, gro_phone: e.target.value }))}
                                placeholder="+91-XXXXXXXXXX"
                                style={{ width: '100%', padding: '8px 12px', border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', fontSize: '13px', boxSizing: 'border-box' }} />
                        </div>
                        <div style={{ display: 'flex', alignItems: 'center', gap: '10px', paddingTop: '22px' }}>
                            <input type="checkbox" id="sdf-checkbox" checked={form.is_significant_data_fiduciary}
                                onChange={e => setForm(f => ({ ...f, is_significant_data_fiduciary: e.target.checked }))}
                                style={{ width: '16px', height: '16px', cursor: 'pointer' }} />
                            <label htmlFor="sdf-checkbox" style={{ fontSize: '13px', color: theme.colors.text.primary, cursor: 'pointer' }}>
                                Significant Data Fiduciary
                            </label>
                        </div>
                        <div style={{ gridColumn: '1 / -1', display: 'flex', gap: '12px', alignItems: 'center', marginTop: '4px' }}>
                            <button onClick={handleSave} disabled={saving}
                                style={{ padding: '8px 24px', backgroundColor: theme.colors.primary.DEFAULT, color: 'white', border: 'none', borderRadius: '8px', fontSize: '13px', fontWeight: 600, cursor: saving ? 'not-allowed' : 'pointer', opacity: saving ? 0.7 : 1 }}>
                                {saving ? 'Saving…' : 'Save Settings'}
                            </button>
                            {saved && <span style={{ fontSize: '13px', color: '#166534', fontWeight: 600 }}>✓ Saved successfully</span>}
                        </div>
                    </div>
                )}
            </div>
        </div>
    );
}

function SystemHealthStatus() {
    const [healthData, setHealthData] = React.useState<HealthData | null>(null);
    const [loading, setLoading] = React.useState(true);

    React.useEffect(() => {
        fetchHealthData();
        const interval = setInterval(fetchHealthData, 30000); // Refresh every 30 seconds
        return () => clearInterval(interval);
    }, []);

    const fetchHealthData = async () => {
        try {
            const data = await get<HealthData>('/health/components');
            setHealthData(data);
        } catch (error) {
            console.error('Failed to fetch health data', error);
        } finally {
            setLoading(false);
        }
    };

    if (loading) {
        return <div style={{ fontSize: '13px', color: theme.colors.text.secondary }}>Loading system health...</div>;
    }

    if (!healthData || !healthData.components) {
        return <div style={{ fontSize: '13px', color: theme.colors.text.secondary }}>Health data unavailable</div>;
    }

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'online': return '✅';
            case 'degraded': return '⚠️';
            case 'offline': return '❌';
            default: return '❓';
        }
    };

    return (
        <div style={{ fontSize: '13px', color: theme.colors.text.secondary, lineHeight: '1.6' }}>
            {healthData.components.map((comp) => (
                <p key={comp.name}>
                    {getStatusIcon(comp.status)} {comp.name}: <strong style={{
                        color: comp.status === 'online' ? theme.colors.risk.low :
                            comp.status === 'degraded' ? theme.colors.risk.medium :
                                theme.colors.risk.critical
                    }}>{comp.status}</strong>
                    {comp.message && <span style={{ fontSize: '12px', marginLeft: '8px' }}>({comp.message})</span>}
                </p>
            ))}
            <p style={{ marginTop: '12px', paddingTop: '12px', borderTop: `1px solid ${theme.colors.border.default}` }}>
                Last check: <strong>{new Date(healthData.timestamp).toLocaleTimeString()}</strong>
            </p>
            <p style={{ marginTop: '4px' }}>
                Overall: <strong style={{
                    color: healthData.status === 'healthy' ? theme.colors.risk.low :
                        healthData.status === 'degraded' ? theme.colors.risk.medium :
                            theme.colors.risk.critical
                }}>{healthData.status}</strong>
            </p>
        </div>
    );
}
