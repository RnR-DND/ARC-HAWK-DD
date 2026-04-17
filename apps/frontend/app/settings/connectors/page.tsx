'use client';

import React, { useState, useEffect, useCallback } from 'react';
import {
    Plus, Trash2, Edit2, RefreshCw, CheckCircle, XCircle,
    AlertTriangle, Database, Plug, Loader2, Server, Cloud, HardDrive, FileText,
} from 'lucide-react';
import {
    getConnections,
    addConnection,
    deleteConnection,
    testConnection,
    getAvailableTypes,
    type Connection,
    type ConnectionConfig,
    type AvailableSourceType,
} from '@/services/connections.api';
import { put, post } from '@/utils/api-client';

// ─── Types ────────────────────────────────────────────────────────────────────

type ConnectorStatus = 'active' | 'inactive' | 'error';

interface ConnectorRow extends Connection {
    status?: ConnectorStatus;
}

interface ConnectorType {
    value: string;
    label: string;
    icon: React.ReactNode;
    category: string;
}

// Full list of all 36 supported connector types — used as fallback when API unavailable
const FALLBACK_CONNECTOR_TYPES: ConnectorType[] = [
    // Databases
    { value: 'postgresql',     label: 'PostgreSQL',         icon: <Database className="w-4 h-4" />,  category: 'Databases' },
    { value: 'mysql',          label: 'MySQL',              icon: <Database className="w-4 h-4" />,  category: 'Databases' },
    { value: 'mongodb',        label: 'MongoDB',            icon: <Database className="w-4 h-4" />,  category: 'Databases' },
    { value: 'redis',          label: 'Redis',              icon: <Server className="w-4 h-4" />,    category: 'Databases' },
    { value: 'sqlite',         label: 'SQLite',             icon: <Database className="w-4 h-4" />,  category: 'Databases' },
    { value: 'mssql',          label: 'SQL Server',         icon: <Database className="w-4 h-4" />,  category: 'Databases' },
    { value: 'couchdb',        label: 'CouchDB',            icon: <Database className="w-4 h-4" />,  category: 'Databases' },
    { value: 'firebase',       label: 'Firebase',           icon: <Cloud className="w-4 h-4" />,     category: 'Databases' },
    { value: 'oracle',         label: 'Oracle',             icon: <Database className="w-4 h-4" />,  category: 'Databases' },
    // Cloud
    { value: 's3',             label: 'AWS S3',             icon: <Cloud className="w-4 h-4" />,     category: 'Cloud' },
    { value: 'gcs',            label: 'Google Cloud Storage', icon: <Cloud className="w-4 h-4" />,   category: 'Cloud' },
    { value: 'azure_blob',     label: 'Azure Blob',         icon: <Cloud className="w-4 h-4" />,     category: 'Cloud' },
    { value: 'gdrive',         label: 'Google Drive',       icon: <Cloud className="w-4 h-4" />,     category: 'Cloud' },
    { value: 'gdrive_workspace', label: 'Google Workspace', icon: <Cloud className="w-4 h-4" />,     category: 'Cloud' },
    // Warehouses
    { value: 'bigquery',       label: 'BigQuery',           icon: <Server className="w-4 h-4" />,    category: 'Warehouses' },
    { value: 'snowflake',      label: 'Snowflake',          icon: <Server className="w-4 h-4" />,    category: 'Warehouses' },
    { value: 'redshift',       label: 'Redshift',           icon: <Server className="w-4 h-4" />,    category: 'Warehouses' },
    // Queues
    { value: 'kafka',          label: 'Kafka',              icon: <Server className="w-4 h-4" />,    category: 'Queues' },
    { value: 'kinesis',        label: 'AWS Kinesis',        icon: <Server className="w-4 h-4" />,    category: 'Queues' },
    // Files
    { value: 'filesystem',     label: 'Filesystem',         icon: <HardDrive className="w-4 h-4" />, category: 'Files' },
    { value: 'text',           label: 'Text Files',         icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'csv_excel',      label: 'CSV / Excel',        icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'html',           label: 'HTML Files',         icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'pdf',            label: 'PDF',                icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'docx',           label: 'Word (DOCX)',        icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'pptx',           label: 'PowerPoint (PPTX)', icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'email',          label: 'Email (EML)',        icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'avro',           label: 'Avro',               icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'parquet',        label: 'Parquet',            icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'orc',            label: 'ORC',                icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    { value: 'scanned_images', label: 'Scanned Images',     icon: <FileText className="w-4 h-4" />,  category: 'Files' },
    // SaaS
    { value: 'slack',          label: 'Slack',              icon: <Plug className="w-4 h-4" />,      category: 'SaaS' },
    { value: 'jira',           label: 'Jira',               icon: <Plug className="w-4 h-4" />,      category: 'SaaS' },
    { value: 'salesforce',     label: 'Salesforce',         icon: <Plug className="w-4 h-4" />,      category: 'SaaS' },
    { value: 'hubspot',        label: 'HubSpot',            icon: <Plug className="w-4 h-4" />,      category: 'SaaS' },
    { value: 'ms_teams',       label: 'Microsoft Teams',    icon: <Plug className="w-4 h-4" />,      category: 'SaaS' },
];

function iconForType(iconHint: string, category: string): React.ReactNode {
    const hint = (iconHint || '').toLowerCase();
    const cat = (category || '').toLowerCase();
    if (hint === 'database' || cat === 'databases') return <Database className="w-4 h-4" />;
    if (hint === 'cloud' || cat === 'cloud') return <Cloud className="w-4 h-4" />;
    if (hint === 'server' || cat === 'warehouses' || cat === 'queues') return <Server className="w-4 h-4" />;
    if (hint === 'hard-drive' || cat === 'files') return <HardDrive className="w-4 h-4" />;
    if (cat === 'saas' || cat === 'apps') return <Plug className="w-4 h-4" />;
    return <Database className="w-4 h-4" />;
}

function mapApiTypes(apiTypes: AvailableSourceType[]): ConnectorType[] {
    return apiTypes.map(t => ({
        value: t.source_type,
        label: t.display_name,
        icon: iconForType(t.icon, t.category),
        // Normalize category capitalization
        category: t.category.charAt(0).toUpperCase() + t.category.slice(1),
    }));
}

const DB_FIELDS: Record<string, { key: string; label: string; type?: string; placeholder?: string }[]> = {
    postgresql: [
        { key: 'host',     label: 'Host',     placeholder: 'db.example.com' },
        { key: 'port',     label: 'Port',     placeholder: '5432' },
        { key: 'database', label: 'Database', placeholder: 'mydb' },
        { key: 'user',     label: 'Username', placeholder: 'postgres' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
    ],
    mysql: [
        { key: 'host',     label: 'Host',     placeholder: 'db.example.com' },
        { key: 'port',     label: 'Port',     placeholder: '3306' },
        { key: 'database', label: 'Database', placeholder: 'mydb' },
        { key: 'user',     label: 'Username', placeholder: 'root' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
    ],
    mongodb: [
        { key: 'host',     label: 'Connection URI', placeholder: 'mongodb+srv://user:pass@cluster.mongodb.net/db' },
        { key: 'database', label: 'Database',        placeholder: 'mydb' },
    ],
    redis: [
        { key: 'host',     label: 'Host',     placeholder: 'redis.example.com' },
        { key: 'port',     label: 'Port',     placeholder: '6379' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
    ],
    sqlite: [
        { key: 'path', label: 'Database File Path', placeholder: '/data/mydb.sqlite' },
    ],
    mssql: [
        { key: 'host',     label: 'Host',     placeholder: 'mssql.example.com' },
        { key: 'port',     label: 'Port',     placeholder: '1433' },
        { key: 'database', label: 'Database', placeholder: 'mydb' },
        { key: 'user',     label: 'Username', placeholder: 'sa' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
    ],
    oracle: [
        { key: 'host',     label: 'Host',           placeholder: 'oracle.example.com' },
        { key: 'port',     label: 'Port',           placeholder: '1521' },
        { key: 'service',  label: 'Service Name',   placeholder: 'ORCL' },
        { key: 'user',     label: 'Username',       placeholder: 'system' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
    ],
    couchdb: [
        { key: 'host',     label: 'Host',     placeholder: 'couchdb.example.com' },
        { key: 'port',     label: 'Port',     placeholder: '5984' },
        { key: 'user',     label: 'Username', placeholder: 'admin' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
        { key: 'database', label: 'Database', placeholder: 'mydb' },
    ],
    firebase: [
        { key: 'credentials_file', label: 'Service Account JSON Path', placeholder: '/path/to/serviceAccount.json' },
        { key: 'storage_bucket',   label: 'Storage Bucket',            placeholder: 'my-project.appspot.com' },
    ],
    s3: [
        { key: 'region',     label: 'Region',          placeholder: 'us-east-1' },
        { key: 'bucket',     label: 'Bucket Name',     placeholder: 'my-data-bucket' },
        { key: 'access_key', label: 'Access Key ID',   placeholder: 'AKIAIOSFODNN7EXAMPLE' },
        { key: 'secret_key', label: 'Secret Key', type: 'password', placeholder: '••••••••' },
    ],
    gcs: [
        { key: 'bucket',           label: 'Bucket Name',          placeholder: 'my-gcs-bucket' },
        { key: 'project',          label: 'Project ID',           placeholder: 'my-gcp-project' },
        { key: 'credentials_json', label: 'Service Account JSON', placeholder: '{"type":"service_account",...}' },
    ],
    azure_blob: [
        { key: 'account_name', label: 'Storage Account Name', placeholder: 'mystorageaccount' },
        { key: 'account_key',  label: 'Account Key', type: 'password', placeholder: 'base64key...' },
        { key: 'container',    label: 'Container Name',       placeholder: 'my-container' },
    ],
    gdrive: [
        { key: 'credentials_file', label: 'OAuth Credentials JSON', placeholder: '/path/to/credentials.json' },
        { key: 'token_file',       label: 'Token File',             placeholder: '/path/to/token.json' },
    ],
    gdrive_workspace: [
        { key: 'credentials_json', label: 'Service Account JSON', placeholder: '{"type":"service_account",...}' },
        { key: 'admin_email',      label: 'Admin Email',           placeholder: 'admin@domain.com' },
    ],
    bigquery: [
        { key: 'project',          label: 'Project ID',           placeholder: 'my-gcp-project' },
        { key: 'dataset',          label: 'Dataset (optional)',    placeholder: 'my_dataset' },
        { key: 'credentials_json', label: 'Service Account JSON', placeholder: '{"type":"service_account",...}' },
    ],
    snowflake: [
        { key: 'account',   label: 'Account',   placeholder: 'xy12345.us-east-1' },
        { key: 'warehouse', label: 'Warehouse', placeholder: 'COMPUTE_WH' },
        { key: 'database',  label: 'Database',  placeholder: 'MY_DB' },
        { key: 'schema',    label: 'Schema',    placeholder: 'PUBLIC' },
        { key: 'user',      label: 'Username',  placeholder: 'myuser' },
        { key: 'password',  label: 'Password', type: 'password', placeholder: '••••••••' },
    ],
    redshift: [
        { key: 'host',     label: 'Host',     placeholder: 'cluster.region.redshift.amazonaws.com' },
        { key: 'port',     label: 'Port',     placeholder: '5439' },
        { key: 'database', label: 'Database', placeholder: 'mydb' },
        { key: 'user',     label: 'Username', placeholder: 'awsuser' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
    ],
    kafka: [
        { key: 'brokers', label: 'Brokers (comma-separated)', placeholder: 'broker1:9092,broker2:9092' },
        { key: 'topic',   label: 'Topic',                     placeholder: 'my-topic' },
    ],
    kinesis: [
        { key: 'region',      label: 'Region',      placeholder: 'us-east-1' },
        { key: 'stream_name', label: 'Stream Name', placeholder: 'my-stream' },
        { key: 'access_key',  label: 'Access Key ID', placeholder: 'AKIAIOSFODNN7EXAMPLE' },
        { key: 'secret_key',  label: 'Secret Key', type: 'password', placeholder: '••••••••' },
    ],
    filesystem: [
        { key: 'path', label: 'Root Path', placeholder: '/data/uploads' },
    ],
    // Legacy alias
    fs: [
        { key: 'path', label: 'Root Path', placeholder: '/data/uploads' },
    ],
    text:          [{ key: 'path', label: 'File Path', placeholder: '/data/file.txt' }],
    csv_excel:     [{ key: 'path', label: 'File Path', placeholder: '/data/file.csv' }],
    html:          [{ key: 'path', label: 'File Path', placeholder: '/data/file.html' }],
    pdf:           [{ key: 'path', label: 'File Path', placeholder: '/data/file.pdf' }],
    docx:          [{ key: 'path', label: 'File Path', placeholder: '/data/file.docx' }],
    pptx:          [{ key: 'path', label: 'File Path', placeholder: '/data/file.pptx' }],
    email:         [{ key: 'path', label: 'File / Directory Path', placeholder: '/data/emails/' }],
    avro:          [{ key: 'path', label: 'File Path', placeholder: '/data/file.avro' }],
    parquet:       [{ key: 'path', label: 'File Path', placeholder: '/data/file.parquet' }],
    orc:           [{ key: 'path', label: 'File Path', placeholder: '/data/file.orc' }],
    scanned_images:[{ key: 'path', label: 'File / Directory Path', placeholder: '/data/scans/' }],
    slack: [
        { key: 'token', label: 'Bot Token', placeholder: 'xoxb-...' },
    ],
    jira: [
        { key: 'url',   label: 'Site URL', placeholder: 'https://yoursite.atlassian.net' },
        { key: 'user',  label: 'Email',    placeholder: 'you@example.com' },
        { key: 'token', label: 'API Token', type: 'password', placeholder: '••••••••' },
    ],
    salesforce: [
        { key: 'instance_url',    label: 'Instance URL',    placeholder: 'https://yourorg.salesforce.com' },
        { key: 'username',        label: 'Username',        placeholder: 'user@example.com' },
        { key: 'password',        label: 'Password', type: 'password', placeholder: '••••••••' },
        { key: 'security_token',  label: 'Security Token', type: 'password', placeholder: 'token...' },
    ],
    hubspot: [
        { key: 'api_key', label: 'API Key / Private App Token', type: 'password', placeholder: 'pat-na1-...' },
    ],
    ms_teams: [
        { key: 'tenant_id',     label: 'Tenant ID',     placeholder: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx' },
        { key: 'client_id',     label: 'Client ID',     placeholder: 'xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx' },
        { key: 'client_secret', label: 'Client Secret', type: 'password', placeholder: '••••••••' },
    ],
};

function getConfigFields(sourceType: string) {
    return DB_FIELDS[sourceType] ?? [
        { key: 'host',     label: 'Host / Endpoint',  placeholder: 'host' },
        { key: 'user',     label: 'Username',          placeholder: 'user' },
        { key: 'password', label: 'Password', type: 'password', placeholder: '••••••••' },
    ];
}

function statusBadge(status?: string) {
    switch (status) {
        case 'active':
        case 'valid':
            return (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-green-100 text-green-800 border border-green-200">
                    <CheckCircle className="w-3 h-3" /> Active
                </span>
            );
        case 'error':
        case 'invalid':
            return (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-red-100 text-red-800 border border-red-200">
                    <AlertTriangle className="w-3 h-3" /> Error
                </span>
            );
        default:
            return (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold bg-slate-100 text-slate-500 border border-slate-200">
                    Pending
                </span>
            );
    }
}

function formatDate(d?: string) {
    if (!d) return '—';
    try { return new Date(d).toLocaleDateString(); } catch { return d; }
}

// ─── Add / Edit Modal ─────────────────────────────────────────────────────────

interface ConnectorFormState {
    source_type: string;
    profile_name: string;
    config: Record<string, string>;
}

function ConnectorModal({
    initial,
    connectorTypes,
    onSave,
    onClose,
}: {
    initial?: ConnectorRow;
    connectorTypes: ConnectorType[];
    onSave: (data: ConnectorFormState) => Promise<void>;
    onClose: () => void;
}) {
    const [form, setForm] = useState<ConnectorFormState>({
        source_type: initial?.source_type ?? '',
        profile_name: initial?.profile_name ?? '',
        config: {},
    });
    const [saving, setSaving] = useState(false);
    const [testing, setTesting] = useState(false);
    const [testResult, setTestResult] = useState<{ ok: boolean; message: string } | null>(null);

    const configFields = form.source_type ? getConfigFields(form.source_type) : [];
    const categories = [...new Set(connectorTypes.map(c => c.category))];

    const setConfig = (key: string, value: string) => {
        setForm(f => ({ ...f, config: { ...f.config, [key]: value } }));
        setTestResult(null);
    };

    const handleTest = async () => {
        setTesting(true);
        setTestResult(null);
        try {
            await testConnection({ source_type: form.source_type, profile_name: form.profile_name, config: form.config } as ConnectionConfig);
            setTestResult({ ok: true, message: 'Connection successful.' });
        } catch (e: any) {
            setTestResult({ ok: false, message: e?.response?.data?.error ?? e?.message ?? 'Connection failed.' });
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
                    <button onClick={onClose} className="text-slate-400 hover:text-slate-600">
                        <XCircle className="w-5 h-5" />
                    </button>
                </div>

                <form onSubmit={handleSubmit} className="p-6 space-y-5">
                    {/* Source type picker */}
                    <div className="space-y-2">
                        <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">
                            Source Type <span className="text-red-500">*</span>
                        </label>
                        {categories.map(cat => (
                            <div key={cat}>
                                <div className="text-[11px] font-semibold text-slate-400 uppercase tracking-wider mb-1.5">{cat}</div>
                                <div className="grid grid-cols-3 gap-2 mb-3">
                                    {connectorTypes.filter(c => c.category === cat).map(ct => (
                                        <label
                                            key={ct.value}
                                            className={`flex items-center gap-2 px-3 py-2 rounded-lg border cursor-pointer transition-colors text-sm ${
                                                form.source_type === ct.value
                                                    ? 'border-blue-500 bg-blue-50 text-blue-700 font-medium'
                                                    : 'border-slate-200 hover:border-slate-300 text-slate-700'
                                            }`}
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
                                            <span className={form.source_type === ct.value ? 'text-blue-600' : 'text-slate-400'}>
                                                {ct.icon}
                                            </span>
                                            {ct.label}
                                        </label>
                                    ))}
                                </div>
                            </div>
                        ))}
                    </div>

                    {/* Profile name */}
                    <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">
                            Profile Name <span className="text-red-500">*</span>
                        </label>
                        <input
                            required
                            value={form.profile_name}
                            onChange={e => setForm(f => ({ ...f, profile_name: e.target.value }))}
                            placeholder="e.g. production-db"
                            className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900"
                        />
                        <p className="text-xs text-slate-400">Used to identify this connection when configuring scans.</p>
                    </div>

                    {/* Config fields */}
                    {form.source_type && configFields.length > 0 && (
                        <div className="space-y-3 border-t border-slate-100 pt-4">
                            <div className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Connection Details</div>
                            {configFields.map(field => (
                                <div key={field.key} className="space-y-1">
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
                                onClick={handleTest}
                                disabled={testing}
                                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 disabled:opacity-50"
                            >
                                {testing ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plug className="w-4 h-4" />}
                                {testing ? 'Testing…' : 'Test Connection'}
                            </button>
                            {testResult && (
                                <div className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm border ${
                                    testResult.ok ? 'bg-green-50 border-green-200 text-green-800' : 'bg-red-50 border-red-200 text-red-800'
                                }`}>
                                    {testResult.ok ? <CheckCircle className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
                                    {testResult.message}
                                </div>
                            )}
                        </div>
                    )}

                    <div className="flex justify-end gap-3 pt-2 border-t border-slate-100">
                        <button type="button" onClick={onClose} className="px-4 py-2 text-sm text-slate-600 hover:text-slate-900">
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={saving}
                            className="flex items-center gap-2 px-5 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                        >
                            {saving && <RefreshCw className="w-4 h-4 animate-spin" />}
                            {saving ? 'Saving…' : initial ? 'Save Changes' : 'Add Connector'}
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
    const [availableTypes, setAvailableTypes] = useState<ConnectorType[]>(FALLBACK_CONNECTOR_TYPES);

    // Fetch available connector types from backend; fall back to static list on error
    useEffect(() => {
        getAvailableTypes()
            .then(res => {
                if (res?.types?.length) {
                    setAvailableTypes(mapApiTypes(res.types));
                }
            })
            .catch(() => {
                // Backend may be stale — static fallback already in state
            });
    }, []);

    const loadConnectors = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const res = await getConnections();
            const rows: ConnectorRow[] = (res?.connections ?? []).map(c => ({
                ...c,
                status: (c.validation_status === 'valid' ? 'active' : c.validation_status === 'invalid' ? 'error' : undefined) as ConnectorStatus | undefined,
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
        await addConnection({ source_type: formData.source_type, profile_name: formData.profile_name, config: formData.config } as ConnectionConfig);
        setShowAdd(false);
        await loadConnectors();
    };

    const handleEdit = async (formData: ConnectorFormState) => {
        if (!editTarget?.id) return;
        await put<unknown>(`/connections/${editTarget.id}`, {
            source_type: formData.source_type,
            profile_name: formData.profile_name,
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

    const handleTest = async (c: ConnectorRow) => {
        setTestingId(c.id);
        try {
            await post<unknown>(`/connections/${c.id}/test`, {});
            setTestResults(prev => ({ ...prev, [c.id]: { ok: true, message: 'Connection successful.' } }));
        } catch (e: any) {
            const msg = e?.response?.data?.error ?? e?.message ?? 'Connection failed.';
            setTestResults(prev => ({ ...prev, [c.id]: { ok: false, message: msg } }));
        } finally {
            setTestingId(null);
        }
    };

    const typeLabel = (t: string) => availableTypes.find(c => c.value === t)?.label ?? t;
    const typeIcon  = (t: string) => availableTypes.find(c => c.value === t)?.icon ?? <Database className="w-4 h-4" />;

    return (
        <div className="p-8 space-y-6">
            {/* Header */}
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Data Connectors</h1>
                    <p className="text-slate-500 mt-1 text-sm">
                        Configure connections to data sources for PII scanning. Each connector is isolated per organization and visible to all admins.
                    </p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                    <button
                        onClick={loadConnectors}
                        className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50"
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

            {/* Table */}
            {loading ? (
                <div className="space-y-3">
                    {[...Array(4)].map((_, i) => <div key={i} className="h-14 bg-slate-100 rounded-lg animate-pulse" />)}
                </div>
            ) : error ? (
                <div className="p-6 text-center text-red-600 bg-red-50 rounded-lg border border-red-100">{error}</div>
            ) : connectors.length === 0 ? (
                <div className="p-12 text-center bg-slate-50 rounded-xl border border-dashed border-slate-200">
                    <Database className="w-10 h-10 text-slate-300 mx-auto mb-3" />
                    <p className="text-slate-600 font-medium">No connectors configured</p>
                    <p className="text-slate-400 text-sm mt-1">Add a data source to start scanning for PII.</p>
                    <button
                        onClick={() => setShowAdd(true)}
                        className="mt-4 inline-flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                    >
                        <Plus className="w-4 h-4" /> Add Connector
                    </button>
                </div>
            ) : (
                <div className="border border-slate-200 rounded-xl overflow-hidden shadow-sm">
                    <table className="w-full text-sm">
                        <thead className="bg-slate-50 border-b border-slate-200">
                            <tr>
                                <th className="text-left px-5 py-3 font-semibold text-slate-600">Connector</th>
                                <th className="text-left px-5 py-3 font-semibold text-slate-600">Type</th>
                                <th className="text-left px-5 py-3 font-semibold text-slate-600">Status</th>
                                <th className="text-left px-5 py-3 font-semibold text-slate-600">Added</th>
                                <th className="text-right px-5 py-3 font-semibold text-slate-600">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {connectors.map(c => (
                                <React.Fragment key={c.id}>
                                    <tr className="hover:bg-slate-50 transition-colors">
                                        <td className="px-5 py-3">
                                            <div className="font-semibold text-slate-900">{c.profile_name}</div>
                                            <div className="text-xs text-slate-400 font-mono mt-0.5">{c.id.slice(0, 8)}…</div>
                                        </td>
                                        <td className="px-5 py-3">
                                            <div className="flex items-center gap-2 text-slate-700">
                                                <span className="text-slate-400">{typeIcon(c.source_type)}</span>
                                                {typeLabel(c.source_type)}
                                            </div>
                                        </td>
                                        <td className="px-5 py-3">{statusBadge(c.status ?? c.validation_status)}</td>
                                        <td className="px-5 py-3 text-slate-500 text-xs">{formatDate(c.created_at)}</td>
                                        <td className="px-5 py-3">
                                            <div className="flex items-center justify-end gap-1">
                                                <button
                                                    onClick={() => handleTest(c)}
                                                    disabled={testingId === c.id}
                                                    className="flex items-center gap-1.5 px-2.5 py-1.5 text-xs font-medium text-slate-600 bg-white border border-slate-200 rounded-md hover:bg-slate-50 disabled:opacity-50"
                                                >
                                                    {testingId === c.id ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Plug className="w-3.5 h-3.5" />}
                                                    Test
                                                </button>
                                                <button
                                                    onClick={() => setEditTarget(c)}
                                                    className="p-1.5 text-slate-400 hover:text-slate-700 hover:bg-slate-100 rounded-md"
                                                    title="Edit"
                                                >
                                                    <Edit2 className="w-4 h-4" />
                                                </button>
                                                <button
                                                    onClick={() => setDeleteConfirmId(c.id)}
                                                    className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md"
                                                    title="Delete"
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            </div>
                                        </td>
                                    </tr>

                                    {/* Test result row */}
                                    {testResults[c.id] && (
                                        <tr>
                                            <td colSpan={5} className={`px-5 py-2 text-xs font-medium ${testResults[c.id].ok ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}`}>
                                                <div className="flex items-center gap-2">
                                                    {testResults[c.id].ok ? <CheckCircle className="w-3.5 h-3.5" /> : <XCircle className="w-3.5 h-3.5" />}
                                                    {testResults[c.id].message}
                                                    <button
                                                        onClick={() => setTestResults(prev => { const n = { ...prev }; delete n[c.id]; return n; })}
                                                        className="ml-auto text-current opacity-50 hover:opacity-100"
                                                    >
                                                        <XCircle className="w-3.5 h-3.5" />
                                                    </button>
                                                </div>
                                            </td>
                                        </tr>
                                    )}

                                    {/* Delete confirm row */}
                                    {deleteConfirmId === c.id && (
                                        <tr>
                                            <td colSpan={5} className="px-5 py-3 bg-red-50 border-t border-red-100">
                                                <div className="flex items-center gap-4 text-sm">
                                                    <AlertTriangle className="w-4 h-4 text-red-500 shrink-0" />
                                                    <span className="text-red-700 font-medium flex-1">
                                                        Delete <strong>{c.profile_name}</strong>? Existing scan references will be preserved.
                                                    </span>
                                                    <button onClick={() => setDeleteConfirmId(null)} className="px-3 py-1.5 text-sm text-slate-600 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">Cancel</button>
                                                    <button onClick={() => handleDelete(c.id)} className="px-3 py-1.5 text-sm text-white bg-red-600 rounded-lg hover:bg-red-700">Delete</button>
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

            {showAdd && <ConnectorModal connectorTypes={availableTypes} onSave={handleAdd} onClose={() => setShowAdd(false)} />}
            {editTarget && <ConnectorModal initial={editTarget} connectorTypes={availableTypes} onSave={handleEdit} onClose={() => setEditTarget(null)} />}
        </div>
    );
}
