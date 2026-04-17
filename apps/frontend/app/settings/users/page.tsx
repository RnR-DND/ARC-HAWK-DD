'use client';

import React, { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Users, Shield, CheckCircle, XCircle, Loader2, ArrowLeft } from 'lucide-react';
import { get, put } from '@/utils/api-client';

interface User {
    id: string;
    name: string;
    email: string;
    role: string;
    active: boolean;
    created_at: string;
}

export default function UserManagementPage() {
    const router = useRouter();
    const [users, setUsers] = useState<User[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [togglingId, setTogglingId] = useState<string | null>(null);

    useEffect(() => {
        fetchUsers();
    }, []);

    const fetchUsers = async () => {
        try {
            setLoading(true);
            setError(null);
            const data = await get<any>('/auth/users');
            const list: User[] = Array.isArray(data)
                ? data
                : Array.isArray(data?.users)
                ? data.users
                : [];
            setUsers(list);
        } catch (err: any) {
            if (err?.response?.status === 403 || err?.response?.status === 401) {
                setError('Access denied. Admin role required to view user management.');
            } else {
                setError('Failed to load users.');
            }
        } finally {
            setLoading(false);
        }
    };

    const handleToggleActive = async (userId: string, currentActive: boolean) => {
        setTogglingId(userId);
        try {
            await put<any>(`/auth/users/${userId}`, { active: !currentActive });
            setUsers(prev =>
                prev.map(u => u.id === userId ? { ...u, active: !currentActive } : u)
            );
        } catch (err) {
            console.error('Failed to toggle user status:', err);
        } finally {
            setTogglingId(null);
        }
    };

    const roleColors: Record<string, string> = {
        admin: 'bg-purple-50 text-purple-700 border-purple-200',
        user: 'bg-blue-50 text-blue-700 border-blue-200',
        viewer: 'bg-slate-50 text-slate-600 border-slate-200',
    };

    return (
        <div className="min-h-screen bg-slate-50">
            <div className="max-w-5xl mx-auto px-6 py-8">
                <button
                    onClick={() => router.push('/settings')}
                    className="flex items-center gap-2 text-slate-500 hover:text-slate-900 mb-6 text-sm font-medium transition-colors group"
                >
                    <ArrowLeft className="w-4 h-4 group-hover:-translate-x-1 transition-transform" />
                    Back to Settings
                </button>

                <div className="flex items-center gap-3 mb-8">
                    <div className="p-2 bg-purple-50 rounded-lg border border-purple-100">
                        <Users className="w-6 h-6 text-purple-600" />
                    </div>
                    <div>
                        <h1 className="text-2xl font-bold text-slate-900">User Management</h1>
                        <p className="text-slate-500 text-sm">Manage system users, roles, and access status</p>
                    </div>
                    <div className="ml-auto flex items-center gap-2 px-3 py-1 bg-amber-50 border border-amber-200 rounded-lg">
                        <Shield className="w-4 h-4 text-amber-600" />
                        <span className="text-xs font-semibold text-amber-700">Admin Only</span>
                    </div>
                </div>

                {loading ? (
                    <div className="flex items-center justify-center py-24 text-slate-500">
                        <Loader2 className="w-8 h-8 animate-spin mr-3" />
                        <span>Loading users...</span>
                    </div>
                ) : error ? (
                    <div className="flex flex-col items-center justify-center py-24">
                        <div className="w-14 h-14 bg-red-50 rounded-full flex items-center justify-center mb-4">
                            <Shield className="w-7 h-7 text-red-500" />
                        </div>
                        <p className="text-slate-700 font-medium mb-1">Access Error</p>
                        <p className="text-slate-500 text-sm">{error}</p>
                    </div>
                ) : (
                    <div className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden">
                        <div className="px-6 py-4 border-b border-slate-100 flex items-center justify-between">
                            <span className="text-sm font-semibold text-slate-700">{users.length} user{users.length !== 1 ? 's' : ''}</span>
                        </div>
                        {users.length === 0 ? (
                            <div className="flex flex-col items-center justify-center py-16 text-slate-400">
                                <Users className="w-12 h-12 mb-3 opacity-20" />
                                <p className="text-sm">No users found.</p>
                            </div>
                        ) : (
                            <table className="w-full text-sm text-left">
                                <thead className="bg-slate-50 text-xs text-slate-500 uppercase tracking-wider">
                                    <tr>
                                        <th className="px-6 py-3 font-semibold">Name</th>
                                        <th className="px-6 py-3 font-semibold">Email</th>
                                        <th className="px-6 py-3 font-semibold">Role</th>
                                        <th className="px-6 py-3 font-semibold">Status</th>
                                        <th className="px-6 py-3 font-semibold">Created</th>
                                        <th className="px-6 py-3 font-semibold text-right">Actions</th>
                                    </tr>
                                </thead>
                                <tbody className="divide-y divide-slate-100">
                                    {users.map(user => (
                                        <tr key={user.id} className="hover:bg-slate-50 transition-colors">
                                            <td className="px-6 py-4 font-medium text-slate-900">{user.name || '—'}</td>
                                            <td className="px-6 py-4 text-slate-600 font-mono text-xs">{user.email}</td>
                                            <td className="px-6 py-4">
                                                <span className={`px-2.5 py-0.5 rounded-full text-xs font-semibold border capitalize ${roleColors[user.role?.toLowerCase()] ?? 'bg-slate-50 text-slate-600 border-slate-200'}`}>
                                                    {user.role || 'user'}
                                                </span>
                                            </td>
                                            <td className="px-6 py-4">
                                                <div className="flex items-center gap-1.5">
                                                    {user.active ? (
                                                        <CheckCircle className="w-4 h-4 text-green-500" />
                                                    ) : (
                                                        <XCircle className="w-4 h-4 text-slate-400" />
                                                    )}
                                                    <span className={`text-xs font-semibold ${user.active ? 'text-green-700' : 'text-slate-400'}`}>
                                                        {user.active ? 'Active' : 'Inactive'}
                                                    </span>
                                                </div>
                                            </td>
                                            <td className="px-6 py-4 text-slate-500 text-xs">
                                                {user.created_at ? new Date(user.created_at).toLocaleDateString() : '—'}
                                            </td>
                                            <td className="px-6 py-4 text-right">
                                                <button
                                                    onClick={() => handleToggleActive(user.id, user.active)}
                                                    disabled={togglingId === user.id}
                                                    className={`px-3 py-1.5 rounded-lg text-xs font-semibold border transition-colors disabled:opacity-50 ${
                                                        user.active
                                                            ? 'border-red-200 text-red-600 hover:bg-red-50'
                                                            : 'border-green-200 text-green-600 hover:bg-green-50'
                                                    }`}
                                                >
                                                    {togglingId === user.id ? (
                                                        <Loader2 className="w-3 h-3 animate-spin inline" />
                                                    ) : user.active ? 'Deactivate' : 'Activate'}
                                                </button>
                                            </td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
                        )}
                    </div>
                )}
            </div>
        </div>
    );
}
