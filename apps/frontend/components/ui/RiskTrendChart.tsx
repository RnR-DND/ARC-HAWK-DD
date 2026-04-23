'use client';

import React, { useEffect, useState } from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, ReferenceLine } from 'recharts';
import { TrendingUp, TrendingDown, Minus } from 'lucide-react';
import { getRiskTrend, RiskTrendPoint } from '@/services/dashboard.api';

function fmt(dateStr: string) {
    try {
        const d = new Date(dateStr);
        return d.toLocaleDateString('en-IN', { month: 'short', day: 'numeric' });
    } catch {
        return dateStr;
    }
}

const CustomTooltip = ({ active, payload, label }: any) => {
    if (!active || !payload?.length) return null;
    const { score, scan_count } = payload[0].payload;
    return (
        <div className="bg-white border border-slate-200 rounded-lg px-3 py-2 shadow-lg text-sm">
            <div className="font-semibold text-slate-900 mb-1">{label}</div>
            <div className="flex items-center gap-2 text-slate-600">
                <span className="w-2 h-2 rounded-full bg-blue-500 inline-block" />
                Risk Score: <span className="font-bold text-slate-900">{score}</span>
            </div>
            <div className="text-slate-500 text-xs mt-0.5">{scan_count} scan{scan_count !== 1 ? 's' : ''}</div>
        </div>
    );
};

export default function RiskTrendChart({ days = 30 }: { days?: number }) {
    const [data, setData] = useState<RiskTrendPoint[]>([]);
    const [loading, setLoading] = useState(true);

    useEffect(() => {
        getRiskTrend(days).then(pts => { setData(pts); setLoading(false); });
    }, [days]);

    const chartData = data.map(p => ({ ...p, label: fmt(p.date) }));

    const trend = (() => {
        if (chartData.length < 2) return 'flat';
        const first = chartData[0].score;
        const last = chartData[chartData.length - 1].score;
        if (last > first + 5) return 'up';
        if (last < first - 5) return 'down';
        return 'flat';
    })();

    const trendColor = trend === 'up' ? 'text-red-500' : trend === 'down' ? 'text-green-500' : 'text-slate-400';
    const TrendIcon = trend === 'up' ? TrendingUp : trend === 'down' ? TrendingDown : Minus;

    if (loading) {
        return (
            <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
                <div className="animate-pulse space-y-4">
                    <div className="h-5 w-40 bg-slate-100 rounded" />
                    <div className="h-48 bg-slate-50 rounded" />
                </div>
            </div>
        );
    }

    return (
        <div className="bg-white border border-slate-200 rounded-xl p-6 shadow-sm">
            <div className="flex items-center justify-between mb-4">
                <div>
                    <h3 className="text-base font-bold text-slate-900">Risk Score Trend</h3>
                    <p className="text-xs text-slate-500 mt-0.5">Last {days} days · severity-weighted 0–100</p>
                </div>
                <div className={`flex items-center gap-1 text-sm font-semibold ${trendColor}`}>
                    <TrendIcon className="w-4 h-4" />
                    {trend === 'up' ? 'Worsening' : trend === 'down' ? 'Improving' : 'Stable'}
                </div>
            </div>

            {chartData.length === 0 ? (
                <div className="flex items-center justify-center h-48 text-slate-400 text-sm">
                    No scan data for this period
                </div>
            ) : (
                <ResponsiveContainer width="100%" height={180}>
                    <LineChart data={chartData} margin={{ top: 4, right: 8, left: -20, bottom: 0 }}>
                        <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                        <XAxis
                            dataKey="label"
                            tick={{ fontSize: 11, fill: '#94a3b8' }}
                            tickLine={false}
                            axisLine={false}
                            interval="preserveStartEnd"
                        />
                        <YAxis
                            domain={[0, 100]}
                            tick={{ fontSize: 11, fill: '#94a3b8' }}
                            tickLine={false}
                            axisLine={false}
                        />
                        <Tooltip content={<CustomTooltip />} />
                        <ReferenceLine y={70} stroke="#fca5a5" strokeDasharray="4 4" label={{ value: 'High', fill: '#ef4444', fontSize: 10 }} />
                        <Line
                            type="monotone"
                            dataKey="score"
                            stroke="#3b82f6"
                            strokeWidth={2}
                            dot={{ r: 3, fill: '#3b82f6', strokeWidth: 0 }}
                            activeDot={{ r: 5 }}
                        />
                    </LineChart>
                </ResponsiveContainer>
            )}
        </div>
    );
}
