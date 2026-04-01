'use client';

import React from 'react';
import { User, Mail, Shield, Building, Calendar } from 'lucide-react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';

export default function ProfilePage() {
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
                                AU
                            </AvatarFallback>
                        </Avatar>
                        <div className="pb-1">
                            <h2 className="text-xl font-bold text-slate-900">Admin User</h2>
                            <p className="text-sm text-slate-500">Platform Administrator</p>
                        </div>
                    </div>

                    {/* Info Grid */}
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                            <div className="p-2 bg-blue-100 rounded-lg">
                                <Mail className="w-4 h-4 text-blue-600" />
                            </div>
                            <div>
                                <div className="text-xs text-slate-500 font-medium">Email</div>
                                <div className="text-sm font-semibold text-slate-900">admin@company.com</div>
                            </div>
                        </div>

                        <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                            <div className="p-2 bg-purple-100 rounded-lg">
                                <Shield className="w-4 h-4 text-purple-600" />
                            </div>
                            <div>
                                <div className="text-xs text-slate-500 font-medium">Role</div>
                                <div className="text-sm font-semibold text-slate-900">Administrator</div>
                            </div>
                        </div>

                        <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                            <div className="p-2 bg-green-100 rounded-lg">
                                <Building className="w-4 h-4 text-green-600" />
                            </div>
                            <div>
                                <div className="text-xs text-slate-500 font-medium">Organization</div>
                                <div className="text-sm font-semibold text-slate-900">ARC Platform</div>
                            </div>
                        </div>

                        <div className="flex items-center gap-3 p-4 bg-slate-50 rounded-lg">
                            <div className="p-2 bg-orange-100 rounded-lg">
                                <Calendar className="w-4 h-4 text-orange-600" />
                            </div>
                            <div>
                                <div className="text-xs text-slate-500 font-medium">Member Since</div>
                                <div className="text-sm font-semibold text-slate-900">January 2026</div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
