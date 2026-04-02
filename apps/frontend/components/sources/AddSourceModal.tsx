import React, { useState } from 'react';
import { X, Database, HardDrive, Cloud, Server, Plus, Loader2, CheckCircle, AlertCircle, Play } from 'lucide-react';
import { connectionsApi } from '@/services/connections.api';

interface SourceType {
    id: 'database' | 'filesystem' | 's3' | 'gcs' | 'other';
    name: string;
    icon: React.ReactNode;
    description: string;
}

const SOURCE_TYPES: SourceType[] = [
    {
        id: 'database',
        name: 'Database',
        icon: <Database className="w-6 h-6" />,
        description: 'PostgreSQL, MySQL, MongoDB, etc.',
    },
    {
        id: 'filesystem',
        name: 'Filesystem',
        icon: <HardDrive className="w-6 h-6" />,
        description: 'Local or network file systems',
    },
    {
        id: 's3',
        name: 'AWS S3',
        icon: <Cloud className="w-6 h-6" />,
        description: 'Amazon Simple Storage Service',
    },
    {
        id: 'gcs',
        name: 'Google Cloud Storage',
        icon: <Cloud className="w-6 h-6" />,
        description: 'GCS Buckets',
    },
    {
        id: 'other',
        name: 'Other',
        icon: <Server className="w-6 h-6" />,
        description: 'Custom data sources',
    }
];

interface AddSourceModalProps {
    isOpen: boolean;
    onClose: () => void;
}

export function AddSourceModal({ isOpen, onClose }: AddSourceModalProps) {
    const [step, setStep] = useState(1);
    const [selectedType, setSelectedType] = useState<SourceType['id'] | null>(null);

    // Common State
    const [name, setName] = useState('');
    const [environment, setEnvironment] = useState('prod');
    const [isReadOnly, setIsReadOnly] = useState(true);
    const [allowRemediation, setAllowRemediation] = useState(false);

    // Source Specific State
    const [dbConfig, setDbConfig] = useState({ type: 'postgresql', host: '', port: '', database: '', username: '', password: '', sslMode: 'require' });
    const [fsConfig, setFsConfig] = useState({ path: '', osType: 'linux' });
    const [s3Config, setS3Config] = useState({ bucket: '', region: 'us-east-1', accessKeyId: '', secretAccessKey: '' });
    const [gcsConfig, setGcsConfig] = useState({ bucket: '', projectId: '', serviceAccountJson: '' });

    const [isSubmitting, setIsSubmitting] = useState(false);
    const [isTesting, setIsTesting] = useState(false);
    const [error, setError] = useState<string | null>(null);
    const [success, setSuccess] = useState(false);
    const [testResult, setTestResult] = useState<{ success: boolean; message: string } | null>(null);

    if (!isOpen) return null;

    const resetForm = () => {
        setStep(1);
        setSelectedType(null);
        setName('');
        setEnvironment('prod');
        setError(null);
        setTestResult(null);
        setDbConfig({ type: 'postgresql', host: '', port: '', database: '', username: '', password: '', sslMode: 'require' });
        setFsConfig({ path: '', osType: 'linux' });
        setS3Config({ bucket: '', region: 'us-east-1', accessKeyId: '', secretAccessKey: '' });
        setGcsConfig({ bucket: '', projectId: '', serviceAccountJson: '' });
    };

    const handleClose = () => {
        resetForm();
        onClose();
    };

    const getConfiguration = () => {
        switch (selectedType) {
            case 'database':
                return {
                    host: dbConfig.host,
                    port: dbConfig.port,
                    database: dbConfig.database,
                    username: dbConfig.username,
                    password: dbConfig.password,
                    ssl_mode: dbConfig.sslMode
                };
            case 'filesystem': return fsConfig;
            case 's3':
                return {
                    bucket: s3Config.bucket,
                    region: s3Config.region,
                    access_key: s3Config.accessKeyId,
                    secret_key: s3Config.secretAccessKey
                };
            case 'gcs': return {
                bucket: gcsConfig.bucket,
                project_id: gcsConfig.projectId,
                service_account_json: gcsConfig.serviceAccountJson,
                credentials_file: gcsConfig.serviceAccountJson // Alias for backend compatibility
            };
            default: return {};
        }
    };

    const getSourceTypeID = () => {
        if (selectedType === 'database') return dbConfig.type; // 'postgresql' or 'mysql'
        return selectedType!;
    };

    const validateForm = () => {
        if (!name) return 'Source Name is required';
        if (selectedType === 'database' && (!dbConfig.host || !dbConfig.database)) return 'Host and Database are required';
        if (selectedType === 'filesystem' && !fsConfig.path) return 'Path is required';
        if (selectedType === 's3' && (!s3Config.bucket || !s3Config.region)) return 'Bucket and Region are required';
        if (selectedType === 'gcs' && (!gcsConfig.bucket || !gcsConfig.projectId)) return 'Bucket and Project ID are required';
        return null;
    };

    const handleTestConnection = async () => {
        const validationError = validateForm();
        if (validationError) {
            setError(validationError);
            return;
        }

        setIsTesting(true);
        setError(null);
        setTestResult(null);

        try {
            const config = getConfiguration();
            // @ts-ignore - API types might need update but valid payload
            const result: any = await connectionsApi.testConnection({
                source_type: getSourceTypeID(),
                profile_name: name, // Using name for test context
                config: {
                    ...config,
                    environment,
                    read_only: isReadOnly // Pass read-only flag for accurate testing
                }
            });

            setTestResult({
                success: result.success !== false, // Assume success unless explicitly false
                message: result.message || 'Connection successful'
            });
        } catch (err: any) {
            setTestResult({
                success: false,
                message: err.message || 'Connection failed'
            });
        } finally {
            setIsTesting(false);
        }
    };

    const handleSubmit = async () => {
        const validationError = validateForm();
        if (validationError) {
            setError(validationError);
            return;
        }

        setIsSubmitting(true);
        setError(null);

        try {
            await connectionsApi.addConnection({
                source_type: getSourceTypeID(),
                profile_name: name,
                config: {
                    ...getConfiguration(),
                    environment,
                    read_only: isReadOnly,
                    allow_remediation: allowRemediation
                }
            });

            setSuccess(true);
            setTimeout(() => {
                handleClose();
                setSuccess(false);
            }, 1500);
        } catch (err: any) {
            setError(err.message || 'Unable to establish connection to your data source. This prevents PII discovery scans from running. Please verify your credentials and network settings to ensure DPDPA compliance scanning can proceed.');
        } finally {
            setIsSubmitting(false);
        }
    };

    const renderCommonFields = () => (
        <div className="grid grid-cols-2 gap-4 mb-4">
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Source Name *</label>
                <input
                    type="text"
                    value={name}
                    onChange={(e) => setName(e.target.value)}
                    placeholder="e.g., Production DB"
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500 transition-all"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Environment *</label>
                <select
                    value={environment}
                    onChange={(e) => setEnvironment(e.target.value)}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                    <option value="prod">Production</option>
                    <option value="dev">Development</option>
                    <option value="qa">QA</option>
                </select>
            </div>
        </div>
    );

    const renderDatabaseForm = () => (
        <div className="space-y-4">
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Database Type</label>
                <select
                    value={dbConfig.type}
                    onChange={(e) => setDbConfig({ ...dbConfig, type: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                    <option value="postgresql">PostgreSQL</option>
                    <option value="mysql">MySQL</option>
                </select>
            </div>
            <div className="grid grid-cols-3 gap-4">
                <div className="col-span-2">
                    <label className="block text-sm font-medium text-foreground font-semibold mb-2">Host *</label>
                    <input
                        type="text"
                        value={dbConfig.host}
                        onChange={(e) => setDbConfig({ ...dbConfig, host: e.target.value })}
                        placeholder="localhost"
                        className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-foreground font-semibold mb-2">Port</label>
                    <input
                        type="text"
                        value={dbConfig.port}
                        onChange={(e) => setDbConfig({ ...dbConfig, port: e.target.value })}
                        placeholder="5432"
                        className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                </div>
            </div>
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Database Name *</label>
                <input
                    type="text"
                    value={dbConfig.database}
                    onChange={(e) => setDbConfig({ ...dbConfig, database: e.target.value })}
                    placeholder="my_database"
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
            </div>
            <div className="grid grid-cols-2 gap-4">
                <div>
                    <label className="block text-sm font-medium text-foreground font-semibold mb-2">Username</label>
                    <input
                        type="text"
                        value={dbConfig.username}
                        onChange={(e) => setDbConfig({ ...dbConfig, username: e.target.value })}
                        placeholder="admin"
                        className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-foreground font-semibold mb-2">Password</label>
                    <input
                        type="password"
                        value={dbConfig.password}
                        onChange={(e) => setDbConfig({ ...dbConfig, password: e.target.value })}
                        placeholder="••••••••"
                        className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                </div>
            </div>
        </div>
    );

    const renderFilesystemForm = () => (
        <div className="space-y-4">
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Root Path *</label>
                <input
                    type="text"
                    value={fsConfig.path}
                    onChange={(e) => setFsConfig({ ...fsConfig, path: e.target.value })}
                    placeholder="/var/www/html or C:\Users\Data"
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">OS Type</label>
                <select
                    value={fsConfig.osType}
                    onChange={(e) => setFsConfig({ ...fsConfig, osType: e.target.value })}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
                >
                    <option value="linux">Linux / Unix</option>
                    <option value="windows">Windows</option>
                    <option value="macos">macOS</option>
                </select>
            </div>
        </div>
    );

    const renderS3Form = () => (
        <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
                <div>
                    <label className="block text-sm font-medium text-foreground font-semibold mb-2">Bucket Name *</label>
                    <input
                        type="text"
                        value={s3Config.bucket}
                        onChange={(e) => setS3Config({ ...s3Config, bucket: e.target.value })}
                        placeholder="my-company-data"
                        className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-foreground font-semibold mb-2">Region *</label>
                    <input
                        type="text"
                        value={s3Config.region}
                        onChange={(e) => setS3Config({ ...s3Config, region: e.target.value })}
                        placeholder="us-east-1"
                        className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    />
                </div>
            </div>
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Access Key ID</label>
                <input
                    type="text"
                    value={s3Config.accessKeyId}
                    onChange={(e) => setS3Config({ ...s3Config, accessKeyId: e.target.value })}
                    placeholder="AKIA..."
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Secret Access Key</label>
                <input
                    type="password"
                    value={s3Config.secretAccessKey}
                    onChange={(e) => setS3Config({ ...s3Config, secretAccessKey: e.target.value })}
                    placeholder="Within AWS IAM console..."
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
            </div>
        </div>
    );

    const renderGCSForm = () => (
        <div className="space-y-4">
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Project ID *</label>
                <input
                    type="text"
                    value={gcsConfig.projectId}
                    onChange={(e) => setGcsConfig({ ...gcsConfig, projectId: e.target.value })}
                    placeholder="my-gcp-project"
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Bucket Name *</label>
                <input
                    type="text"
                    value={gcsConfig.bucket}
                    onChange={(e) => setGcsConfig({ ...gcsConfig, bucket: e.target.value })}
                    placeholder="my-gcs-bucket"
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
            </div>
            <div>
                <label className="block text-sm font-medium text-foreground font-semibold mb-2">Service Account JSON (Base64 or Path)</label>
                <textarea
                    value={gcsConfig.serviceAccountJson}
                    onChange={(e) => setGcsConfig({ ...gcsConfig, serviceAccountJson: e.target.value })}
                    placeholder="{ ... }"
                    rows={3}
                    className="w-full px-4 py-2 bg-background border border-border rounded-lg text-foreground placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-blue-500 font-mono text-xs"
                />
            </div>
        </div>
    );

    const renderConfigForm = () => {
        switch (selectedType) {
            case 'database': return renderDatabaseForm();
            case 'filesystem': return renderFilesystemForm();
            case 's3': return renderS3Form();
            case 'gcs': return renderGCSForm();
            default: return <div className="p-4 text-muted-foreground text-center">Generic configuration not yet implemented for this type.</div>;
        }
    };

    return (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 backdrop-blur-sm">
            <div className="bg-card rounded-lg shadow-2xl w-full max-w-3xl max-h-[90vh] overflow-hidden border border-border flex flex-col">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-border bg-card">
                    <div>
                        <h2 className="text-xl font-semibold text-foreground">Add Data Source</h2>
                        <p className="text-sm text-muted-foreground mt-1">
                            {step === 1 ? 'Select Source Type' : `Configure ${SOURCE_TYPES.find(s => s.id === selectedType)?.name}`}
                        </p>
                    </div>
                    <button onClick={handleClose} className="p-2 hover:bg-muted rounded-lg transition-colors text-muted-foreground hover:text-foreground">
                        <X className="w-5 h-5" />
                    </button>
                </div>

                {/* Content */}
                <div className="p-6 overflow-y-auto overflow-x-hidden flex-1">
                    {success ? (
                        <div className="flex flex-col items-center justify-center h-full py-12 text-center animate-in fade-in zoom-in duration-300">
                            <div className="w-16 h-16 bg-green-500/10 rounded-full flex items-center justify-center mb-4 border border-green-500/20">
                                <CheckCircle className="w-8 h-8 text-green-500" />
                            </div>
                            <h3 className="text-xl font-semibold text-foreground mb-2">Connection Added!</h3>
                            <p className="text-muted-foreground">Successfully connected to {name}</p>
                        </div>
                    ) : step === 1 ? (
                        <div className="grid grid-cols-2 gap-4 animate-in slide-in-from-right duration-300">
                            {SOURCE_TYPES.map((source) => (
                                <button
                                    key={source.id}
                                    onClick={() => setSelectedType(source.id)}
                                    className={`
                                        p-4 rounded-lg border-2 transition-all text-left group
                                        ${selectedType === source.id ? 'border-blue-500 bg-blue-500/10' : 'border-border bg-secondary hover:border-border hover:bg-muted'}
                                    `}
                                >
                                    <div className="flex items-start gap-4">
                                        <div className={`
                                            p-3 rounded-lg transition-colors
                                            ${selectedType === source.id ? 'bg-blue-500/20 text-blue-600' : 'bg-slate-100 group-hover:bg-slate-200 text-slate-500 group-hover:text-slate-700'}
                                        `}>
                                            {source.icon}
                                        </div>
                                        <div className="flex-1">
                                            <div className={`font-semibold ${selectedType === source.id ? 'text-blue-700 font-bold' : 'text-slate-700'}`}>
                                                {source.name}
                                            </div>
                                            <div className="text-sm text-muted-foreground mt-1">{source.description}</div>
                                        </div>
                                    </div>
                                </button>
                            ))}
                        </div>
                    ) : (
                        <div className="space-y-6 animate-in slide-in-from-right duration-300">
                            {error && (
                                <div className="p-4 bg-red-50 border border-red-200 rounded-lg flex items-center gap-3 text-red-700">
                                    <AlertCircle className="w-5 h-5 flex-shrink-0" />
                                    <p className="text-sm font-medium">{error}</p>
                                </div>
                            )}

                            {renderCommonFields()}

                            <div className="bg-background/50 p-6 border border-border rounded-xl">
                                {renderConfigForm()}
                            </div>

                            <div className="bg-background p-4 rounded-lg border border-border space-y-3">
                                <label className="block text-sm font-medium text-foreground font-semibold">
                                    Connection Scope
                                </label>
                                <div className="flex items-center gap-3">
                                    <div className="flex items-center gap-2">
                                        <input
                                            type="checkbox"
                                            id="read-only"
                                            checked={isReadOnly}
                                            onChange={(e) => setIsReadOnly(e.target.checked)}
                                            className="w-4 h-4 rounded border-border bg-muted text-blue-500 focus:ring-blue-500 focus:ring-offset-0 focus:ring-offset-white"
                                        />
                                        <label htmlFor="read-only" className="text-sm text-muted-foreground select-none cursor-pointer">
                                            Read-only access
                                        </label>
                                    </div>
                                    <div className="w-px h-4 bg-accent mx-2"></div>
                                    <div className="flex items-center gap-2 opacity-50 cursor-not-allowed" title="Remediation is currently disabled">
                                        <input
                                            type="checkbox"
                                            id="allow-remediation"
                                            checked={allowRemediation}
                                            onChange={(e) => setAllowRemediation(e.target.checked)}
                                            disabled={true}
                                            className="w-4 h-4 rounded border-border bg-muted text-blue-500 focus:ring-blue-500 focus:ring-offset-0 focus:ring-offset-white cursor-not-allowed"
                                        />
                                        <label htmlFor="allow-remediation" className="text-sm text-muted-foreground select-none cursor-not-allowed">
                                            Allow remediation actions (Coming Soon)
                                        </label>
                                    </div>
                                </div>
                            </div>

                            {/* Test Connection Result */}
                            {testResult && (
                                <div className={`p-4 rounded-lg border flex items-center gap-3 ${testResult.success
                                    ? 'bg-green-50 border-green-200 text-green-700'
                                    : 'bg-red-50 border-red-200 text-red-700'
                                    }`}>
                                    {testResult.success ? <CheckCircle className="w-5 h-5 flex-shrink-0" /> : <AlertCircle className="w-5 h-5 flex-shrink-0" />}
                                    <div className="text-sm font-medium flex-1">
                                        {testResult.success ? 'Connection verified successfully' : testResult.message}
                                    </div>
                                </div>
                            )}
                        </div>
                    )}
                </div>

                {/* Footer */}
                <div className="flex items-center justify-between px-6 py-4 border-t border-border bg-card">
                    <button
                        onClick={handleClose}
                        disabled={isSubmitting || isTesting}
                        className="px-4 py-2 text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
                    >
                        Cancel
                    </button>
                    {!success && (
                        <div className="flex gap-3">
                            {step === 2 && (
                                <>
                                    <button
                                        onClick={() => setStep(1)}
                                        disabled={isSubmitting || isTesting}
                                        className="px-4 py-2 bg-muted hover:bg-accent text-foreground rounded-lg font-medium transition-colors border border-border disabled:opacity-50"
                                    >
                                        Back
                                    </button>
                                    <button
                                        onClick={handleTestConnection}
                                        disabled={isSubmitting || isTesting}
                                        className="px-4 py-2 bg-muted hover:bg-accent text-foreground rounded-lg font-medium transition-colors border border-border disabled:opacity-50 flex items-center gap-2"
                                    >
                                        {isTesting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                                        Test Connection
                                    </button>
                                    <button
                                        onClick={handleSubmit}
                                        disabled={isSubmitting || isTesting}
                                        className="flex items-center gap-2 px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-blue-600/50 text-white rounded-lg font-medium transition-colors shadow-lg shadow-blue-200/50"
                                    >
                                        {isSubmitting ? (
                                            <>
                                                <Loader2 className="w-4 h-4 animate-spin" />
                                                Saving...
                                            </>
                                        ) : (
                                            'Save Connection'
                                        )}
                                    </button>
                                </>
                            )}
                            {step === 1 && (
                                <button
                                    onClick={() => setStep(2)}
                                    disabled={!selectedType}
                                    className="px-6 py-2 bg-blue-600 hover:bg-blue-700 disabled:bg-muted disabled:text-slate-500 text-white rounded-lg font-medium transition-colors shadow-lg shadow-blue-200/50"
                                >
                                    Next Step
                                </button>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}
