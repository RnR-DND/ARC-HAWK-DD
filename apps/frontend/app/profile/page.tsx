'use client';

import React, { useEffect, useState } from 'react';
import { Mail, Shield, Building, Calendar } from 'lucide-react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { authApi } from '@/services/auth.api';
import { User } from '@/types/api';

function getInitials(name: string): string {
    return name
        .split(' ')
        .map(n => n[0])
        .join('')
        .toUpperCase()
        .slice(0, 2);
}

function formatDate(dateStr?: string): string {
    if (!dateStr) return '—';
    try {
        return new Date(dateStr).toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
    } catch {
        return dateStr;
    }
}

export default function ProfilePage() {
    const [user, setUser] = useState<User | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        authApi.getProfile()
            .then(setUser)
            .catch(() => setError('Failed to load profile'))
            .finally(() => setLoading(false));
    }, []);

    const initials = user ? getInitials(user.name || user.email) : '??';
    const displayName = user?.name || user?.email || 'Unknown User';
    const displayEmail = user?.email || '—';
    const displayRole = user?.role || '—';

    return (
        <div className="p-8 max-w-3xl mx-auto">
            <h1 className="text-2xl font-bold text-slate-900 mb-8">Profile</h1>

            {/* Profile Card */}
            <div className="bg-white border border-slate-200 rounded-xl shadow-sm overflow-hidden">
                {/* Header Banner */}
                <div className="h-32 bg-gradient-to-r from-blue-600 to-purple-700" />

                {/* Avatar & Info */}
                <div className="px-8 pb-8">
                    <div className="flex items-end gap-6 -mt-12 mb-6">
                        <Avatar className="h-24 w-24 ring-4 ring-white shadow-lg">
                            <AvatarImage src="" />
                            <AvatarFallback className="bg-gradient-to-br from-blue-600 to-purple-700 text-white text-2xl font-bold">
                                {loading ? '…' : initials}
                            </AvatarFallback>
                        </Avatar>
                        <div className="pb-1">
                            {loading ? (
                                <>
                                    <div className="h-6 w-40 bg-slate-200 rounded animate-pulse mb-2" />
                                    <div className="h-4 w-32 bg-slate-100 rounded animate-pulse" />
                                </>
                            ) : (
                                <>
                                    <h2 className="text-xl font-bold text-slate-900">{displayName}</h2>
                                    <p className="text-sm text-slate-500 capitalize">{displayRole}</p>
                                </>
                            )}
                        </div>
                    </div>

                    {error && (
                        <div className="mb-6 px-4 py-3 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
                            {error}
                        </div>
                    )}

                    {/* Info Grid */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                            <div className="p-2 bg-blue-100 rounded-lg">
                                <Mail className="w-4 h-4 text-blue-600" />
                            </div>
                            <div>
                                <div className="text-xs text-slate-500 font-medium">Email</div>
                                {loading ? (
                                    <div className="h-4 w-36 bg-slate-200 rounded animate-pulse mt-1" />
                                ) : (
                                    <div className="text-sm font-semibold text-slate-900">{displayEmail}</div>
                                )}
                            </div>
                        </div>

                        <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                            <div className="p-2 bg-purple-100 rounded-lg">
                                <Shield className="w-4 h-4 text-purple-600" />
                            </div>
                            <div>
                                <div className="text-xs text-slate-500 font-medium">Role</div>
                                {loading ? (
                                    <div className="h-4 w-24 bg-slate-200 rounded animate-pulse mt-1" />
                                ) : (
                                    <div className="text-sm font-semibold text-slate-900 capitalize">{displayRole}</div>
                                )}
                            </div>
                        </div>

                        <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                            <div className="p-2 bg-green-100 rounded-lg">
                                <Building className="w-4 h-4 text-green-600" />
                            </div>
                            <div>
                                <div className="text-xs text-slate-500 font-medium">Tenant ID</div>
                                {loading ? (
                                    <div className="h-4 w-28 bg-slate-200 rounded animate-pulse mt-1" />
                                ) : (
                                    <div className="text-sm font-semibold text-slate-900 font-mono">
                                        {user?.tenant_id ? user.tenant_id.slice(0, 8) + '…' : '—'}
                                    </div>
                                )}
                            </div>
                        </div>

                        <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                            <div className="p-2 bg-orange-100 rounded-lg">
                                <Calendar className="w-4 h-4 text-orange-600" />
                            </div>
                            <div>
                                <div className="text-xs text-slate-500 font-medium">Member Since</div>
                                {loading ? (
                                    <div className="h-4 w-28 bg-slate-200 rounded animate-pulse mt-1" />
                                ) : (
                                    <div className="text-sm font-semibold text-slate-900">
                                        {formatDate(user?.created_at)}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
