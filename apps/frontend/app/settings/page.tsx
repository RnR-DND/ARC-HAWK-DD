'use client';

import React, { useState, useEffect } from 'react';
import { Settings, Shield, Bell, Database, Users, Key, Save, RefreshCw } from 'lucide-react';
import { settingsApi } from '@/services/settings.api';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { UserSettings } from '@/types';

interface SettingSection {
    id: string;
    title: string;
    description: string;
    icon: React.ReactNode;
    settings: SettingItem[];
}

interface SettingItem {
    id: keyof UserSettings;
    label: string;
    description: string;
    type: 'toggle' | 'select' | 'input' | 'textarea';
    value: any;
    options?: { value: string; label: string }[];
}

export default function SettingsPage() {
    const [loading, setLoading] = useState(true);
    const [settings, setSettings] = useState<UserSettings>({
        // Security Settings
        enableJWT: true,
        sessionTimeout: '3600',
        passwordPolicy: 'strong',
        twoFactorEnabled: false,

        // Scanner Settings
        scanFrequency: 'daily',
        maxFileSize: '100',
        supportedFormats: ['json', 'csv', 'xml', 'sql'],
        enableDeepScan: true,

        // Notification Settings
        emailNotifications: true,
        slackNotifications: false,
        criticalAlertsOnly: false,
        weeklyReports: true,

        // Data Retention
        logRetention: '90',
        scanHistoryRetention: '365',
        backupFrequency: 'weekly'
    });

    const [hasUnsavedChanges, setHasUnsavedChanges] = useState(false);
    const [saving, setSaving] = useState(false);

    useEffect(() => {
        const fetchSettings = async () => {
            try {
                const data = await settingsApi.getSettings();
                if (data && Object.keys(data).length > 0) {
                    setSettings(prev => ({
                        ...prev,
                        ...data
                    }));
                }
            } catch (error) {
                console.error('Failed to load settings:', error);
            } finally {
                setLoading(false);
            }
        };

        fetchSettings();
    }, []);

    const settingSections: SettingSection[] = [
        {
            id: 'security',
            title: 'Security Configuration',
            description: 'Authentication, authorization, and access control settings',
            icon: <Shield className="w-5 h-5" />,
            settings: [
                {
                    id: 'enableJWT',
                    label: 'Enable JWT Authentication',
                    description: 'Use JSON Web Tokens for API authentication',
                    type: 'toggle',
                    value: settings.enableJWT
                },
                {
                    id: 'sessionTimeout',
                    label: 'Session Timeout (seconds)',
                    description: 'Maximum session duration before requiring re-authentication',
                    type: 'select',
                    value: settings.sessionTimeout,
                    options: [
                        { value: '1800', label: '30 minutes' },
                        { value: '3600', label: '1 hour' },
                        { value: '7200', label: '2 hours' },
                        { value: '86400', label: '24 hours' }
                    ]
                },
                {
                    id: 'passwordPolicy',
                    label: 'Password Policy',
                    description: 'Password complexity requirements',
                    type: 'select',
                    value: settings.passwordPolicy,
                    options: [
                        { value: 'basic', label: 'Basic (8+ characters)' },
                        { value: 'strong', label: 'Strong (12+ chars, mixed case, numbers)' },
                        { value: 'complex', label: 'Complex (16+ chars, special characters)' }
                    ]
                },
                {
                    id: 'twoFactorEnabled',
                    label: 'Enable Two-Factor Authentication',
                    description: 'Require 2FA for all user accounts',
                    type: 'toggle',
                    value: settings.twoFactorEnabled
                }
            ]
        },
        {
            id: 'scanner',
            title: 'Scanner Configuration',
            description: 'PII detection engine settings and scan parameters',
            icon: <Database className="w-5 h-5" />,
            settings: [
                {
                    id: 'scanFrequency',
                    label: 'Default Scan Frequency',
                    description: 'How often to run automated scans',
                    type: 'select',
                    value: settings.scanFrequency,
                    options: [
                        { value: 'hourly', label: 'Every hour' },
                        { value: 'daily', label: 'Daily' },
                        { value: 'weekly', label: 'Weekly' },
                        { value: 'manual', label: 'Manual only' }
                    ]
                },
                {
                    id: 'maxFileSize',
                    label: 'Maximum File Size (MB)',
                    description: 'Skip files larger than this size during scanning',
                    type: 'input',
                    value: settings.maxFileSize
                },
                {
                    id: 'enableDeepScan',
                    label: 'Enable Deep Content Analysis',
                    description: 'Perform detailed analysis of file contents (slower but more accurate)',
                    type: 'toggle',
                    value: settings.enableDeepScan
                }
            ]
        },
        {
            id: 'notifications',
            title: 'Notification Settings',
            description: 'Configure alerts and reporting preferences',
            icon: <Bell className="w-5 h-5" />,
            settings: [
                {
                    id: 'emailNotifications',
                    label: 'Email Notifications',
                    description: 'Send alerts via email',
                    type: 'toggle',
                    value: settings.emailNotifications
                },
                {
                    id: 'slackNotifications',
                    label: 'Slack Integration',
                    description: 'Send alerts to Slack channels',
                    type: 'toggle',
                    value: settings.slackNotifications
                },
                {
                    id: 'criticalAlertsOnly',
                    label: 'Critical Alerts Only',
                    description: 'Only send notifications for critical findings',
                    type: 'toggle',
                    value: settings.criticalAlertsOnly
                },
                {
                    id: 'weeklyReports',
                    label: 'Weekly Summary Reports',
                    description: 'Send weekly compliance and activity reports',
                    type: 'toggle',
                    value: settings.weeklyReports
                }
            ]
        },
        {
            id: 'retention',
            title: 'Data Retention',
            description: 'Configure how long to keep logs, scans, and backups',
            icon: <RefreshCw className="w-5 h-5" />,
            settings: [
                {
                    id: 'logRetention',
                    label: 'Log Retention (days)',
                    description: 'How long to keep system and audit logs',
                    type: 'input',
                    value: settings.logRetention
                },
                {
                    id: 'scanHistoryRetention',
                    label: 'Scan History (days)',
                    description: 'How long to keep scan results and findings',
                    type: 'input',
                    value: settings.scanHistoryRetention
                },
                {
                    id: 'backupFrequency',
                    label: 'Backup Frequency',
                    description: 'How often to create system backups',
                    type: 'select',
                    value: settings.backupFrequency,
                    options: [
                        { value: 'daily', label: 'Daily' },
                        { value: 'weekly', label: 'Weekly' },
                        { value: 'monthly', label: 'Monthly' }
                    ]
                }
            ]
        }
    ];

    const handleSettingChange = (settingId: keyof UserSettings, value: any) => {
        setSettings(prev => ({ ...prev, [settingId]: value }));
        setHasUnsavedChanges(true);
    };

    const handleSaveSettings = async () => {
        setSaving(true);
        try {
            await settingsApi.updateSettings(settings);
            setHasUnsavedChanges(false);
        } catch (error) {
            console.error('Failed to save settings:', error);
        } finally {
            setSaving(false);
        }
    };

    const renderSettingInput = (setting: SettingItem) => {
        switch (setting.type) {
            case 'toggle':
                return (
                    <label className="flex items-center cursor-pointer">
                        <input
                            type="checkbox"
                            checked={setting.value}
                            onChange={(e) => handleSettingChange(setting.id, e.target.checked)}
                            className="sr-only peer"
                        />
                        <div className="relative w-11 h-6 bg-slate-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full rtl:peer-checked:after:-translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:start-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
                        <span className="ms-3 text-sm font-medium text-slate-900">
                            {setting.value ? 'Enabled' : 'Disabled'}
                        </span>
                    </label>
                );
            case 'select':
                return (
                    <select
                        value={setting.value}
                        onChange={(e) => handleSettingChange(setting.id, e.target.value)}
                        className="bg-white border border-slate-300 text-slate-900 text-sm rounded-lg focus:ring-blue-500 focus:border-blue-500 block w-full p-2.5 max-w-[200px]"
                    >
                        {setting.options?.map(option => (
                            <option key={option.value} value={option.value}>
                                {option.label}
                            </option>
                        ))}
                    </select>
                );
            case 'input':
                return (
                    <Input
                        type="text"
                        value={setting.value}
                        onChange={(e) => handleSettingChange(setting.id, e.target.value)}
                        className="max-w-[200px]"
                    />
                );
            default:
                return null;
        }
    };

    return (
        <div className="min-h-screen bg-white">

            <div className="container mx-auto px-4 py-8 max-w-6xl">
                {/* Header */}
                <div className="flex justify-between items-center mb-8">
                    <div>
                        <h1 className="text-3xl font-bold text-slate-900 mb-2 tracking-tight">
                            System Settings
                        </h1>
                        <p className="text-slate-500 text-base">
                            Configure ARC-Hawk system behavior, security policies, and preferences
                        </p>
                    </div>
                    {hasUnsavedChanges && (
                        <Button
                            onClick={handleSaveSettings}
                            disabled={saving}
                            className={`${saving ? 'bg-slate-400' : 'bg-blue-600 hover:bg-blue-700'}`}
                        >
                            {saving ? (
                                <>
                                    <RefreshCw className="w-4 h-4 mr-2 animate-spin" />
                                    Saving...
                                </>
                            ) : (
                                <>
                                    <Save className="w-4 h-4 mr-2" />
                                    Save Changes
                                </>
                            )}
                        </Button>
                    )}
                </div>

                {/* Settings Sections */}
                <div className="space-y-6">
                    {settingSections.map(section => (
                        <Card key={section.id}>
                            <CardHeader className="pb-4 border-b border-border">
                                <div className="flex items-center gap-3">
                                    <div className="p-2 bg-blue-50 rounded-lg text-blue-600 border border-blue-100">
                                        {section.icon}
                                    </div>
                                    <div>
                                        <CardTitle>{section.title}</CardTitle>
                                        <CardDescription>{section.description}</CardDescription>
                                    </div>
                                </div>
                            </CardHeader>
                            <CardContent className="pt-6 space-y-6">
                                {section.settings.map((setting, idx) => (
                                    <div key={setting.id}>
                                        <div className="flex justify-between items-start py-4">
                                            <div className="space-y-1 flex-1 mr-6">
                                                <h3 className="text-sm font-medium leading-none text-slate-900">
                                                    {setting.label}
                                                </h3>
                                                <p className="text-sm text-slate-500">
                                                    {setting.description}
                                                </p>
                                            </div>
                                            <div className="flex-shrink-0">
                                                {renderSettingInput(setting)}
                                            </div>
                                        </div>
                                        {idx < section.settings.length - 1 && <Separator className="my-2" />}
                                    </div>
                                ))}
                            </CardContent>
                        </Card>
                    ))}
                </div>

                {/* System Information */}
                <Card className="mt-8">
                    <CardHeader>
                        <CardTitle className="text-lg">System Information</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-6">
                            <div>
                                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">
                                    Version
                                </div>
                                <div className="text-sm font-medium text-slate-900">
                                    ARC-Hawk v1.2.0
                                </div>
                            </div>
                            <div>
                                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">
                                    Last Updated
                                </div>
                                <div className="text-sm font-medium text-slate-900">
                                    January 15, 2026
                                </div>
                            </div>
                            <div>
                                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">
                                    Environment
                                </div>
                                <Badge variant="secondary" className="bg-emerald-100 text-emerald-700 hover:bg-emerald-100 border-emerald-200">
                                    Production
                                </Badge>
                            </div>
                            <div>
                                <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-1">
                                    License
                                </div>
                                <Badge variant="outline" className="text-blue-700 bg-blue-50 border-blue-200">
                                    Enterprise
                                </Badge>
                            </div>
                        </div>
                    </CardContent>
                </Card>
            </div>
        </div>
    );
}
