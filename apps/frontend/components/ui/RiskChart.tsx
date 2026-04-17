import React from 'react';
import { motion } from 'framer-motion';
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, Legend } from 'recharts';
import { TrendingUp, AlertTriangle, Shield, Database } from 'lucide-react';
import { theme } from '@/design-system/theme';

interface RiskChartProps {
    byPiiType: Record<string, number>;
    byAsset: Record<string, number>;
    byConfidence: Record<string, number>;
    loading?: boolean;
}

const riskColors = {
    Critical: theme.colors.risk.critical,
    High: theme.colors.risk.high,
    Medium: theme.colors.risk.medium,
    Low: theme.colors.risk.low,
    Info: theme.colors.risk.info,
};

const piiTypeColors = [
    '#7c3aed', '#0891b2', '#059669', '#d97706', '#dc2626',
    '#2563eb', '#db2777', '#65a30d', '#ea580c', '#4f46e5'
];

export default function RiskChart({ byPiiType, byAsset, byConfidence, loading = false }: RiskChartProps) {
    const piiTypeData = Object.entries(byPiiType).map(([type, count], index) => ({
        name: type,
        value: count,
        color: piiTypeColors[index % piiTypeColors.length]
    }));

    const assetData = Object.entries(byAsset).slice(0, 8).map(([asset, count]) => ({
        name: asset.length > 20 ? asset.substring(0, 20) + '...' : asset,
        findings: count
    }));

    const confidenceData = Object.entries(byConfidence).map(([level, count]) => ({
        level,
        count,
        color: riskColors[level as keyof typeof riskColors] || riskColors.Info
    }));

    if (loading) {
        return (
            <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
                <div className="animate-pulse space-y-4">
                    <div className="h-6 w-32 bg-slate-100 rounded" />
                    <div className="h-64 bg-slate-50 rounded" />
                </div>
            </div>
        );
    }

    const totalFindings = Object.values(byPiiType).reduce((sum, count) => sum + count, 0);
    const highRiskCount = Object.values(byConfidence).reduce((sum, count, index) => {
        const level = Object.keys(byConfidence)[index];
        return level.includes('> 90') || level.includes('70-90') ? sum + count : sum;
    }, 0);

    return (
        <div className="space-y-6">
            {/* Risk Summary Cards */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                className="grid grid-cols-1 sm:grid-cols-3 gap-4"
            >
                <div className="bg-gradient-to-br from-red-50 to-red-100/50 border border-red-200 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-red-600 text-sm font-medium">High Risk</p>
                            <p className="text-red-900 text-2xl font-bold">{highRiskCount}</p>
                        </div>
                        <AlertTriangle className="w-8 h-8 text-red-500" />
                    </div>
                </div>

                <div className="bg-gradient-to-br from-blue-50 to-blue-100/50 border border-blue-200 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-blue-600 text-sm font-medium">Total Findings</p>
                            <p className="text-blue-900 text-2xl font-bold">{totalFindings}</p>
                        </div>
                        <Shield className="w-8 h-8 text-blue-500" />
                    </div>
                </div>

                <div className="bg-gradient-to-br from-emerald-50 to-emerald-100/50 border border-emerald-200 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-emerald-600 text-sm font-medium">Data Sources</p>
                            <p className="text-emerald-900 text-2xl font-bold">{Object.keys(byAsset).length}</p>
                        </div>
                        <Database className="w-8 h-8 text-emerald-500" />
                    </div>
                </div>
            </motion.div>

            {/* PII Type Distribution */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.1 }}
                className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm"
            >
                <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-purple-100/50 rounded-lg border border-purple-100">
                            <Shield className="w-5 h-5 text-purple-600" />
                        </div>
                        <div>
                            <h3 className="text-lg font-bold text-slate-900">PII Type Distribution</h3>
                            <p className="text-slate-600 text-sm">Breakdown by sensitive data types</p>
                        </div>
                    </div>
                    <div className="text-right">
                        <p className="text-slate-600 text-sm">Total Types</p>
                        <p className="text-slate-950 text-lg font-semibold">{piiTypeData.length}</p>
                    </div>
                </div>

                <div className="h-64">
                    <ResponsiveContainer width="100%" height={256} minWidth={0}>
                        <PieChart>
                            <Pie
                                data={piiTypeData}
                                cx="50%"
                                cy="50%"
                                innerRadius={60}
                                outerRadius={100}
                                paddingAngle={2}
                                dataKey="value"
                            >
                                {piiTypeData.map((entry, index) => (
                                    <Cell key={`cell-${index}`} fill={entry.color} />
                                ))}
                            </Pie>
                            <Tooltip
                                contentStyle={{
                                    backgroundColor: theme.colors.background.primary,
                                    border: `1px solid ${theme.colors.border.default}`,
                                    borderRadius: '8px',
                                    color: theme.colors.text.primary,
                                    boxShadow: theme.shadows.md
                                }}
                                itemStyle={{ color: theme.colors.text.primary }}
                            />
                        </PieChart>
                    </ResponsiveContainer>
                </div>

                <div className="grid grid-cols-2 gap-2 mt-4">
                    {piiTypeData.slice(0, 6).map((item, index) => (
                        <div key={item.name} className="flex items-center gap-2 text-sm">
                            <div
                                className="w-3 h-3 rounded-full"
                                style={{ backgroundColor: item.color }}
                            />
                            <span className="text-slate-700 truncate font-medium">{item.name}</span>
                            <span className="text-slate-500 ml-auto">{item.value}</span>
                        </div>
                    ))}
                </div>
            </motion.div>

            {/* Asset Risk Distribution */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.1 }}
                className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm"
            >
                <div className="flex items-center gap-3 mb-6">
                    <div className="p-2 bg-blue-100/50 rounded-lg border border-blue-100">
                        <TrendingUp className="w-5 h-5 text-blue-600" />
                    </div>
                    <div>
                        <h3 className="text-lg font-bold text-slate-900">Asset Risk Overview</h3>
                        <p className="text-slate-600 text-sm">Findings distribution by asset</p>
                    </div>
                </div>

                <div className="h-64">
                    <ResponsiveContainer width="100%" height={256} minWidth={0}>
                        <BarChart data={assetData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
                            <XAxis
                                dataKey="name"
                                axisLine={false}
                                tickLine={false}
                                tick={{ fill: '#64748b', fontSize: 12 }}
                                angle={-45}
                                textAnchor="end"
                                height={60}
                            />
                            <YAxis
                                axisLine={false}
                                tickLine={false}
                                tick={{ fill: '#64748b', fontSize: 12 }}
                            />
                            <Tooltip
                                contentStyle={{
                                    backgroundColor: theme.colors.background.primary,
                                    border: `1px solid ${theme.colors.border.default}`,
                                    borderRadius: '8px',
                                    color: theme.colors.text.primary,
                                    boxShadow: theme.shadows.md
                                }}
                                cursor={{ fill: theme.colors.background.tertiary }}
                            />
                            <Bar
                                dataKey="findings"
                                fill="#3b82f6"
                                radius={[4, 4, 0, 0]}
                            />
                        </BarChart>
                    </ResponsiveContainer>
                </div>
            </motion.div>

            {/* Confidence Levels */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.2 }}
                className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm"
            >
                <div className="flex items-center gap-3 mb-6">
                    <div className="p-2 bg-emerald-100/50 rounded-lg border border-emerald-100">
                        <AlertTriangle className="w-5 h-5 text-emerald-600" />
                    </div>
                    <div>
                        <h3 className="text-lg font-bold text-slate-900">Confidence Distribution</h3>
                        <p className="text-slate-600 text-sm">Detection confidence levels</p>
                    </div>
                </div>

                <div className="space-y-3">
                    {confidenceData.map((item) => (
                        <div key={item.level} className="flex items-center justify-between">
                            <div className="flex items-center gap-3">
                                <div
                                    className="w-4 h-4 rounded"
                                    style={{ backgroundColor: item.color }}
                                />
                                <span className="text-slate-700 font-medium">{item.level}</span>
                            </div>
                            <div className="flex items-center gap-3">
                                <div className="w-24 bg-slate-100 rounded-full h-2">
                                    <div
                                        className="h-2 rounded-full transition-all duration-500"
                                        style={{
                                            width: `${(item.count / Math.max(...confidenceData.map(d => d.count), 1)) * 100}%`,
                                            backgroundColor: item.color
                                        }}
                                    />
                                </div>
                                <span className="text-slate-600 text-sm w-8 text-right font-medium">{item.count}</span>
                            </div>
                        </div>
                    ))}
                </div>
            </motion.div>
        </div>
    );
}