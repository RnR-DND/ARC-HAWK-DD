'use client';

import React, { useState, useEffect, useCallback, useRef } from 'react';
import {
    Plus, Trash2, Edit2, Download, Upload, RefreshCw,
    AlertTriangle, CheckCircle, XCircle, Play, FlaskConical
} from 'lucide-react';
import { BarChart, Bar, XAxis, YAxis, Tooltip as RechartsTooltip, ResponsiveContainer } from 'recharts';
import { get, post, put, del } from '@/utils/api-client';

// ─── Types ────────────────────────────────────────────────────────────────────

interface CustomPattern {
    id: string;
    name: string;
    display_name: string;
    regex: string;
    pii_category: string;
    sensitivity: 'low' | 'medium' | 'high' | 'critical';
    description: string;
    is_active: boolean;
    match_count: number;
    fp_rate: number;
    created_at: string;
    test_cases?: string[];
    context_keywords?: string[];
    negative_keywords?: string[];
}

interface PatternStats {
    scan_history: { scan_label: string; match_count: number }[];
    total_matches: number;
    fp_rate: number;
}

interface TestResult {
    input: string;
    js_match: boolean;
    backend_match: boolean;
    mismatch: boolean;
}

interface PatternFormData {
    name: string;
    display_name: string;
    regex: string;
    pii_category: string;
    sensitivity: 'low' | 'medium' | 'high' | 'critical';
    description: string;
    test_cases: string[];
    context_keywords: string[];
    negative_keywords: string[];
}

const EMPTY_FORM: PatternFormData = {
    name: '',
    display_name: '',
    regex: '',
    pii_category: '',
    sensitivity: 'medium',
    description: '',
    test_cases: [],
    context_keywords: [],
    negative_keywords: [],
};

const SENSITIVITY_COLORS: Record<string, string> = {
    low: 'bg-green-100 text-green-800 border-green-200',
    medium: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    high: 'bg-orange-100 text-orange-800 border-orange-200',
    critical: 'bg-red-100 text-red-800 border-red-200',
};

const PII_CATEGORIES = [
    'Financial', 'Health', 'Identity', 'Authentication', 'Contact',
    'Location', 'Biometric', 'Government ID', 'Custom',
];

// ─── API helpers ──────────────────────────────────────────────────────────────

async function apiGetPatterns(): Promise<CustomPattern[]> {
    try {
        const res = await get<any>('/patterns');
        return Array.isArray(res?.data) ? res.data : Array.isArray(res) ? res : [];
    } catch { return []; }
}

async function apiCreatePattern(data: Omit<PatternFormData, 'test_cases'> & { test_cases: string[] }): Promise<CustomPattern> {
    const res = await post<any>('/patterns', data);
    return res?.data ?? res;
}

async function apiUpdatePattern(id: string, data: Partial<CustomPattern>): Promise<CustomPattern> {
    const res = await put<any>(`/patterns/${id}`, data);
    return res?.data ?? res;
}

async function apiDeletePattern(id: string): Promise<void> {
    await del<any>(`/patterns/${id}`);
}

async function apiTestPattern(id: string, inputs: string[]): Promise<{ results: { input: string; matched: boolean }[] }> {
    const res = await post<any>(`/patterns/${id}/test`, { inputs });
    return res?.data ?? res;
}

async function apiGetPatternStats(id: string): Promise<PatternStats> {
    const res = await get<any>(`/patterns/${id}/stats`);
    return res?.data ?? res;
}

async function apiRecordFalsePositive(id: string, input: string): Promise<void> {
    await post<any>(`/patterns/${id}/false-positive`, { input });
}

// ─── Sub-components ───────────────────────────────────────────────────────────

function SensitivityBadge({ level }: { level: string }) {
    const cls = SENSITIVITY_COLORS[level] ?? 'bg-slate-100 text-slate-700 border-slate-200';
    return (
        <span className={`inline-flex items-center px-2 py-0.5 rounded-full text-xs font-semibold border ${cls} uppercase`}>
            {level}
        </span>
    );
}

function Toggle({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
    return (
        <label className="flex items-center cursor-pointer">
            <input type="checkbox" checked={checked} onChange={e => onChange(e.target.checked)} className="sr-only peer" />
            <div className="relative w-9 h-5 bg-slate-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600" />
        </label>
    );
}

function FPWarningBadge({ rate }: { rate: number }) {
    if (rate < 25) return null;
    return (
        <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold border bg-amber-50 text-amber-700 border-amber-200">
            <AlertTriangle className="w-3 h-3" />
            FP {rate.toFixed(0)}%
        </span>
    );
}

// ─── Stats Chart Modal ────────────────────────────────────────────────────────

function StatsChart({ patternId, onClose }: { patternId: string; onClose: () => void }) {
    const [stats, setStats] = useState<PatternStats | null>(null);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        apiGetPatternStats(patternId).then(s => { setStats(s); setLoading(false); }).catch(() => setLoading(false));
    }, [patternId]);

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={onClose}>
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-lg p-6" onClick={e => e.stopPropagation()}>
                <div className="flex items-center justify-between mb-4">
                    <h3 className="text-lg font-semibold text-slate-900">Match Frequency — Last 7 Scans</h3>
                    <button onClick={onClose} className="text-slate-400 hover:text-slate-600"><XCircle className="w-5 h-5" /></button>
                </div>
                {loading ? (
                    <div className="h-40 flex items-center justify-center text-slate-400">Loading stats...</div>
                ) : !stats || !stats.scan_history?.length ? (
                    <div className="h-40 flex items-center justify-center text-slate-400">No scan history available.</div>
                ) : (
                    <>
                        <ResponsiveContainer width="100%" height={200} minWidth={0}>
                            <BarChart data={stats.scan_history}>
                                <XAxis dataKey="scan_label" tick={{ fontSize: 11 }} />
                                <YAxis tick={{ fontSize: 11 }} />
                                <RechartsTooltip />
                                <Bar dataKey="match_count" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                            </BarChart>
                        </ResponsiveContainer>
                        <div className="flex gap-6 mt-4 text-sm text-slate-600">
                            <div><span className="font-semibold">{stats.total_matches}</span> total matches</div>
                            <div><span className={`font-semibold ${stats.fp_rate > 25 ? 'text-amber-600' : 'text-slate-900'}`}>{stats.fp_rate?.toFixed(1)}%</span> FP rate</div>
                        </div>
                    </>
                )}
            </div>
        </div>
    );
}

// ─── Inline Test Runner ───────────────────────────────────────────────────────

function TestRunner({ pattern }: { pattern: CustomPattern }) {
    const [inputs, setInputs] = useState('');
    const [results, setResults] = useState<TestResult[]>([]);
    const [running, setRunning] = useState(false);
    const [hasMismatch, setHasMismatch] = useState(false);

    const runTest = useCallback(async () => {
        const lines = inputs.split('\n').map(l => l.trim()).filter(Boolean);
        if (!lines.length) return;
        setRunning(true);
        setResults([]);

        // JS regex evaluation (browser-side proxy for WASM)
        let jsResults: Record<string, boolean> = {};
        try {
            const re = new RegExp(pattern.regex);
            lines.forEach(l => { jsResults[l] = re.test(l); });
        } catch {
            lines.forEach(l => { jsResults[l] = false; });
        }

        // Backend test
        let backendResults: Record<string, boolean> = {};
        try {
            const res = await apiTestPattern(pattern.id, lines);
            (res?.results ?? []).forEach((r: { input: string; matched: boolean }) => {
                backendResults[r.input] = r.matched;
            });
        } catch {
            lines.forEach(l => { backendResults[l] = jsResults[l]; }); // fallback
        }

        const combined: TestResult[] = lines.map(l => ({
            input: l,
            js_match: jsResults[l] ?? false,
            backend_match: backendResults[l] ?? false,
            mismatch: jsResults[l] !== backendResults[l],
        }));

        setHasMismatch(combined.some(r => r.mismatch));
        setResults(combined);
        setRunning(false);
    }, [inputs, pattern]);

    const recordFP = async (input: string) => {
        await apiRecordFalsePositive(pattern.id, input);
    };

    return (
        <div className="mt-4 border border-slate-200 rounded-lg overflow-hidden">
            <div className="px-4 py-3 bg-slate-50 border-b border-slate-200 flex items-center gap-2">
                <FlaskConical className="w-4 h-4 text-slate-500" />
                <span className="text-sm font-medium text-slate-700">Inline Test Runner</span>
            </div>
            <div className="grid grid-cols-2 divide-x divide-slate-200">
                {/* Input */}
                <div className="p-4 space-y-3">
                    <label className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Test Inputs (one per line)</label>
                    <textarea
                        value={inputs}
                        onChange={e => setInputs(e.target.value)}
                        placeholder="john.doe@example.com&#10;not-an-email&#10;test@test.org"
                        rows={6}
                        className="w-full text-sm font-mono border border-slate-200 rounded-lg p-2.5 focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none text-slate-900"
                    />
                    <button
                        onClick={runTest}
                        disabled={running || !inputs.trim()}
                        className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                    >
                        {running ? <RefreshCw className="w-4 h-4 animate-spin" /> : <Play className="w-4 h-4" />}
                        {running ? 'Running...' : 'Run Tests'}
                    </button>
                </div>

                {/* Results */}
                <div className="p-4 space-y-3">
                    <label className="text-xs font-semibold text-slate-500 uppercase tracking-wide">Results</label>
                    {hasMismatch && (
                        <div className="flex items-center gap-2 px-3 py-2 bg-yellow-50 border border-yellow-200 rounded-lg text-xs font-semibold text-yellow-800">
                            <AlertTriangle className="w-3.5 h-3.5" />
                            REGEX_ENGINE_MISMATCH — JS and backend results differ
                        </div>
                    )}
                    {results.length === 0 ? (
                        <div className="text-sm text-slate-400 py-4 text-center">Results will appear here after running tests.</div>
                    ) : (
                        <div className="space-y-2 max-h-52 overflow-y-auto">
                            {results.map((r, i) => (
                                <div key={i} className={`flex items-start gap-2 px-2.5 py-2 rounded-lg text-sm ${r.mismatch ? 'bg-yellow-50 border border-yellow-200' : ''}`}>
                                    <div className="flex flex-col gap-1 flex-1 min-w-0">
                                        <code className="text-xs font-mono text-slate-700 truncate">{r.input}</code>
                                        <div className="flex gap-3 text-xs text-slate-500">
                                            <span className="flex items-center gap-1">
                                                {r.js_match ? <CheckCircle className="w-3 h-3 text-green-500" /> : <XCircle className="w-3 h-3 text-red-400" />}
                                                JS
                                            </span>
                                            <span className="flex items-center gap-1">
                                                {r.backend_match ? <CheckCircle className="w-3 h-3 text-green-500" /> : <XCircle className="w-3 h-3 text-red-400" />}
                                                Backend
                                            </span>
                                        </div>
                                    </div>
                                    {r.backend_match && (
                                        <button
                                            onClick={() => recordFP(r.input)}
                                            title="Mark as false positive"
                                            className="text-xs text-slate-400 hover:text-amber-600 shrink-0 mt-0.5"
                                        >
                                            FP
                                        </button>
                                    )}
                                </div>
                            ))}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );
}

// ─── Create/Edit Modal ────────────────────────────────────────────────────────

function PatternModal({
    initial,
    onSave,
    onClose,
}: {
    initial?: CustomPattern;
    onSave: (data: PatternFormData) => Promise<void>;
    onClose: () => void;
}) {
    const [form, setForm] = useState<PatternFormData>(
        initial
            ? {
                name: initial.name,
                display_name: initial.display_name,
                regex: initial.regex,
                pii_category: initial.pii_category,
                sensitivity: initial.sensitivity,
                description: initial.description,
                test_cases: initial.test_cases ?? [],
                context_keywords: initial.context_keywords ?? [],
                negative_keywords: initial.negative_keywords ?? [],
            }
            : { ...EMPTY_FORM }
    );
    const [saving, setSaving] = useState(false);
    const [regexError, setRegexError] = useState('');
    const [testCaseInput, setTestCaseInput] = useState('');
    const [contextKwInput, setContextKwInput] = useState('');
    const [negativeKwInput, setNegativeKwInput] = useState('');

    const validateRegex = (r: string) => {
        try { new RegExp(r); setRegexError(''); }
        catch (e: any) { setRegexError(e.message); }
    };

    const setField = <K extends keyof PatternFormData>(k: K, v: PatternFormData[K]) => {
        setForm(f => ({ ...f, [k]: v }));
        if (k === 'regex') validateRegex(v as string);
    };

    const addTestCase = () => {
        const tc = testCaseInput.trim();
        if (tc && !form.test_cases.includes(tc)) {
            setForm(f => ({ ...f, test_cases: [...f.test_cases, tc] }));
            setTestCaseInput('');
        }
    };

    const removeTestCase = (tc: string) => {
        setForm(f => ({ ...f, test_cases: f.test_cases.filter(t => t !== tc) }));
    };

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault();
        if (regexError) return;
        setSaving(true);
        try { await onSave(form); }
        finally { setSaving(false); }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4" onClick={onClose}>
            <div className="bg-white rounded-xl shadow-2xl w-full max-w-2xl max-h-[90vh] overflow-y-auto" onClick={e => e.stopPropagation()}>
                <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between sticky top-0 bg-white z-10">
                    <h2 className="text-lg font-semibold text-slate-900">{initial ? 'Edit Pattern' : 'Create Pattern'}</h2>
                    <button onClick={onClose} className="text-slate-400 hover:text-slate-600"><XCircle className="w-5 h-5" /></button>
                </div>
                <form onSubmit={handleSubmit} className="p-6 space-y-5">
                    <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-1.5">
                            <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Name <span className="text-red-500">*</span></label>
                            <input
                                required
                                value={form.name}
                                onChange={e => setField('name', e.target.value)}
                                placeholder="email_address"
                                className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900"
                            />
                            <p className="text-xs text-slate-400">Unique identifier (snake_case)</p>
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Display Name <span className="text-red-500">*</span></label>
                            <input
                                required
                                value={form.display_name}
                                onChange={e => setField('display_name', e.target.value)}
                                placeholder="Email Address"
                                className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900"
                            />
                        </div>
                    </div>

                    <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Regex Pattern <span className="text-red-500">*</span></label>
                        <input
                            required
                            value={form.regex}
                            onChange={e => setField('regex', e.target.value)}
                            placeholder="[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}"
                            className={`w-full text-sm font-mono border rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900 ${regexError ? 'border-red-400 bg-red-50' : 'border-slate-200'}`}
                        />
                        {regexError && <p className="text-xs text-red-600">{regexError}</p>}
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                        <div className="space-y-1.5">
                            <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">PII Category <span className="text-red-500">*</span></label>
                            <select
                                required
                                value={form.pii_category}
                                onChange={e => setField('pii_category', e.target.value)}
                                className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900 bg-white"
                            >
                                <option value="">Select category...</option>
                                {PII_CATEGORIES.map(c => <option key={c} value={c}>{c}</option>)}
                            </select>
                        </div>
                        <div className="space-y-1.5">
                            <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Sensitivity <span className="text-red-500">*</span></label>
                            <select
                                value={form.sensitivity}
                                onChange={e => setField('sensitivity', e.target.value as PatternFormData['sensitivity'])}
                                className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900 bg-white"
                            >
                                <option value="low">Low</option>
                                <option value="medium">Medium</option>
                                <option value="high">High</option>
                                <option value="critical">Critical</option>
                            </select>
                        </div>
                    </div>

                    <div className="space-y-1.5">
                        <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Description</label>
                        <textarea
                            value={form.description}
                            onChange={e => setField('description', e.target.value)}
                            placeholder="Detects email addresses in standard format..."
                            rows={2}
                            className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900 resize-none"
                        />
                    </div>

                    <div className="space-y-2">
                        <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Test Cases</label>
                        <div className="flex gap-2">
                            <input
                                value={testCaseInput}
                                onChange={e => setTestCaseInput(e.target.value)}
                                onKeyDown={e => { if (e.key === 'Enter') { e.preventDefault(); addTestCase(); } }}
                                placeholder="Add a test case string..."
                                className="flex-1 text-sm font-mono border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 text-slate-900"
                            />
                            <button type="button" onClick={addTestCase} className="px-3 py-2 bg-slate-100 hover:bg-slate-200 rounded-lg text-sm font-medium text-slate-700">
                                Add
                            </button>
                        </div>
                        {form.test_cases.length > 0 && (
                            <div className="flex flex-wrap gap-2">
                                {form.test_cases.map(tc => (
                                    <span key={tc} className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-slate-100 rounded-full text-xs font-mono text-slate-700">
                                        {tc}
                                        <button type="button" onClick={() => removeTestCase(tc)} className="text-slate-400 hover:text-red-500">
                                            <XCircle className="w-3 h-3" />
                                        </button>
                                    </span>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Context Keywords */}
                    <div className="space-y-2 border-t border-slate-100 pt-4">
                        <div>
                            <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Boost Keywords</label>
                            <p className="text-xs text-slate-400 mt-0.5">{'Words that, if found near a match, increase confidence (e.g. "aadhaar", "national id").'}</p>
                        </div>
                        <div className="flex gap-2">
                            <input
                                value={contextKwInput}
                                onChange={e => setContextKwInput(e.target.value)}
                                onKeyDown={e => {
                                    if (e.key === 'Enter') {
                                        e.preventDefault();
                                        const kw = contextKwInput.trim();
                                        if (kw && !form.context_keywords.includes(kw)) {
                                            setForm(f => ({ ...f, context_keywords: [...f.context_keywords, kw] }));
                                            setContextKwInput('');
                                        }
                                    }
                                }}
                                placeholder="e.g. aadhaar, national id..."
                                className="flex-1 text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-green-500 text-slate-900"
                            />
                            <button
                                type="button"
                                onClick={() => {
                                    const kw = contextKwInput.trim();
                                    if (kw && !form.context_keywords.includes(kw)) {
                                        setForm(f => ({ ...f, context_keywords: [...f.context_keywords, kw] }));
                                        setContextKwInput('');
                                    }
                                }}
                                className="px-3 py-2 bg-green-50 hover:bg-green-100 border border-green-200 rounded-lg text-sm font-medium text-green-700"
                            >
                                Add
                            </button>
                        </div>
                        {form.context_keywords.length > 0 && (
                            <div className="flex flex-wrap gap-2">
                                {form.context_keywords.map(kw => (
                                    <span key={kw} className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-green-50 border border-green-200 rounded-full text-xs font-medium text-green-700">
                                        {kw}
                                        <button type="button" onClick={() => setForm(f => ({ ...f, context_keywords: f.context_keywords.filter(k => k !== kw) }))} className="text-green-400 hover:text-red-500">
                                            <XCircle className="w-3 h-3" />
                                        </button>
                                    </span>
                                ))}
                            </div>
                        )}
                    </div>

                    {/* Negative Keywords */}
                    <div className="space-y-2">
                        <div>
                            <label className="text-xs font-semibold text-slate-600 uppercase tracking-wide">Suppress Keywords</label>
                            <p className="text-xs text-slate-400 mt-0.5">{'Words that, if found near a match, decrease confidence (e.g. "test", "sample", "dummy").'}</p>
                        </div>
                        <div className="flex gap-2">
                            <input
                                value={negativeKwInput}
                                onChange={e => setNegativeKwInput(e.target.value)}
                                onKeyDown={e => {
                                    if (e.key === 'Enter') {
                                        e.preventDefault();
                                        const kw = negativeKwInput.trim();
                                        if (kw && !form.negative_keywords.includes(kw)) {
                                            setForm(f => ({ ...f, negative_keywords: [...f.negative_keywords, kw] }));
                                            setNegativeKwInput('');
                                        }
                                    }
                                }}
                                placeholder="e.g. test, sample, dummy..."
                                className="flex-1 text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-red-400 text-slate-900"
                            />
                            <button
                                type="button"
                                onClick={() => {
                                    const kw = negativeKwInput.trim();
                                    if (kw && !form.negative_keywords.includes(kw)) {
                                        setForm(f => ({ ...f, negative_keywords: [...f.negative_keywords, kw] }));
                                        setNegativeKwInput('');
                                    }
                                }}
                                className="px-3 py-2 bg-red-50 hover:bg-red-100 border border-red-200 rounded-lg text-sm font-medium text-red-700"
                            >
                                Add
                            </button>
                        </div>
                        {form.negative_keywords.length > 0 && (
                            <div className="flex flex-wrap gap-2">
                                {form.negative_keywords.map(kw => (
                                    <span key={kw} className="inline-flex items-center gap-1.5 px-2.5 py-1 bg-red-50 border border-red-200 rounded-full text-xs font-medium text-red-700">
                                        {kw}
                                        <button type="button" onClick={() => setForm(f => ({ ...f, negative_keywords: f.negative_keywords.filter(k => k !== kw) }))} className="text-red-400 hover:text-red-600">
                                            <XCircle className="w-3 h-3" />
                                        </button>
                                    </span>
                                ))}
                            </div>
                        )}
                    </div>

                    <div className="flex justify-end gap-3 pt-2 border-t border-slate-100">
                        <button type="button" onClick={onClose} className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50">
                            Cancel
                        </button>
                        <button
                            type="submit"
                            disabled={saving || !!regexError}
                            className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        >
                            {saving && <RefreshCw className="w-4 h-4 animate-spin" />}
                            {saving ? 'Saving...' : initial ? 'Save Changes' : 'Create Pattern'}
                        </button>
                    </div>
                </form>
            </div>
        </div>
    );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function RegexSettingsPage() {
    const [patterns, setPatterns] = useState<CustomPattern[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [showCreate, setShowCreate] = useState(false);
    const [editTarget, setEditTarget] = useState<CustomPattern | null>(null);
    const [expandedId, setExpandedId] = useState<string | null>(null);
    const [statsId, setStatsId] = useState<string | null>(null);
    const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
    const fileInputRef = useRef<HTMLInputElement>(null);

    const loadPatterns = useCallback(async () => {
        setLoading(true);
        setError(null);
        try {
            const data = await apiGetPatterns();
            setPatterns(data);
        } catch {
            setError('Failed to load patterns.');
        } finally {
            setLoading(false);
        }
    }, []);

    useEffect(() => { loadPatterns(); }, [loadPatterns]);

    const handleCreate = async (formData: PatternFormData) => {
        await apiCreatePattern(formData);
        setShowCreate(false);
        await loadPatterns();
    };

    const handleEdit = async (formData: PatternFormData) => {
        if (!editTarget?.id) return;
        await apiUpdatePattern(editTarget.id, formData as Partial<CustomPattern>);
        setEditTarget(null);
        await loadPatterns();
    };

    const handleToggleActive = async (p: CustomPattern) => {
        await apiUpdatePattern(p.id, { is_active: !p.is_active });
        setPatterns(prev => prev.map(x => x.id === p.id ? { ...x, is_active: !x.is_active } : x));
    };

    const handleDelete = async (id: string) => {
        await apiDeletePattern(id);
        setDeleteConfirmId(null);
        setPatterns(prev => prev.filter(p => p.id !== id));
    };

    const handleExport = () => {
        const blob = new Blob([JSON.stringify(patterns, null, 2)], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `arc-hawk-patterns-${new Date().toISOString().slice(0, 10)}.json`;
        a.click();
        URL.revokeObjectURL(url);
    };

    const handleImport = (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0];
        if (!file) return;
        const reader = new FileReader();
        reader.onload = async (ev) => {
            try {
                const data = JSON.parse(ev.target?.result as string) as CustomPattern[];
                for (const p of data) {
                    const { id, created_at, match_count, fp_rate, ...rest } = p;
                    await apiCreatePattern(rest as PatternFormData);
                }
                await loadPatterns();
            } catch {
                alert('Invalid JSON file. Please export from ARC-Hawk to get the correct format.');
            }
        };
        reader.readAsText(file);
        // reset so same file can be re-imported
        e.target.value = '';
    };

    return (
        <div className="p-8 space-y-6 bg-white min-h-screen">
            {/* Header */}
            <div className="flex items-start justify-between gap-4">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900">Custom Regex Patterns</h1>
                    <p className="text-slate-500 mt-1 text-sm">
                        Manage custom PII detection patterns. Patterns are applied during scanning alongside built-in detectors.
                    </p>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                    <button
                        onClick={handleExport}
                        className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50"
                    >
                        <Download className="w-4 h-4" />
                        Export
                    </button>
                    <button
                        onClick={() => fileInputRef.current?.click()}
                        className="flex items-center gap-2 px-3 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50"
                    >
                        <Upload className="w-4 h-4" />
                        Import
                    </button>
                    <input ref={fileInputRef} type="file" accept=".json" onChange={handleImport} className="hidden" />
                    <button
                        onClick={() => setShowCreate(true)}
                        className="flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700"
                    >
                        <Plus className="w-4 h-4" />
                        Create Pattern
                    </button>
                </div>
            </div>

            {/* Table */}
            {loading ? (
                <div className="space-y-3">
                    {[...Array(5)].map((_, i) => (
                        <div key={i} className="h-14 bg-slate-100 rounded-lg animate-pulse" />
                    ))}
                </div>
            ) : error ? (
                <div className="p-6 text-center text-red-600 bg-red-50 rounded-lg border border-red-100">{error}</div>
            ) : patterns.length === 0 ? (
                <div className="p-12 text-center bg-slate-50 rounded-lg border border-slate-100 border-dashed">
                    <FlaskConical className="w-10 h-10 text-slate-300 mx-auto mb-3" />
                    <p className="text-slate-500 font-medium">No custom patterns yet</p>
                    <p className="text-slate-400 text-sm mt-1">Create your first pattern to extend PII detection.</p>
                    <button
                        onClick={() => setShowCreate(true)}
                        className="mt-4 flex items-center gap-2 px-4 py-2 text-sm font-medium bg-blue-600 text-white rounded-lg hover:bg-blue-700 mx-auto"
                    >
                        <Plus className="w-4 h-4" />
                        Create Pattern
                    </button>
                </div>
            ) : (
                <div className="border border-slate-200 rounded-lg overflow-hidden">
                    <table className="w-full text-sm">
                        <thead className="bg-slate-50 border-b border-slate-200">
                            <tr>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Name</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Pattern</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">PII Category</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Sensitivity</th>
                                <th className="text-left px-4 py-3 font-medium text-slate-600">Active</th>
                                <th className="text-right px-4 py-3 font-medium text-slate-600">Matches</th>
                                <th className="text-right px-4 py-3 font-medium text-slate-600">FP Rate</th>
                                <th className="text-right px-4 py-3 font-medium text-slate-600">Actions</th>
                            </tr>
                        </thead>
                        <tbody className="divide-y divide-slate-100">
                            {patterns.map(p => (
                                <React.Fragment key={p.id}>
                                    <tr className={`hover:bg-slate-50 transition-colors ${expandedId === p.id ? 'bg-blue-50/40' : ''}`}>
                                        <td className="px-4 py-3">
                                            <div className="font-medium text-slate-900">{p.display_name}</div>
                                            <div className="text-xs text-slate-400 font-mono">{p.name}</div>
                                        </td>
                                        <td className="px-4 py-3 max-w-48">
                                            <code className="text-xs font-mono text-slate-600 bg-slate-100 px-1.5 py-0.5 rounded truncate block" title={p.regex}>
                                                {p.regex}
                                            </code>
                                        </td>
                                        <td className="px-4 py-3">
                                            <span className="text-slate-700">{p.pii_category}</span>
                                        </td>
                                        <td className="px-4 py-3">
                                            <SensitivityBadge level={p.sensitivity} />
                                        </td>
                                        <td className="px-4 py-3">
                                            <Toggle checked={!!p.is_active} onChange={() => handleToggleActive(p)} />
                                        </td>
                                        <td className="px-4 py-3 text-right">
                                            <button
                                                onClick={() => setStatsId(p.id)}
                                                className="text-blue-600 hover:text-blue-800 font-medium hover:underline"
                                                title="View match frequency chart"
                                            >
                                                {p.match_count?.toLocaleString() ?? '—'}
                                            </button>
                                        </td>
                                        <td className="px-4 py-3 text-right">
                                            <div className="flex items-center justify-end gap-1">
                                                <span className={`text-sm font-medium ${p.fp_rate > 25 ? 'text-amber-600' : 'text-slate-700'}`}>
                                                    {p.fp_rate != null ? `${p.fp_rate.toFixed(1)}%` : '—'}
                                                </span>
                                                <FPWarningBadge rate={p.fp_rate ?? 0} />
                                            </div>
                                        </td>
                                        <td className="px-4 py-3">
                                            <div className="flex items-center justify-end gap-1">
                                                <button
                                                    onClick={() => setExpandedId(expandedId === p.id ? null : p.id)}
                                                    className="p-1.5 text-slate-400 hover:text-blue-600 hover:bg-blue-50 rounded-md"
                                                    title="Toggle test runner"
                                                >
                                                    <FlaskConical className="w-4 h-4" />
                                                </button>
                                                <button
                                                    onClick={() => setEditTarget(p)}
                                                    className="p-1.5 text-slate-400 hover:text-slate-700 hover:bg-slate-100 rounded-md"
                                                    title="Edit pattern"
                                                >
                                                    <Edit2 className="w-4 h-4" />
                                                </button>
                                                <button
                                                    onClick={() => setDeleteConfirmId(p.id)}
                                                    className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md"
                                                    title="Delete pattern"
                                                >
                                                    <Trash2 className="w-4 h-4" />
                                                </button>
                                            </div>
                                        </td>
                                    </tr>

                                    {/* Expanded: inline test runner */}
                                    {expandedId === p.id && (
                                        <tr>
                                            <td colSpan={8} className="px-6 pb-4 bg-slate-50/60">
                                                <TestRunner pattern={p} />
                                            </td>
                                        </tr>
                                    )}

                                    {/* Delete confirm inline */}
                                    {deleteConfirmId === p.id && (
                                        <tr>
                                            <td colSpan={8} className="px-4 py-3 bg-red-50 border-t border-red-100">
                                                <div className="flex items-center gap-4 text-sm">
                                                    <AlertTriangle className="w-4 h-4 text-red-500 shrink-0" />
                                                    <span className="text-red-700 font-medium flex-1">
                                                        Delete <strong>{p.display_name}</strong>? This cannot be undone.
                                                    </span>
                                                    <button
                                                        onClick={() => setDeleteConfirmId(null)}
                                                        className="px-3 py-1.5 text-sm font-medium text-slate-600 bg-white border border-slate-200 rounded-lg hover:bg-slate-50"
                                                    >
                                                        Cancel
                                                    </button>
                                                    <button
                                                        onClick={() => handleDelete(p.id)}
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

            {/* Stats Chart Modal */}
            {statsId && <StatsChart patternId={statsId} onClose={() => setStatsId(null)} />}

            {/* Create Modal */}
            {showCreate && <PatternModal onSave={handleCreate} onClose={() => setShowCreate(false)} />}

            {/* Edit Modal */}
            {editTarget && <PatternModal initial={editTarget} onSave={handleEdit} onClose={() => setEditTarget(null)} />}
        </div>
    );
}
