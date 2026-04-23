'use client';

import React, { useState, useEffect } from 'react';
import { Bell, Mail, Slack, Save, Plus, X, Loader2, CheckCircle, AlertTriangle } from 'lucide-react';
import { getNotificationSettings, saveNotificationSettings, type NotificationSettings } from '@/services/notifications.api';

const SEVERITIES = ['Critical', 'High', 'Medium', 'Low'] as const;

const DEFAULT: NotificationSettings = {
    email_enabled: false,
    email_recipients: [],
    slack_enabled: false,
    slack_webhook_url: '',
    notify_on_scan_complete: true,
    notify_on_high_severity: true,
    notify_on_stale_connector: false,
    severity_threshold: 'High',
};

export default function NotificationSettingsPage() {
    const [settings, setSettings] = useState<NotificationSettings>(DEFAULT);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [toast, setToast] = useState<{ type: 'success' | 'error'; msg: string } | null>(null);
    const [newEmail, setNewEmail] = useState('');

    useEffect(() => {
        getNotificationSettings()
            .then(s => setSettings(s))
            .catch(() => setSettings(DEFAULT))
            .finally(() => setLoading(false));
    }, []);

    const showToast = (type: 'success' | 'error', msg: string) => {
        setToast({ type, msg });
        setTimeout(() => setToast(null), 3000);
    };

    const handleSave = async () => {
        setSaving(true);
        try {
            const { id: _id, ...payload } = settings;
            await saveNotificationSettings(payload);
            showToast('success', 'Notification settings saved');
        } catch {
            showToast('error', 'Failed to save settings');
        } finally {
            setSaving(false);
        }
    };

    const addEmail = () => {
        const trimmed = newEmail.trim();
        if (!trimmed || !trimmed.includes('@')) return;
        if (settings.email_recipients.includes(trimmed)) return;
        setSettings(s => ({ ...s, email_recipients: [...s.email_recipients, trimmed] }));
        setNewEmail('');
    };

    const removeEmail = (email: string) => {
        setSettings(s => ({ ...s, email_recipients: s.email_recipients.filter(e => e !== email) }));
    };

    const toggle = (key: keyof NotificationSettings) => {
        setSettings(s => ({ ...s, [key]: !s[key as keyof typeof s] }));
    };

    if (loading) {
        return (
            <div className="p-8 max-w-3xl">
                {[1, 2, 3].map(i => <div key={i} className="h-32 bg-slate-100 rounded-xl animate-pulse mb-4" />)}
            </div>
        );
    }

    return (
        <div className="p-8 max-w-3xl space-y-6">
            {/* Toast */}
            {toast && (
                <div className={`fixed top-6 right-6 z-50 flex items-center gap-2 px-4 py-3 rounded-xl shadow-lg text-sm font-medium
                    ${toast.type === 'success' ? 'bg-emerald-50 text-emerald-800 border border-emerald-200' : 'bg-red-50 text-red-800 border border-red-200'}`}>
                    {toast.type === 'success' ? <CheckCircle className="w-4 h-4" /> : <AlertTriangle className="w-4 h-4" />}
                    {toast.msg}
                </div>
            )}

            {/* Header */}
            <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <div className="w-10 h-10 bg-blue-50 rounded-xl flex items-center justify-center">
                        <Bell className="w-5 h-5 text-blue-600" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900">Notification Settings</h1>
                        <p className="text-slate-500 text-sm mt-0.5">Configure alerts for scans, findings, and connector health</p>
                    </div>
                </div>
                <button
                    onClick={handleSave}
                    disabled={saving}
                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors disabled:opacity-50"
                >
                    {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Save className="w-4 h-4" />}
                    {saving ? 'Saving…' : 'Save Settings'}
                </button>
            </div>

            {/* Trigger Events */}
            <div className="bg-white border border-slate-200 rounded-xl p-6 space-y-4">
                <h2 className="text-sm font-semibold text-slate-700 uppercase tracking-wide">Notify When</h2>

                {[
                    { key: 'notify_on_scan_complete' as const, label: 'Scan completes', desc: 'Get notified when any scan finishes' },
                    { key: 'notify_on_high_severity' as const, label: 'High/Critical finding detected', desc: 'Alert on new findings at or above threshold' },
                    { key: 'notify_on_stale_connector' as const, label: 'Connector goes stale', desc: 'Alert when a connector has no scan for 72+ hours' },
                ].map(({ key, label, desc }) => (
                    <label key={key} className="flex items-center justify-between cursor-pointer group">
                        <div>
                            <div className="text-sm font-medium text-slate-800">{label}</div>
                            <div className="text-xs text-slate-500">{desc}</div>
                        </div>
                        <div
                            role="switch"
                            aria-checked={settings[key] as boolean}
                            onClick={() => toggle(key)}
                            className={`relative w-11 h-6 rounded-full transition-colors cursor-pointer
                                ${settings[key] ? 'bg-blue-600' : 'bg-slate-200'}`}
                        >
                            <span className={`absolute top-1 left-1 w-4 h-4 bg-white rounded-full shadow transition-transform
                                ${settings[key] ? 'translate-x-5' : 'translate-x-0'}`} />
                        </div>
                    </label>
                ))}

                <div>
                    <label className="block text-sm font-medium text-slate-700 mb-1">Severity Threshold</label>
                    <p className="text-xs text-slate-500 mb-2">Only notify for findings at this severity or above</p>
                    <div className="flex gap-2 flex-wrap">
                        {SEVERITIES.map(s => (
                            <button
                                key={s}
                                onClick={() => setSettings(prev => ({ ...prev, severity_threshold: s }))}
                                className={`px-3 py-1.5 rounded-lg text-sm font-medium border transition-colors
                                    ${settings.severity_threshold === s
                                        ? 'bg-blue-600 text-white border-blue-600'
                                        : 'bg-white text-slate-600 border-slate-200 hover:border-blue-300'}`}
                            >
                                {s}
                            </button>
                        ))}
                    </div>
                </div>
            </div>

            {/* Email */}
            <div className="bg-white border border-slate-200 rounded-xl p-6 space-y-4">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <Mail className="w-5 h-5 text-slate-500" />
                        <h2 className="text-sm font-semibold text-slate-700">Email Notifications</h2>
                    </div>
                    <div
                        role="switch"
                        aria-checked={settings.email_enabled}
                        onClick={() => toggle('email_enabled')}
                        className={`relative w-11 h-6 rounded-full transition-colors cursor-pointer
                            ${settings.email_enabled ? 'bg-blue-600' : 'bg-slate-200'}`}
                    >
                        <span className={`absolute top-1 left-1 w-4 h-4 bg-white rounded-full shadow transition-transform
                            ${settings.email_enabled ? 'translate-x-5' : 'translate-x-0'}`} />
                    </div>
                </div>

                {settings.email_enabled && (
                    <div className="space-y-3">
                        <div className="flex gap-2">
                            <input
                                type="email"
                                value={newEmail}
                                onChange={e => setNewEmail(e.target.value)}
                                onKeyDown={e => e.key === 'Enter' && addEmail()}
                                placeholder="email@example.com"
                                className="flex-1 border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                            />
                            <button
                                onClick={addEmail}
                                className="px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg transition-colors"
                            >
                                <Plus className="w-4 h-4" />
                            </button>
                        </div>
                        <div className="flex flex-wrap gap-2">
                            {settings.email_recipients.map(email => (
                                <span key={email} className="flex items-center gap-1.5 px-3 py-1 bg-blue-50 text-blue-800 rounded-full text-xs font-medium">
                                    {email}
                                    <button onClick={() => removeEmail(email)} className="hover:text-blue-600">
                                        <X className="w-3 h-3" />
                                    </button>
                                </span>
                            ))}
                            {settings.email_recipients.length === 0 && (
                                <span className="text-xs text-slate-400">No recipients added</span>
                            )}
                        </div>
                    </div>
                )}
            </div>

            {/* Slack */}
            <div className="bg-white border border-slate-200 rounded-xl p-6 space-y-4">
                <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                        <Slack className="w-5 h-5 text-slate-500" />
                        <h2 className="text-sm font-semibold text-slate-700">Slack Notifications</h2>
                    </div>
                    <div
                        role="switch"
                        aria-checked={settings.slack_enabled}
                        onClick={() => toggle('slack_enabled')}
                        className={`relative w-11 h-6 rounded-full transition-colors cursor-pointer
                            ${settings.slack_enabled ? 'bg-blue-600' : 'bg-slate-200'}`}
                    >
                        <span className={`absolute top-1 left-1 w-4 h-4 bg-white rounded-full shadow transition-transform
                            ${settings.slack_enabled ? 'translate-x-5' : 'translate-x-0'}`} />
                    </div>
                </div>

                {settings.slack_enabled && (
                    <div>
                        <label className="block text-xs font-medium text-slate-600 mb-1">Webhook URL</label>
                        <input
                            type="url"
                            value={settings.slack_webhook_url}
                            onChange={e => setSettings(s => ({ ...s, slack_webhook_url: e.target.value }))}
                            placeholder="https://hooks.slack.com/services/..."
                            className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        />
                        <p className="text-xs text-slate-400 mt-1">Create an Incoming Webhook in your Slack workspace settings.</p>
                    </div>
                )}
            </div>
        </div>
    );
}
