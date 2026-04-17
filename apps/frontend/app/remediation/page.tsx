'use client';

import React, { useState, useEffect } from 'react';
import { Shield, AlertTriangle, CheckCircle, Clock, Play, RotateCcw, History, BookOpen, ChevronDown, ChevronRight, Search, Siren, X } from 'lucide-react';
import Link from 'next/link';
import { theme } from '@/design-system/theme';
import { remediationApi, SOP, EscalationPreview } from '@/services/remediation.api';

interface RemediationTask {
    id: string;
    finding_id: string;
    asset_name: string;
    asset_path: string;
    pii_type: string;
    risk_level: string;
    action_type: 'MASK' | 'DELETE' | 'ENCRYPT';
    status: 'PENDING' | 'IN_PROGRESS' | 'COMPLETED' | 'FAILED';
    created_at: string;
    completed_at?: string;
    error_message?: string;
}

interface RemediationStats {
    totalTasks: number;
    pendingTasks: number;
    completedTasks: number;
    failedTasks: number;
    successRate: number;
}

export default function RemediationPage() {
    const [tasks, setTasks] = useState<RemediationTask[]>([]);
    const [stats, setStats] = useState<RemediationStats>({
        totalTasks: 0,
        pendingTasks: 0,
        completedTasks: 0,
        failedTasks: 0,
        successRate: 0
    });
    const [loading, setLoading] = useState(true);
    const [filter, setFilter] = useState<'ALL' | 'PENDING' | 'COMPLETED' | 'FAILED'>('ALL');

    // Tab state
    const [activeTab, setActiveTab] = useState<'tasks' | 'sops' | 'escalation'>('tasks');

    // SOPs state
    const [sops, setSOPs] = useState<SOP[]>([]);
    const [sopsLoading, setSOPsLoading] = useState(false);
    const [sopSearch, setSOPSearch] = useState('');
    const [expandedSOP, setExpandedSOP] = useState<string | null>(null);

    // Escalation state
    const [escalationPreview, setEscalationPreview] = useState<EscalationPreview | null>(null);
    const [escalationLoading, setEscalationLoading] = useState(false);
    const [escalationRunning, setEscalationRunning] = useState(false);
    const [showEscalationModal, setShowEscalationModal] = useState(false);
    const [escalationToast, setEscalationToast] = useState<{ msg: string; ok: boolean } | null>(null);

    useEffect(() => {
        fetchRemediationData();
    }, []);

    useEffect(() => {
        if (activeTab === 'sops' && sops.length === 0) {
            setSOPsLoading(true);
            remediationApi.getSOPs()
                .then(res => setSOPs(res.sops || []))
                .catch(err => console.error('Failed to load SOPs:', err))
                .finally(() => setSOPsLoading(false));
        }
    }, [activeTab]);

    const handlePreviewEscalation = async () => {
        setEscalationLoading(true);
        try {
            const preview = await remediationApi.previewEscalation();
            setEscalationPreview(preview);
            setShowEscalationModal(true);
        } catch (err) {
            console.error('Failed to preview escalation:', err);
        } finally {
            setEscalationLoading(false);
        }
    };

    const handleRunEscalation = async () => {
        setEscalationRunning(true);
        try {
            const result = await remediationApi.runEscalation();
            setShowEscalationModal(false);
            setEscalationToast({ msg: result.message || `Escalated ${result.escalated} finding(s)`, ok: true });
        } catch (err) {
            setEscalationToast({ msg: 'Escalation failed. Please try again.', ok: false });
        } finally {
            setEscalationRunning(false);
            setTimeout(() => setEscalationToast(null), 4000);
        }
    };

    const filteredSOPs = sops.filter(s =>
        sopSearch === '' ||
        s.issue_type.toLowerCase().includes(sopSearch.toLowerCase()) ||
        s.title.toLowerCase().includes(sopSearch.toLowerCase())
    );

    const fetchRemediationData = async () => {
        try {
            setLoading(true);
            const response = await remediationApi.getRemediationHistory({ limit: 100 });

            // Adapt API response to UI model — use enriched fields from backend JOIN
            const realTasks: RemediationTask[] = response.history.map(item => ({
                id: item.id,
                finding_id: item.finding_id || '',
                asset_name: item.asset_name || 'Unknown Asset',
                asset_path: item.asset_path || '',
                pii_type: item.pii_type || item.pattern_name || 'Unknown',
                risk_level: item.risk_level || item.severity || 'Medium',
                action_type: item.action as any,
                status: item.status as any,
                created_at: item.executed_at,
                completed_at: item.status === 'COMPLETED' ? item.executed_at : undefined
            }));

            setTasks(realTasks);
            calculateStats(realTasks);
        } catch (error) {
            console.error('Failed to fetch remediation data:', error);
        } finally {
            setLoading(false);
        }
    };

    const calculateStats = (taskList: RemediationTask[]) => {
        const totalTasks = taskList.length;
        const completedTasks = taskList.filter(t => t.status === 'COMPLETED').length;
        const failedTasks = taskList.filter(t => t.status === 'FAILED').length;
        const pendingTasks = taskList.filter(t => t.status === 'PENDING' || t.status === 'IN_PROGRESS').length;
        const successRate = totalTasks > 0 ? Math.round((completedTasks / (completedTasks + failedTasks || 1)) * 100) : 0;

        setStats({
            totalTasks,
            pendingTasks,
            completedTasks,
            failedTasks,
            successRate
        });
    };

    const filteredTasks = tasks.filter(task => {
        if (filter === 'ALL') return true;
        return task.status === filter;
    });

    const getStatusIcon = (status: string) => {
        switch (status) {
            case 'COMPLETED': return <CheckCircle style={{ width: '16px', height: '16px', color: theme.colors.status.success }} />;
            case 'IN_PROGRESS': return <Clock style={{ width: '16px', height: '16px', color: theme.colors.status.info }} />;
            case 'FAILED': return <AlertTriangle style={{ width: '16px', height: '16px', color: theme.colors.status.error }} />;
            default: return <Clock style={{ width: '16px', height: '16px', color: theme.colors.text.muted }} />;
        }
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'COMPLETED': return theme.colors.status.success;
            case 'IN_PROGRESS': return theme.colors.status.info;
            case 'FAILED': return theme.colors.status.error;
            default: return theme.colors.text.secondary;
        }
    };

    const handleRetryTask = async (taskId: string) => {
        try {
            await remediationApi.rollback(taskId);
            await fetchRemediationData();
        } catch (error) {
            console.error('Failed to retry task:', error);
        }
    };

    const handleNewRemediation = () => {
        // Redirect to findings page as that's where remediations are created
        window.location.href = '/findings?action=remediate';
    };

    const handleRunAllPending = async () => {
        const pending = tasks.filter(t => t.status === 'PENDING');
        if (pending.length === 0) return;

        // In a real app, this would be a single batch API call
        // Here we simulate it by iterating
        for (const task of pending) {
            try {
                // Determine action based on type (defaulting to MASK for safety if unknown)
                const action = task.action_type || 'MASK';
                await remediationApi.executeRemediation({
                    finding_ids: [task.finding_id],
                    action_type: action as any,
                    user_id: 'current-user-id'
                });
            } catch (e) {
                console.error(`Failed to execute task ${task.id}`, e);
            }
        }
        // Refresh after batch
        await fetchRemediationData();
    };

    if (loading) {
        return (
            <div style={{ minHeight: '100vh', backgroundColor: theme.colors.background.primary, padding: '32px' }}>
                <div style={{ color: theme.colors.text.primary }}>Loading remediation dashboard...</div>
            </div>
        );
    }

    return (
        <div style={{ minHeight: '100vh', backgroundColor: theme.colors.background.primary }}>
            <div className="container" style={{ padding: '32px', maxWidth: '1600px', margin: '0 auto' }}>

                {/* Header */}
                <div style={{ marginBottom: '32px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                            <h1 style={{ fontSize: '32px', fontWeight: 800, color: theme.colors.text.primary, marginBottom: '8px', letterSpacing: '-0.02em' }}>
                                Remediation Center
                            </h1>
                            <p style={{ color: theme.colors.text.secondary, fontSize: '16px' }}>
                                Manage automated risk reduction actions and track remediation progress
                            </p>
                        </div>
                        <div style={{ display: 'flex', gap: '12px' }}>
                            <Link
                                href="/history"
                                style={{
                                    display: 'inline-flex', alignItems: 'center', gap: '8px',
                                    padding: '10px 16px', borderRadius: '8px',
                                    border: `1px solid ${theme.colors.border.default}`,
                                    backgroundColor: theme.colors.background.card,
                                    color: theme.colors.text.secondary,
                                    fontSize: '14px', fontWeight: 500, textDecoration: 'none',
                                }}
                            >
                                <History style={{ width: '16px', height: '16px' }} />
                                Action History
                            </Link>
                            <button
                                onClick={handleRunAllPending}
                                disabled={loading || stats.pendingTasks === 0}
                                style={{
                                    padding: '12px 20px',
                                    borderRadius: '8px',
                                    border: `1px solid ${theme.colors.border.default}`,
                                    backgroundColor: theme.colors.background.card,
                                    color: stats.pendingTasks === 0 ? theme.colors.text.muted : theme.colors.text.primary,
                                    fontWeight: 600,
                                    cursor: stats.pendingTasks === 0 ? 'not-allowed' : 'pointer',
                                    opacity: stats.pendingTasks === 0 ? 0.6 : 1
                                }}>
                                <Play style={{ width: '16px', height: '16px', marginRight: '8px', display: 'inline' }} />
                                Run All Pending
                            </button>
                            <button
                                onClick={handleNewRemediation}
                                style={{
                                    padding: '12px 20px',
                                    borderRadius: '8px',
                                    border: `1px solid ${theme.colors.primary.DEFAULT}`,
                                    backgroundColor: theme.colors.primary.DEFAULT,
                                    color: '#fff',
                                    fontWeight: 600,
                                    cursor: 'pointer'
                                }}>
                                <Shield style={{ width: '16px', height: '16px', marginRight: '8px', display: 'inline' }} />
                                New Remediation
                            </button>
                        </div>
                    </div>
                </div>

                {/* Stats Cards */}
                <div style={{ display: 'grid', gridTemplateColumns: 'repeat(5, 1fr)', gap: '20px', marginBottom: '32px' }}>
                    <StatCard
                        title="Total Tasks"
                        value={stats.totalTasks}
                        color={theme.colors.text.primary}
                        icon={<Shield style={{ width: '20px', height: '20px' }} />}
                    />
                    <StatCard
                        title="Pending"
                        value={stats.pendingTasks}
                        color={theme.colors.status.warning}
                        icon={<Clock style={{ width: '20px', height: '20px' }} />}
                    />
                    <StatCard
                        title="Completed"
                        value={stats.completedTasks}
                        color={theme.colors.status.success}
                        icon={<CheckCircle style={{ width: '20px', height: '20px' }} />}
                    />
                    <StatCard
                        title="Failed"
                        value={stats.failedTasks}
                        color={theme.colors.status.error}
                        icon={<AlertTriangle style={{ width: '20px', height: '20px' }} />}
                    />
                    <StatCard
                        title="Success Rate"
                        value={`${stats.successRate}%`}
                        color={stats.successRate > 80 ? theme.colors.status.success : theme.colors.status.warning}
                        icon={<CheckCircle style={{ width: '20px', height: '20px' }} />}
                    />
                </div>

                {/* Tab Nav */}
                <div style={{ display: 'flex', gap: '4px', marginBottom: '24px', borderBottom: `1px solid ${theme.colors.border.default}` }}>
                    {([
                        { id: 'tasks', label: 'Tasks', icon: <Shield style={{ width: '15px', height: '15px' }} /> },
                        { id: 'sops', label: 'SOPs', icon: <BookOpen style={{ width: '15px', height: '15px' }} /> },
                        { id: 'escalation', label: 'Escalation', icon: <Siren style={{ width: '15px', height: '15px' }} /> },
                    ] as const).map(tab => (
                        <button
                            key={tab.id}
                            onClick={() => setActiveTab(tab.id)}
                            style={{
                                display: 'flex', alignItems: 'center', gap: '6px',
                                padding: '10px 20px', fontSize: '14px', fontWeight: 600,
                                cursor: 'pointer', background: 'none', border: 'none',
                                borderBottom: activeTab === tab.id ? `2px solid ${theme.colors.primary.DEFAULT}` : '2px solid transparent',
                                color: activeTab === tab.id ? theme.colors.primary.DEFAULT : theme.colors.text.secondary,
                                marginBottom: '-1px',
                                transition: 'color 0.15s',
                            }}
                        >
                            {tab.icon}{tab.label}
                        </button>
                    ))}
                </div>

                {/* Tasks Tab */}
                {activeTab === 'tasks' && <div style={{
                    backgroundColor: theme.colors.background.card,
                    borderRadius: '12px',
                    border: `1px solid ${theme.colors.border.default}`,
                    overflow: 'hidden'
                }}>
                    <div style={{ padding: '24px', borderBottom: `1px solid ${theme.colors.border.default}` }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                            <h2 style={{ fontSize: '18px', fontWeight: 700, color: theme.colors.text.primary, margin: 0 }}>
                                Remediation Tasks
                            </h2>
                            <div style={{ display: 'flex', gap: '8px' }}>
                                {(['ALL', 'PENDING', 'COMPLETED', 'FAILED'] as const).map(status => (
                                    <button
                                        key={status}
                                        onClick={() => setFilter(status)}
                                        style={{
                                            padding: '6px 12px',
                                            borderRadius: '6px',
                                            border: `1px solid ${filter === status ? theme.colors.primary.DEFAULT : theme.colors.border.default}`,
                                            backgroundColor: filter === status ? `${theme.colors.primary.DEFAULT}10` : 'transparent',
                                            color: filter === status ? theme.colors.primary.DEFAULT : theme.colors.text.secondary,
                                            fontSize: '12px',
                                            fontWeight: 600,
                                            cursor: 'pointer'
                                        }}
                                    >
                                        {status === 'ALL' ? 'All Tasks' : status}
                                    </button>
                                ))}
                            </div>
                        </div>
                    </div>

                    <div style={{ overflowX: 'auto' }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                            <thead>
                                <tr style={{ backgroundColor: theme.colors.background.tertiary }}>
                                    <th style={tableHeaderStyle}>Asset</th>
                                    <th style={tableHeaderStyle}>PII Type</th>
                                    <th style={tableHeaderStyle}>Action</th>
                                    <th style={tableHeaderStyle}>Risk Level</th>
                                    <th style={tableHeaderStyle}>Status</th>
                                    <th style={tableHeaderStyle}>Created</th>
                                    <th style={tableHeaderStyle}>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {filteredTasks.length > 0 ? (
                                    filteredTasks.map(task => (
                                        <tr key={task.id} style={{ borderBottom: `1px solid ${theme.colors.border.subtle}` }}>
                                            <td style={tableCellStyle}>
                                                <div style={{ fontWeight: 600, color: theme.colors.text.primary }}>
                                                    {task.asset_name}
                                                </div>
                                                <div style={{ fontSize: '12px', color: theme.colors.text.muted, fontFamily: 'monospace' }}>
                                                    {task.asset_path}
                                                </div>
                                            </td>
                                            <td style={tableCellStyle}>
                                                <span style={{
                                                    padding: '4px 8px',
                                                    borderRadius: '4px',
                                                    backgroundColor: theme.colors.background.tertiary,
                                                    fontSize: '12px',
                                                    fontWeight: 600,
                                                    color: theme.colors.text.secondary
                                                }}>
                                                    {task.pii_type}
                                                </span>
                                            </td>
                                            <td style={tableCellStyle}>
                                                <span style={{
                                                    padding: '4px 8px',
                                                    borderRadius: '4px',
                                                    backgroundColor: getActionColor(task.action_type),
                                                    fontSize: '12px',
                                                    fontWeight: 600,
                                                    color: '#fff'
                                                }}>
                                                    {task.action_type}
                                                </span>
                                            </td>
                                            <td style={tableCellStyle}>
                                                <span style={{
                                                    padding: '4px 8px',
                                                    borderRadius: '12px',
                                                    fontSize: '12px',
                                                    fontWeight: 700,
                                                    backgroundColor: `${getRiskColor(task.risk_level)}20`,
                                                    color: getRiskColor(task.risk_level)
                                                }}>
                                                    {task.risk_level}
                                                </span>
                                            </td>
                                            <td style={tableCellStyle}>
                                                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                                    {getStatusIcon(task.status)}
                                                    <span style={{
                                                        fontSize: '12px',
                                                        fontWeight: 600,
                                                        color: getStatusColor(task.status)
                                                    }}>
                                                        {task.status.replace('_', ' ')}
                                                    </span>
                                                </div>
                                            </td>
                                            <td style={tableCellStyle}>
                                                <div style={{ fontSize: '13px', color: theme.colors.text.secondary }}>
                                                    {new Date(task.created_at).toLocaleDateString()}
                                                </div>
                                                <div style={{ fontSize: '11px', color: theme.colors.text.muted }}>
                                                    {new Date(task.created_at).toLocaleTimeString()}
                                                </div>
                                            </td>
                                            <td style={tableCellStyle}>
                                                {task.status === 'FAILED' && (
                                                    <button
                                                        onClick={() => handleRetryTask(task.id)}
                                                        style={{
                                                            padding: '6px 12px',
                                                            borderRadius: '6px',
                                                            border: `1px solid ${theme.colors.primary.DEFAULT}`,
                                                            backgroundColor: 'transparent',
                                                            color: theme.colors.primary.DEFAULT,
                                                            fontSize: '12px',
                                                            fontWeight: 600,
                                                            cursor: 'pointer',
                                                            display: 'flex',
                                                            alignItems: 'center',
                                                            gap: '4px'
                                                        }}
                                                    >
                                                        <RotateCcw style={{ width: '12px', height: '12px' }} />
                                                        Retry
                                                    </button>
                                                )}
                                                {task.status === 'IN_PROGRESS' && (
                                                    <span style={{ fontSize: '12px', color: theme.colors.text.muted }}>
                                                        Processing...
                                                    </span>
                                                )}
                                            </td>
                                        </tr>
                                    ))
                                ) : (
                                    <tr>
                                        <td colSpan={7} style={{ padding: '48px', textAlign: 'center', color: theme.colors.text.secondary }}>
                                            No {filter !== 'ALL' ? filter.toLowerCase() : ''} remediation tasks found
                                        </td>
                                    </tr>
                                )}
                            </tbody>
                        </table>
                    </div>
                </div>}

                {/* SOPs Tab */}
                {activeTab === 'sops' && (
                    <div style={{ backgroundColor: theme.colors.background.card, borderRadius: '12px', border: `1px solid ${theme.colors.border.default}`, padding: '24px' }}>
                        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '20px' }}>
                            <h2 style={{ fontSize: '18px', fontWeight: 700, color: theme.colors.text.primary, margin: 0 }}>
                                Standard Operating Procedures
                            </h2>
                            <div style={{ position: 'relative' }}>
                                <Search style={{ position: 'absolute', left: '10px', top: '50%', transform: 'translateY(-50%)', width: '14px', height: '14px', color: theme.colors.text.muted }} />
                                <input
                                    type="text"
                                    placeholder="Search SOPs..."
                                    value={sopSearch}
                                    onChange={e => setSOPSearch(e.target.value)}
                                    style={{
                                        paddingLeft: '32px', paddingRight: '12px', paddingTop: '8px', paddingBottom: '8px',
                                        borderRadius: '8px', border: `1px solid ${theme.colors.border.default}`,
                                        fontSize: '13px', color: theme.colors.text.primary,
                                        backgroundColor: theme.colors.background.primary, outline: 'none', width: '240px'
                                    }}
                                />
                            </div>
                        </div>
                        {sopsLoading ? (
                            <div style={{ padding: '48px', textAlign: 'center', color: theme.colors.text.secondary }}>Loading SOPs...</div>
                        ) : filteredSOPs.length === 0 ? (
                            <div style={{ padding: '48px', textAlign: 'center', color: theme.colors.text.secondary }}>
                                {sopSearch ? 'No SOPs match your search.' : 'No SOPs available.'}
                            </div>
                        ) : (
                            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                {filteredSOPs.map(sop => (
                                    <div key={sop.issue_type} style={{ border: `1px solid ${theme.colors.border.default}`, borderRadius: '8px', overflow: 'hidden' }}>
                                        <button
                                            onClick={() => setExpandedSOP(expandedSOP === sop.issue_type ? null : sop.issue_type)}
                                            style={{
                                                width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
                                                padding: '14px 16px', background: 'none', border: 'none', cursor: 'pointer',
                                                backgroundColor: expandedSOP === sop.issue_type ? `${theme.colors.primary.DEFAULT}08` : 'transparent',
                                            }}
                                        >
                                            <div style={{ display: 'flex', alignItems: 'center', gap: '12px', textAlign: 'left' }}>
                                                <span style={{
                                                    padding: '2px 8px', borderRadius: '4px', fontSize: '11px', fontWeight: 700,
                                                    backgroundColor: `${theme.colors.primary.DEFAULT}15`, color: theme.colors.primary.DEFAULT, fontFamily: 'monospace'
                                                }}>
                                                    {sop.issue_type}
                                                </span>
                                                <span style={{ fontSize: '14px', fontWeight: 600, color: theme.colors.text.primary }}>{sop.title}</span>
                                                <span style={{ fontSize: '12px', color: theme.colors.text.muted }}>{sop.steps.length} steps</span>
                                            </div>
                                            {expandedSOP === sop.issue_type
                                                ? <ChevronDown style={{ width: '16px', height: '16px', color: theme.colors.text.muted }} />
                                                : <ChevronRight style={{ width: '16px', height: '16px', color: theme.colors.text.muted }} />}
                                        </button>
                                        {expandedSOP === sop.issue_type && (
                                            <div style={{ padding: '16px', borderTop: `1px solid ${theme.colors.border.subtle}`, backgroundColor: theme.colors.background.primary }}>
                                                <ol style={{ margin: 0, paddingLeft: '20px', display: 'flex', flexDirection: 'column', gap: '8px' }}>
                                                    {sop.steps.map((step, i) => (
                                                        <li key={i} style={{ fontSize: '14px', color: theme.colors.text.secondary, lineHeight: '1.6' }}>{step}</li>
                                                    ))}
                                                </ol>
                                                {sop.references && sop.references.length > 0 && (
                                                    <div style={{ marginTop: '12px', paddingTop: '12px', borderTop: `1px solid ${theme.colors.border.subtle}` }}>
                                                        <span style={{ fontSize: '12px', fontWeight: 600, color: theme.colors.text.muted, textTransform: 'uppercase', letterSpacing: '0.05em' }}>References: </span>
                                                        {sop.references.map((ref, i) => (
                                                            <span key={i} style={{ fontSize: '12px', color: theme.colors.primary.DEFAULT, marginRight: '8px' }}>{ref}</span>
                                                        ))}
                                                    </div>
                                                )}
                                            </div>
                                        )}
                                    </div>
                                ))}
                            </div>
                        )}
                    </div>
                )}

                {/* Escalation Tab */}
                {activeTab === 'escalation' && (
                    <div style={{ backgroundColor: theme.colors.background.card, borderRadius: '12px', border: `1px solid ${theme.colors.border.default}`, padding: '32px' }}>
                        {escalationToast && (
                            <div style={{
                                position: 'fixed', top: '20px', right: '20px', zIndex: 3000,
                                padding: '12px 20px', borderRadius: '8px', fontSize: '14px', fontWeight: 500, color: '#fff',
                                backgroundColor: escalationToast.ok ? '#16a34a' : '#dc2626',
                                boxShadow: '0 4px 12px rgba(0,0,0,0.15)'
                            }}>
                                {escalationToast.msg}
                            </div>
                        )}
                        <div style={{ maxWidth: '640px' }}>
                            <h2 style={{ fontSize: '24px', fontWeight: 800, color: theme.colors.text.primary, marginBottom: '8px' }}>
                                Escalation Management
                            </h2>
                            <p style={{ color: theme.colors.text.secondary, fontSize: '15px', marginBottom: '32px', lineHeight: '1.6' }}>
                                Preview overdue or unresolved findings that qualify for escalation, then trigger the escalation workflow to notify stakeholders.
                            </p>
                            <button
                                onClick={handlePreviewEscalation}
                                disabled={escalationLoading}
                                style={{
                                    display: 'inline-flex', alignItems: 'center', gap: '8px',
                                    padding: '12px 24px', borderRadius: '8px', fontWeight: 700, fontSize: '15px',
                                    border: `1px solid ${theme.colors.primary.DEFAULT}`, backgroundColor: theme.colors.primary.DEFAULT,
                                    color: '#fff', cursor: escalationLoading ? 'wait' : 'pointer',
                                    opacity: escalationLoading ? 0.7 : 1
                                }}
                            >
                                <Siren style={{ width: '16px', height: '16px' }} />
                                {escalationLoading ? 'Loading Preview...' : 'Preview Escalation'}
                            </button>
                        </div>

                        {/* Escalation Modal */}
                        {showEscalationModal && escalationPreview && (
                            <div style={{
                                position: 'fixed', inset: 0, backgroundColor: 'rgba(15,23,42,0.4)',
                                display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 2000,
                                backdropFilter: 'blur(4px)'
                            }}>
                                <div style={{
                                    backgroundColor: '#fff', borderRadius: '12px', width: '90%', maxWidth: '640px',
                                    maxHeight: '80vh', display: 'flex', flexDirection: 'column',
                                    boxShadow: '0 20px 40px rgba(0,0,0,0.15)', border: '1px solid #e2e8f0'
                                }}>
                                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '20px 24px', borderBottom: '1px solid #e2e8f0' }}>
                                        <h3 style={{ margin: 0, fontSize: '18px', fontWeight: 700, color: '#0f172a' }}>
                                            Escalation Preview — {escalationPreview.total} finding(s)
                                        </h3>
                                        <button onClick={() => setShowEscalationModal(false)} style={{ background: 'none', border: 'none', cursor: 'pointer', color: '#94a3b8', fontSize: '22px' }}>
                                            <X style={{ width: '20px', height: '20px' }} />
                                        </button>
                                    </div>
                                    <div style={{ overflowY: 'auto', flex: 1, padding: '16px 24px' }}>
                                        {escalationPreview.findings.length === 0 ? (
                                            <div style={{ padding: '32px', textAlign: 'center', color: '#64748b' }}>No findings qualify for escalation.</div>
                                        ) : (
                                            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
                                                <thead>
                                                    <tr style={{ backgroundColor: '#f8fafc' }}>
                                                        <th style={{ padding: '10px 12px', textAlign: 'left', color: '#64748b', fontWeight: 700, fontSize: '11px', textTransform: 'uppercase' }}>Asset</th>
                                                        <th style={{ padding: '10px 12px', textAlign: 'left', color: '#64748b', fontWeight: 700, fontSize: '11px', textTransform: 'uppercase' }}>PII Type</th>
                                                        <th style={{ padding: '10px 12px', textAlign: 'left', color: '#64748b', fontWeight: 700, fontSize: '11px', textTransform: 'uppercase' }}>Risk</th>
                                                        <th style={{ padding: '10px 12px', textAlign: 'right', color: '#64748b', fontWeight: 700, fontSize: '11px', textTransform: 'uppercase' }}>Days Open</th>
                                                    </tr>
                                                </thead>
                                                <tbody>
                                                    {escalationPreview.findings.map(f => (
                                                        <tr key={f.id} style={{ borderBottom: '1px solid #f1f5f9' }}>
                                                            <td style={{ padding: '10px 12px', color: '#0f172a', fontWeight: 500 }}>{f.asset_name || '—'}</td>
                                                            <td style={{ padding: '10px 12px', color: '#475569' }}>{f.pii_type || '—'}</td>
                                                            <td style={{ padding: '10px 12px' }}>
                                                                <span style={{ padding: '2px 8px', borderRadius: '12px', fontSize: '11px', fontWeight: 700, backgroundColor: getRiskColor(f.risk_level || '') + '20', color: getRiskColor(f.risk_level || '') }}>
                                                                    {f.risk_level || '—'}
                                                                </span>
                                                            </td>
                                                            <td style={{ padding: '10px 12px', textAlign: 'right', fontFamily: 'monospace', color: '#475569' }}>{f.days_open ?? '—'}</td>
                                                        </tr>
                                                    ))}
                                                </tbody>
                                            </table>
                                        )}
                                    </div>
                                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '12px', padding: '16px 24px', borderTop: '1px solid #e2e8f0' }}>
                                        <button
                                            onClick={() => setShowEscalationModal(false)}
                                            disabled={escalationRunning}
                                            style={{ padding: '10px 20px', borderRadius: '8px', border: '1px solid #e2e8f0', backgroundColor: '#f1f5f9', color: '#475569', fontWeight: 600, cursor: 'pointer', fontSize: '14px' }}
                                        >
                                            Cancel
                                        </button>
                                        <button
                                            onClick={handleRunEscalation}
                                            disabled={escalationRunning || escalationPreview.total === 0}
                                            style={{
                                                padding: '10px 20px', borderRadius: '8px', fontWeight: 700, fontSize: '14px',
                                                border: 'none', backgroundColor: '#dc2626', color: '#fff',
                                                cursor: (escalationRunning || escalationPreview.total === 0) ? 'not-allowed' : 'pointer',
                                                opacity: (escalationRunning || escalationPreview.total === 0) ? 0.6 : 1,
                                                display: 'flex', alignItems: 'center', gap: '8px'
                                            }}
                                        >
                                            <Siren style={{ width: '14px', height: '14px' }} />
                                            {escalationRunning ? 'Running...' : 'Run Escalation'}
                                        </button>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                )}

            </div>
        </div>
    );
}

function StatCard({ title, value, color, icon }: any) {
    return (
        <div style={{
            backgroundColor: theme.colors.background.card,
            borderRadius: '12px',
            border: `1px solid ${theme.colors.border.default}`,
            padding: '20px',
            textAlign: 'center'
        }}>
            <div style={{ color, marginBottom: '8px' }}>{icon}</div>
            <div style={{ fontSize: '24px', fontWeight: 800, color, marginBottom: '4px' }}>{value}</div>
            <div style={{ fontSize: '12px', color: theme.colors.text.secondary, fontWeight: 600 }}>{title}</div>
        </div>
    );
}

const tableHeaderStyle: React.CSSProperties = {
    padding: '16px',
    textAlign: 'left',
    fontSize: '12px',
    fontWeight: 700,
    color: theme.colors.text.secondary,
    textTransform: 'uppercase',
    letterSpacing: '0.05em'
};

const tableCellStyle: React.CSSProperties = {
    padding: '16px',
    fontSize: '14px',
    color: theme.colors.text.primary
};

function getRiskColor(riskLevel: string) {
    switch (riskLevel.toLowerCase()) {
        case 'critical': return theme.colors.risk.critical;
        case 'high': return theme.colors.risk.high;
        case 'medium': return theme.colors.risk.medium;
        default: return theme.colors.risk.low;
    }
}

function getActionColor(action: string) {
    switch (action) {
        case 'DELETE': return '#DC2626'; // red
        case 'MASK': return '#F59E0B'; // orange
        case 'ENCRYPT': return '#059669'; // green
        default: return theme.colors.text.secondary;
    }
}
