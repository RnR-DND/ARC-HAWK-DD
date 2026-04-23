'use client';

import React, { useState, useEffect } from 'react';
import { X, Calendar, Clock, Trash2, ToggleLeft, ToggleRight, Plus, Loader2 } from 'lucide-react';
import { connectionsApi, type Connection } from '@/services/connections.api';
import {
    getSchedules,
    createSchedule,
    deleteSchedule,
    toggleSchedule,
    type ScanSchedule,
} from '@/services/schedules.api';

interface Props {
    isOpen: boolean;
    onClose: () => void;
}

const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const HOURS = Array.from({ length: 24 }, (_, i) => i);

function formatHour(h: number) {
    if (h === 0) return '12:00 AM';
    if (h < 12) return `${h}:00 AM`;
    if (h === 12) return '12:00 PM';
    return `${h - 12}:00 PM`;
}

function nextRunLabel(s: ScanSchedule) {
    if (!s.enabled) return 'Paused';
    if (!s.next_run_at) return '—';
    return new Date(s.next_run_at).toLocaleString();
}

export function ScanScheduleModal({ isOpen, onClose }: Props) {
    const [schedules, setSchedules] = useState<ScanSchedule[]>([]);
    const [connections, setConnections] = useState<Connection[]>([]);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const [profileName, setProfileName] = useState('');
    const [frequency, setFrequency] = useState<'daily' | 'weekly' | 'monthly'>('daily');
    const [hour, setHour] = useState(2);
    const [dayOfWeek, setDayOfWeek] = useState(1);
    const [dayOfMonth, setDayOfMonth] = useState(1);

    useEffect(() => {
        if (!isOpen) return;
        setLoading(true);
        Promise.all([getSchedules(), connectionsApi.getConnections()])
            .then(([s, c]) => {
                setSchedules(s);
                setConnections(c.connections);
                if (c.connections.length > 0 && !profileName) {
                    setProfileName(c.connections[0].profile_name);
                }
            })
            .catch(() => setError('Failed to load data'))
            .finally(() => setLoading(false));
    }, [isOpen]);

    if (!isOpen) return null;

    const handleCreate = async () => {
        if (!profileName) return;
        setSaving(true);
        setError(null);
        try {
            await createSchedule({
                profile_name: profileName,
                frequency,
                hour,
                day_of_week: frequency === 'weekly' ? dayOfWeek : undefined,
                day_of_month: frequency === 'monthly' ? dayOfMonth : undefined,
            });
            const updated = await getSchedules();
            setSchedules(updated);
        } catch {
            setError('Failed to save schedule');
        } finally {
            setSaving(false);
        }
    };

    const handleDelete = async (id: string) => {
        try {
            await deleteSchedule(id);
            setSchedules(s => s.filter(x => x.id !== id));
        } catch {
            setError('Failed to delete schedule');
        }
    };

    const handleToggle = async (id: string) => {
        try {
            const { enabled } = await toggleSchedule(id);
            setSchedules(s => s.map(x => x.id === id ? { ...x, enabled } : x));
        } catch {
            setError('Failed to toggle schedule');
        }
    };

    return (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 backdrop-blur-sm">
            <div className="bg-white rounded-2xl shadow-2xl w-full max-w-2xl mx-4 flex flex-col max-h-[90vh]">
                {/* Header */}
                <div className="flex items-center justify-between px-6 py-4 border-b border-slate-200">
                    <div className="flex items-center gap-3">
                        <div className="w-9 h-9 bg-blue-50 rounded-lg flex items-center justify-center">
                            <Calendar className="w-5 h-5 text-blue-600" />
                        </div>
                        <div>
                            <h2 className="text-lg font-bold text-slate-900">Scan Schedules</h2>
                            <p className="text-xs text-slate-500">Configure recurring scans per connector</p>
                        </div>
                    </div>
                    <button onClick={onClose} className="p-2 hover:bg-slate-100 rounded-lg transition-colors">
                        <X className="w-5 h-5 text-slate-500" />
                    </button>
                </div>

                <div className="flex-1 overflow-y-auto p-6 space-y-6">
                    {error && (
                        <div className="bg-red-50 border border-red-200 text-red-700 rounded-lg px-4 py-2 text-sm">
                            {error}
                        </div>
                    )}

                    {/* Create form */}
                    <div className="bg-slate-50 rounded-xl p-4 space-y-4 border border-slate-200">
                        <h3 className="text-sm font-semibold text-slate-700 flex items-center gap-2">
                            <Plus className="w-4 h-4" /> New Schedule
                        </h3>

                        <div className="grid grid-cols-2 gap-3">
                            <div>
                                <label className="block text-xs font-medium text-slate-600 mb-1">Connector</label>
                                <select
                                    value={profileName}
                                    onChange={e => setProfileName(e.target.value)}
                                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                                >
                                    {connections.map(c => (
                                        <option key={c.id} value={c.profile_name}>{c.profile_name}</option>
                                    ))}
                                </select>
                            </div>

                            <div>
                                <label className="block text-xs font-medium text-slate-600 mb-1">Frequency</label>
                                <select
                                    value={frequency}
                                    onChange={e => setFrequency(e.target.value as 'daily' | 'weekly' | 'monthly')}
                                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                                >
                                    <option value="daily">Daily</option>
                                    <option value="weekly">Weekly</option>
                                    <option value="monthly">Monthly</option>
                                </select>
                            </div>
                        </div>

                        <div className="grid grid-cols-2 gap-3">
                            <div>
                                <label className="block text-xs font-medium text-slate-600 mb-1 flex items-center gap-1">
                                    <Clock className="w-3 h-3" /> Hour (UTC)
                                </label>
                                <select
                                    value={hour}
                                    onChange={e => setHour(Number(e.target.value))}
                                    className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                                >
                                    {HOURS.map(h => (
                                        <option key={h} value={h}>{formatHour(h)}</option>
                                    ))}
                                </select>
                            </div>

                            {frequency === 'weekly' && (
                                <div>
                                    <label className="block text-xs font-medium text-slate-600 mb-1">Day of Week</label>
                                    <select
                                        value={dayOfWeek}
                                        onChange={e => setDayOfWeek(Number(e.target.value))}
                                        className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm bg-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                                    >
                                        {DAYS.map((d, i) => <option key={i} value={i}>{d}</option>)}
                                    </select>
                                </div>
                            )}

                            {frequency === 'monthly' && (
                                <div>
                                    <label className="block text-xs font-medium text-slate-600 mb-1">Day of Month</label>
                                    <input
                                        type="number"
                                        min={1}
                                        max={28}
                                        value={dayOfMonth}
                                        onChange={e => setDayOfMonth(Number(e.target.value))}
                                        className="w-full border border-slate-200 rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                                    />
                                </div>
                            )}
                        </div>

                        <button
                            onClick={handleCreate}
                            disabled={saving || !profileName}
                            className="w-full py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-colors disabled:opacity-50 flex items-center justify-center gap-2"
                        >
                            {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                            {saving ? 'Saving…' : 'Create Schedule'}
                        </button>
                    </div>

                    {/* Existing schedules */}
                    <div className="space-y-2">
                        <h3 className="text-sm font-semibold text-slate-700">Active Schedules</h3>
                        {loading ? (
                            <div className="space-y-2">
                                {[1, 2].map(i => <div key={i} className="h-16 bg-slate-100 rounded-xl animate-pulse" />)}
                            </div>
                        ) : schedules.length === 0 ? (
                            <div className="text-center py-8 text-slate-400 text-sm">No schedules configured yet.</div>
                        ) : (
                            schedules.map(s => (
                                <div key={s.id} className="flex items-center justify-between p-4 bg-white border border-slate-200 rounded-xl">
                                    <div className="flex-1 min-w-0">
                                        <div className="flex items-center gap-2">
                                            <span className={`w-2 h-2 rounded-full shrink-0 ${s.enabled ? 'bg-emerald-500' : 'bg-slate-300'}`} />
                                            <span className="font-medium text-slate-800 text-sm truncate">{s.profile_name}</span>
                                            <span className="text-xs px-2 py-0.5 bg-blue-50 text-blue-700 rounded-full capitalize">{s.frequency}</span>
                                        </div>
                                        <div className="text-xs text-slate-500 mt-1 ml-4">
                                            {formatHour(s.hour)} UTC · Next: {nextRunLabel(s)}
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-1 ml-4">
                                        <button
                                            onClick={() => handleToggle(s.id)}
                                            title={s.enabled ? 'Pause' : 'Enable'}
                                            className="p-1.5 hover:bg-slate-100 rounded-lg transition-colors text-slate-500"
                                        >
                                            {s.enabled
                                                ? <ToggleRight className="w-5 h-5 text-emerald-500" />
                                                : <ToggleLeft className="w-5 h-5" />}
                                        </button>
                                        <button
                                            onClick={() => handleDelete(s.id)}
                                            className="p-1.5 hover:bg-red-50 rounded-lg transition-colors text-slate-400 hover:text-red-500"
                                        >
                                            <Trash2 className="w-4 h-4" />
                                        </button>
                                    </div>
                                </div>
                            ))
                        )}
                    </div>
                </div>
            </div>
        </div>
    );
}
