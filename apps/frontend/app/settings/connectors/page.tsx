'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
    Plus, Trash2, Edit2, RefreshCw, CheckCircle, XCircle,
    AlertTriangle, Database, Plug, HelpCircle, Loader2
} from 'lucide-react';
import {
    getConnections,
    addConnection,
    deleteConnection,
    testConnection,
    type Connection,
    type ConnectionConfig,
} from '@/services/connections.api';
import { put } from '@/utils/api-client';

// ─── Types ────────────────────────────────────────────────────────────────────

type AttributeLevel = 'org' | 'person' | 'login';
type ConnectorStatus = 'active' | 'inactive' | 'error' | 'validating';

interface ConnectorRow extends Connection {
    attribute_level?: AttributeLevel;
    status?: ConnectorStatus;
    last_scan?: string;
}

const CONNECTOR_TYPES = [
    { value: 'postgresql', label: 'PostgreSQL', icon: '🐘' },
    { value: 'mysql', label: 'MySQL', icon: '🐬' },
    { value: 'mongodb', label: 'MongoDB', icon: '🍃' },
    { value: 's3', label: 'AWS S3', icon: '☁️' },
    { value: 'bigquery', label: 'Google BigQuery', icon: '📊' },
    { value: 'snowflake', label: 'Snowflake', icon: '❄️' },
    { value: 'redshift', label: 'Amazon Redshift', icon: '🔴' },
    { value: 'elasticsearch', label: 'Elasticsearch', icon: '🔍' },
    { value: 'redis', label: 'Redis', icon: '🔥' },
    { value: 'custom', label: 'Custom / Other', icon: '⚙️' },
];

const ATTRIBUTE_LEVEL_INFO: Record<AttributeLevel, { label: string; description: string }> = {
    org: {
        label: 'Org-wise',
        description: 'Scans are attributed to the entire organization. Use for shared infrastructure.',
    },
    person: {
        label: 'Person-wise',
        description: 'Scans are attributed to specific individuals or accounts in the data source.',
    },
    login: {
        label: 'Login-wise',
        description: 'Scans track individual login sessions. Best for auditing user-specific data access.',
    },
};

const DB_FIELDS: Record<string, { key: string; label: string; type?: string; placeholder?: string }[]> = {
    postgresql: [
        { key: 'host', label: 'Host', placeholder: 'localhost' },
        { key: 'user', label: 'Username', placeholder: 'postgres' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
        { key: 'database', label: 'Database', placeholder: 'mydb' },
        { key: 'port', label: 'Port', placeholder: '5432' },
    ],
    mysql: [
        { key: 'host', label: 'Host', placeholder: 'localhost' },
        { key: 'user', label: 'Username', placeholder: 'root' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
        { key: 'database', label: 'Database', placeholder: 'mydb' },
        { key: 'port', label: 'Port', placeholder: '3306' },
    ],
    mongodb: [
        { key: 'host', label: 'Connection URI', placeholder: 'mongodb://localhost:27017/mydb' },
        { key: 'database', label: 'Database', placeholder: 'mydb' },
    ],
    s3: [
        { key: 'region', label: 'Region', placeholder: 'us-east-1' },
        { key: 'bucket', label: 'Bucket Name', placeholder: 'my-data-bucket' },
        { key: 'access_key', label: 'Access Key ID', placeholder: 'AKIAIOSFODNN7EXAMPLE' },
        { key: 'secret_key', label: 'Secret Access Key', type: 'password', placeholder: '••••••••' },
    ],
    bigquery: [
        { key: 'project', label: 'Project ID', placeholder: 'my-gcp-project' },
        { key: 'dataset', label: 'Dataset', placeholder: 'my_dataset' },
        { key: 'credentials_json', label: 'Service Account JSON', placeholder: '{"type":"service_account",...}' },
    ],
    snowflake: [
        { key: 'account', label: 'Account', placeholder: 'xyz.us-east-1' },
        { key: 'user', label: 'Username', placeholder: 'SVCUSER' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
        { key: 'warehouse', label: 'Warehouse', placeholder: 'COMPUTE_WH' },
        { key: 'database', label: 'Database', placeholder: 'MY_DB' },
    ],
};

function getConfigFields(sourceType: string) {
    return DB_FIELDS[sourceType] ?? [
        { key: 'host', label: 'Host / Endpoint', placeholder: 'host' },
        { key: 'user', label: 'Username', placeholder: 'user' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
    ];
}

// ─── Status helpers ───────────────────────────────────────────────────────────

function statusBadge(status?: string) {
    switch (status) {
        case 'active':
        case 'valid':
            return (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-green-100 text-green-800 border border-green-200">
                    <CheckCircle className="w-3 h-3" /> Active
                </span>
            );
        case 'inactive':
            return (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-600 border border-slate-200">
                    <XCircle className="w-3 h-3" /> Inactive
                </span>
            );
        case 'error':
            return (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-red-100 text-red-800 border border-red-200">
                    <AlertTriangle className="w-3 h-3" /> Error
                </span>
            );
        default:
            return (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-500 border border-slate-200">
                    Unknown
                </span>
            );
    }
}

function formatDate(d?: string) {
    if (!d) return '—';
    try { return new Date(d).toLocaleDateString(); } catch { return d; }
}

// ─── Attribute Level Selector ─────────────────────────────────────────────────

function AttributeLevelSelector({
    value,
    onChange,
}: {
    value: AttributeLevel;
    onChange: (v: AttributeLevel) => void;
}) {
    const [showTooltip, setShowTooltip] = useState<AttributeLevel | null>(null);

    return (
        <div>
            <div className="flex items-center gap-2 mb-2">
                <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Attribute Level</label>
                <HelpCircle className="w-3.5 h-3.5 text-slate-400" />
            </div>
            <div className="flex gap-3">
                {(Object.keys(ATTRIBUTE_LEVEL_INFO) as AttributeLevel[]).map(level => {
                    const info = ATTRIBUTE_LEVEL_INFO[level];
                    const isSelected = value === level;
                    return (
                        <div key={level} className="relative flex-1">
                            <label
                                className={`flex flex-col gap-1 p-3 rounded-lg border-2 cursor-pointer transition-colors ${isSelected ? 'border-blue-500 bg-blue-50' : 'border-slate-200 bg-white hover:border-slate-300'}`}
                                onMouseEnter={() => setShowTooltip(level)}
                                onMouseLeave={() => setShowTooltip(null)}
                            >
                                <div className="flex items-center gap-2">
                                    <input
                                        type="radio"
                                        name="attribute_level"
                                        value={level}
                                        checked={isSelected}
                                        onChange={() => onChange(level)}
                                        className="w-3.5 h-3.5 text-blue-600"
                                    />
                                    <span className={`text-sm font-semibold ${isSelected ? 'text-blue-700' : 'text-slate-700'}`}>
                                        {info.label}
                                    </span>
                                </div>
                            </label>
                            {showTooltip === level && (
                                <div className="absolute bottom-full left-0 mb-1.5 z-10 w-52 px-3 py-2 bg-slate-900 text-white text-xs rounded-lg shadow-lg">
                                    {info.description}
                                    <div className="absolute top-full left-4 border-4 border-transparent border-t-slate-900" />
                                </div>
                            )}
                        </div>
                    );
                })}
            </div>
        </div>
    );
}

// ─── Add/Edit Modal ───────────────────────────────────────────────────────────

interface ConnectorFormState {
    source_type: string;
    profile_name: string;
    attribute_level: AttributeLevel;
    config: Record<string, string>;
}

function ConnectorModal({
    initial,
    onSave,
    onClose,
}: {
    initial?: ConnectorRow;
    onSave: (data: ConnectorFormState) => Promise<void>;
    onClose: () => void;
}) {
    const [form, setForm] = useState<ConnectorFormState>({
        source_type: initial?.source_type ?? '',
        profile_name: initial?.profile_name ?? '',
        attribute_level: initial?.attribute_level ?? 'org',
        config: {},
    });
    const [saving, setSaving] = useState(false);
    const [testing, setTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ ok: boolean; message: string } | null>(null);

    const configFields = form.source_type ? getConfigFields(form.source_type) : [];

    const setConfig = (key: string, value: string) => {
        setForm(f => ({ ...f, config: { ...f.config, [key]: value } }));
        setTestResult(null);
    };

    const handleTestConnection = async () => {
        setTesting(true);
        setTestResult(null);
        try {
            await testConnection({
                source_type: form.source_type,
                profile_name: form.profile_name,
                config: form.config,
            } as ConnectionConfig);
            setTestResult({ ok: true, message: 'Connection successful.' });
        } catch (e: any) {
            const msg = e?.response?.data?.error ?? e?.message ?? 'Connection failed.';
            setTestResult({ ok: false, message: msg });
        } finally {
            setTesting(false);
        }
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        setSaving(true);
        try { await onSave(form); }
        finally { setSaving(false); }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={onClose}>
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
                <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between sticky top-0 bg-white z-10">
                    <h2 className="text-lg font-semibold text-slate-900">{initial ? 'Edit Connector' : 'Add Connector'}</h2>
                    <button onClick={onClose} className="text-slate-400 hover:text-slate-600"><XCircle className="w-5 h-5" /></button>
                </div>

                <form onSubmit={handleSubmit} className="p-6 space-y-5">
                    {/* Connector type */}
                    <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Connector Type <span className="text-red-500">*</span></label>
                        <div className="grid grid-cols-2 gap-2 max-h-52 overflow-y-auto pr-1">
                            {CONNECTOR_TYPES.map(ct => (
                                <label
                                    key={ct.value}
                                    className={`flex items-center gap-2 px-3 py-2.5 rounded-lg border cursor-pointer transition-colors ${form.source_type === ct.value ? 'border-blue-500 bg-blue-50' : 'border-slate-200 hover:border-slate-300'}`}
                                >
                                    <input
                                        type="radio"
                                        name="source_type"
                                        value={ct.value}
                                        required
                                        checked={form.source_type === ct.value}
                                        onChange={() => setForm(f => ({ ...f, source_type: ct.value, config: {} }))}
                                        className="sr-only"
                                    />
                                    <span className="text-lg leading-none">{ct.icon}</span>
                                    <span className={`text-sm font-medium ${form.source_type === ct.value ? 'text-blue-700' : 'text-slate-700'}`}>{ct.label}</span>
                                </label>
                            ))}
                        </div>
                    </div>

                    {/* Profile name */}
                    <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Profile Name <span className="text-red-500">*</span></label>
                        <input
                            required
                            value={form.profile_name}
                            onChange={e => setForm(f => ({ ...f, profile_name: e.target.value }))}
                            placeholder="production-db"
                            className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900"
                        />
                        <p className="text-xs text-slate-400">Unique name to identify this connection.</p>
                    </div>

                    {/* Attribute level */}
                    <AttributeLevelSelector
                        value={form.attribute_level}
                        onChange={v => setForm(f => ({ ...f, attribute_level: v }))}
                    />

                    {/* Dynamic config fields */}
                    {form.source_type && configFields.length > 0 && (
                        <div className="space-y-3">
                            <div className="text-xs font-semibold text-slate-600 uppercase tracking-wide border-t border-slate-100 pt-3">Connection Config</div>
                            {configFields.map(field => (
                                <div key={field.key} className="space-y-1.5">
                                    <label className="text-sm font-medium text-slate-700">{field.label}</label>
                                    <input
                                        type={field.type ?? 'text'}
                                        value={form.config[field.key] ?? ''}
                                        onChange={e => setConfig(field.key, e.target.value)}
                                        placeholder={field.placeholder}
                                        className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900"
                                    />
                                </div>
                            ))}
                        </div>
                    )}

                    {/* Test connection */}
                    {form.source_type && (
                        <div className="space-y-2">
                            <button
                                type="button"
                                onClick={handleTestConnection}
                                disabled={testing}
                                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 disabled:opacity-50"
                            >
                                {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plug className="w-4 h-4" />}
                                {testing ? 'Testing...' : 'Test Connection'}
                            </button>
                            {testResult && (
                                <div className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm border ${testResult.ok ? 'bg-green-50 border-green-200 text-green-800' : 'bg-red-50 border-red-200 text-red-800'}`}>
                                    {testResult.ok ? <CheckCircle className="w-4 h-4 shrink-0" /> : <XCircle className="w-4 h-4 shrink-0" />}
                                    {testResult.message}
                                </div>
                            )}
                        </div>
                    )}

                    <div className="flex justify-end gap-3 pt-2 border-t border-slate-100">
                        <button type="button" onClick={onClose} className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={saving}
                            className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {saving && <RefreshCw className="w-4 h-4 animate-spin" />}
                            {saving ? 'Saving...' : initial ? 'Save Changes' : 'Add Connector'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function ConnectorsSettingsPage() {
    const [connectors, setConnectors] = useState<ConnectorRow[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [showAdd, setShowAdd] = useState(false);
    const [editTarget, setEditTarget] = useState<ConnectorRow | null>(null);
    const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
    const [testingId, setTestingId] = useState<string | null>(null);
    const [testResults, setTestResults] = useState<Record<string, { ok: boolean; message: string }>>({});

    const loadConnectors = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const res = await getConnections();
            const list = res?.connections ?? [];
            // Augment with display status
            const rows: ConnectorRow[] = list.map(c => ({
                ...c,
                status: (c.validation_status === 'valid' ? 'active' : c.validation_status === 'invalid' ? 'error' : 'inactive') as ConnectorStatus,
                attribute_level: 'org' as AttributeLevel,
            }));
            setConnectors(rows);
        } catch {
            setError('Failed to load connectors.');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => { loadConnectors(); }, [loadConnectors]);

    const handleAdd = async (formData: ConnectorFormState) => {
        await addConnection({
            source_type: formData.source_type,
            profile_name: formData.profile_name,
            config: formData.config,
        } as ConnectionConfig);
        setShowAdd(false);
        await loadConnectors();
    };

    const handleEdit = async (formData: ConnectorFormState) => {
        if (!editTarget?.id) return;
        // PUT /api/v1/connections/:id
        await put<unknown>(`/connections/${editTarget.id}`, {
            source_type: formData.source_type,
            profile_name: formData.profile_name,
            attribute_level: formData.attribute_level,
            config: formData.config,
        });
        setEditTarget(null);
        await loadConnectors();
    };

    const handleDelete = async (id: string) => {
        await deleteConnection(id);
        setDeleteConfirmId(null);
        setConnectors(prev => prev.filter(c => c.id !== id));
    };

    const handleTestConnection = async (c: ConnectorRow) => {
        setTestingId(c.id);
        try {
            await testConnection({
                source_type: c.source_type,
                profile_name: c.profile_name,
                config: {},
            } as ConnectionConfig);
            setTestResults(prev => ({ ...prev, [c.id]: { ok: true, message: 'Connection successful.' } }));
        } catch (e: any) {
            const msg = e?.response?.data?.error ?? e?.message ?? 'Connection failed.';
            setTestResults(prev => ({ ...prev, [c.id]: { ok: false, message: msg } }));
        } finally {
            setTestingId(null);
        }
    };

    const connectorTypeLabel = (type: string) => {
        return CONNECTOR_TYPES.find(ct => ct.value === type)?.label ?? type;
    };

    const connectorTypeIcon = (type: string) => {
        return CONNECTOR_TYPES.find(ct => ct.value === type)?.icon ?? '⚙️';
    };

    return (
        <div className="p-8 space-y-6 bg-white min-h-screen">
            {/* Header */}
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Data Connectors</h1>
                    <p className="text-slate-500 mt-1 text-sm">
                        Configure connections to data sources for PII scanning. Each connector defines a source type, credentials, and attribution scope.
                    </p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                    <button
                        onClick={loadConnectors}
                        className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50"
                        title="Refresh connectors"
                    >
                        <RefreshCw className="w-4 h-4" />
                        Refresh
                    </button>
                    <button
                        onClick={() => setShowAdd(true)}
                        className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                    >
                        <Plus className="w-4 h-4" />
                        Add Connector
                    </button>
                </div>
            </div>

            {/* Attribute level explainer */}
            <div className="grid grid-cols-3 gap-4 text-sm">
                {(Object.entries(ATTRIBUTE_LEVEL_INFO) as [AttributeLevel, typeof ATTRIBUTE_LEVEL_INFO['org']][]).map(([key, info]) => (
                    <div key={key} className="p-4 rounded-lg border border-slate-200 bg-slate-50">
                        <div className="font-semibold text-slate-700 mb-1">{info.label}</div>
                        <div className="text-xs text-slate-500 leading-relaxed">{info.description}</div>
                    </div>
                ))}
            </div>

            {/* Table */}
            {loading ? (
                <div className="space-y-3">
                    {[...Array(4)].map((_, i) => (
                        <div key={i} className="h-16 bg-slate-100 rounded-lg animate-pulse" />
                    ))}
                </div>
            ) : error ? (
                <div className="p-6 text-center text-red-600 bg-red-50 rounded-lg border border-red-100">{error}</div>
            ) : connectors.length === 0 ? (
                <div className="p-12 text-center bg-slate-50 rounded-lg border border-slate-100 border-dashed">
                    <Database className="w-10 h-10 text-slate-300 mx-auto mb-3" />
                    <p className="text-slate-500 font-medium">No connectors configured</p>
                    <p className="text-slate-400 text-sm mt-1">Add a data source to start scanning for PII.</p>
                    <button
                        onClick={() => setShowAdd(true)}
                        className="mt-4 flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 mx-auto"
                    >
                        <Plus className="w-4 h-4" />
                        Add Connector
                    </button>
                </div>
            ) : (
                <div className="border border-slate-200 rounded-lg overflow-hidden">
                    <table className="w-full text-sm">
                        <thead className="bg-slate-50 border-b border-slate-200">
                            <tr>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Name</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Type</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Status</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Attribute Level</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Last Updated</th>
                                <th className="text-right px-4 py-3 font-medium text-slate-600">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {connectors.map(c => (
                                <React.Fragment key={c.id}>
                                    <tr className="hover:bg-slate-50 transition-colors">
                                        <td className="px-4 py-3">
                                            <div className="font-medium text-slate-900">{c.profile_name}</div>
                                            <div className="text-xs text-slate-400 font-mono">{c.id}</div>
                                        </td>
                                        <td className="px-4 py-3">
                                            <div className="flex items-center gap-2">
                                                <span className="text-base leading-none">{connectorTypeIcon(c.source_type)}</span>
                                                <span className="text-slate-700">{connectorTypeLabel(c.source_type)}</span>
                                            </div>
                                        </td>
                                        <td className="px-4 py-3">{statusBadge(c.status ?? c.validation_status)}</td>
                                        <td className="px-4 py-3">
                                            <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold bg-blue-50 text-blue-700 border border-blue-100 capitalize">
                                                {ATTRIBUTE_LEVEL_INFO[c.attribute_level ?? 'org']?.label ?? c.attribute_level ?? 'Org-wise'}
                                            </span>
                                        </td>
                                        <td className="px-4 py-3 text-slate-500 text-xs">
                                            {formatDate(c.updated_at)}
                                        </td>
                                        <td className="px-4 py-3">
                                            <div className="flex items-center justify-end gap-1">
                                                <button
                                                    onClick={() => handleTestConnection(c)}
                                                    disabled={testingId === c.id}
                                                    className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium text-slate-600 bg-white border border-slate-200 rounded-md hover:bg-slate-50 disabled:opacity-50"
                                                    title="Test connection"
                                                >
                                                    {testingId === c.id ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Plug className="w-3.5 h-3.5" />}
                                                    Test
                                                </button>
                                                <button
                                                    onClick={() => setEditTarget(c)}
                                                    className="p-1.5 text-slate-400 hover:text-slate-700 hover:bg-slate-100 rounded-md"
                                                    title="Edit connector"
                                                >
                                                    <Edit2 className="w-4 h-4" />
                                                </button>
                                                <button
                                                    onClick={() => setDeleteConfirmId(c.id)}
                                                    className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md"
                                                    title="Delete connector"
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            </div>
                                        </td>
                                    </tr>

                                    {/* Inline test result */}
                                    {testResults[c.id] && (
                                        <tr>
                                            <td colSpan={6} className={`px-4 py-2 ${testResults[c.id].ok ? 'bg-green-50 border-t border-green-100' : 'bg-red-50 border-t border-red-100'}`}>
                                                <div className={`flex items-center gap-2 text-xs font-medium ${testResults[c.id].ok ? 'text-green-700' : 'text-red-700'}`}>
                                                    {testResults[c.id].ok ? <CheckCircle className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
                                                    {testResults[c.id].message}
                                                    <button
                                                        onClick={() => setTestResults(prev => { const n = { ...prev }; delete n[c.id]; return n; })}
                                                        className="ml-auto text-slate-400 hover:text-slate-600"
                                                    >
                                                        <XCircle className="w-3.5 h-3.5" />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    )}

                                    {/* Delete confirm inline */}
                                    {deleteConfirmId === c.id && (
                                        <tr>
                                            <td colSpan={6} className="px-4 py-3 bg-red-50 border-t border-red-100">
                                                <div className="flex items-center gap-4 text-sm">
                                                    <AlertTriangle className="w-4 h-4 text-red-500 shrink-0" />
                                                    <span className="text-red-700 font-medium flex-1">
                                                        Delete connector <strong>{c.profile_name}</strong>? All associated scan data references will be preserved.
                                                    </span>
                                                    <button
                                                        onClick={() => setDeleteConfirmId(null)}
                                                        className="px-3 py-1.5 text-sm font-medium text-slate-600 bg-white border border-slate-200 rounded-lg hover:bg-slate-50"
                                                    >
                                                        Cancel
                                                    </button>
                                                    <button
                                                        onClick={() => handleDelete(c.id)}
                                                        className="px-3 py-1.5 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700"
                                                    >
                                                        Delete
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

            {/* Add Modal */}
            {showAdd && <ConnectorModal onSave={handleAdd} onClose={() => setShowAdd(false)} />}

            {/* Edit Modal */}
            {editTarget && <ConnectorModal initial={editTarget} onSave={handleEdit} onClose={() => setEditTarget(null)} />}
        </div>
    );
}
