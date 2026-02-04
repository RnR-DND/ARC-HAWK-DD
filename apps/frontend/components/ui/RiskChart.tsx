import React from 'react';
import { motion } from 'framer-motion';
import { PieChart, Pie, Cell, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, Tooltip, Legend } from 'recharts';
import { TrendingUp, AlertTriangle, Shield, Database } from 'lucide-react';

interface RiskChartProps {
    byPiiType: Record<string, number>;
    byAsset: Record<string, number>;
    byConfidence: Record<string, number>;
    loading?: boolean;
}

const riskColors = {
    Critical: '#ef4444',
    High: '#f97316',
    Medium: '#eab308',
    Low: '#22c55e',
    Info: '#3b82f6',
};

const piiTypeColors = [
    '#8b5cf6', '#06b6d4', '#10b981', '#f59e0b', '#ef4444',
    '#3b82f6', '#ec4899', '#84cc16', '#f97316', '#6366f1'
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
            <div className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6">
                <div className="animate-pulse space-y-4">
                    <div className="h-6 w-32 bg-accent rounded" />
                    <div className="h-64 bg-accent rounded" />
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
                <div className="bg-gradient-to-br from-red-500/10 to-red-600/10 border border-red-500/30 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-red-400 text-sm font-medium">High Risk</p>
                            <p className="text-white text-2xl font-bold">{highRiskCount}</p>
                        </div>
                        <AlertTriangle className="w-8 h-8 text-red-400" />
                    </div>
                </div>

                <div className="bg-gradient-to-br from-blue-500/10 to-blue-600/10 border border-blue-500/30 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-blue-400 text-sm font-medium">Total Findings</p>
                            <p className="text-white text-2xl font-bold">{totalFindings}</p>
                        </div>
                        <Shield className="w-8 h-8 text-blue-400" />
                    </div>
                </div>

                <div className="bg-gradient-to-br from-emerald-500/10 to-emerald-600/10 border border-emerald-500/30 rounded-lg p-4">
                    <div className="flex items-center justify-between">
                        <div>
                            <p className="text-emerald-400 text-sm font-medium">Data Sources</p>
                            <p className="text-white text-2xl font-bold">{Object.keys(byAsset).length}</p>
                        </div>
                        <Database className="w-8 h-8 text-emerald-400" />
                    </div>
                </div>
            </motion.div>

            {/* PII Type Distribution */}
            <motion.div
                initial={{ opacity: 0, y: 20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: 0.1 }}
                className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6"
            >
                <div className="flex items-center justify-between mb-6">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-purple-500/20 rounded-lg">
                            <Shield className="w-5 h-5 text-purple-400" />
                        </div>
                        <div>
                            <h3 className="text-lg font-semibold text-white">PII Type Distribution</h3>
                            <p className="text-muted-foreground text-sm">Breakdown by sensitive data types</p>
                        </div>
                    </div>
                    <div className="text-right">
                        <p className="text-muted-foreground text-sm">Total Types</p>
                        <p className="text-white text-lg font-semibold">{piiTypeData.length}</p>
                    </div>
                </div>

                <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
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
                                    backgroundColor: '#1e293b',
                                    border: '1px solid #475569',
                                    borderRadius: '8px',
                                    color: '#f8fafc'
                                }}
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
                            <span className="text-slate-300 truncate">{item.name}</span>
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
                className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6"
            >
                <div className="flex items-center gap-3 mb-6">
                    <div className="p-2 bg-blue-500/20 rounded-lg">
                        <TrendingUp className="w-5 h-5 text-blue-400" />
                    </div>
                    <div>
                        <h3 className="text-lg font-semibold text-white">Asset Risk Overview</h3>
                        <p className="text-muted-foreground text-sm">Findings distribution by asset</p>
                    </div>
                </div>

                <div className="h-64">
                    <ResponsiveContainer width="100%" height="100%">
                        <BarChart data={assetData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
                            <XAxis
                                dataKey="name"
                                axisLine={false}
                                tickLine={false}
                                tick={{ fill: '#94a3b8', fontSize: 12 }}
                                angle={-45}
                                textAnchor="end"
                                height={60}
                            />
                            <YAxis
                                axisLine={false}
                                tickLine={false}
                                tick={{ fill: '#94a3b8', fontSize: 12 }}
                            />
                            <Tooltip
                                contentStyle={{
                                    backgroundColor: '#1e293b',
                                    border: '1px solid #475569',
                                    borderRadius: '8px',
                                    color: '#f8fafc'
                                }}
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
                className="bg-secondary backdrop-blur-sm border border-border/30 rounded-xl p-6"
            >
                <div className="flex items-center gap-3 mb-6">
                    <div className="p-2 bg-emerald-500/20 rounded-lg">
                        <AlertTriangle className="w-5 h-5 text-emerald-400" />
                    </div>
                    <div>
                        <h3 className="text-lg font-semibold text-white">Confidence Distribution</h3>
                        <p className="text-muted-foreground text-sm">Detection confidence levels</p>
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
                                <span className="text-slate-300 font-medium">{item.level}</span>
                            </div>
                            <div className="flex items-center gap-3">
                                <div className="w-24 bg-accent rounded-full h-2">
                                    <div
                                        className="h-2 rounded-full transition-all duration-500"
                                        style={{
                                            width: `${(item.count / Math.max(...confidenceData.map(d => d.count))) * 100}%`,
                                            backgroundColor: item.color
                                        }}
                                    />
                                </div>
                                <span className="text-muted-foreground text-sm w-8 text-right">{item.count}</span>
                            </div>
                        </div>
                    ))}
                </div>
            </motion.div>
        </div>
    );
}