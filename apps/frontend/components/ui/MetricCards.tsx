import React from 'react';
import { motion } from 'framer-motion';
import { Shield, AlertTriangle, Database, CheckCircle, TrendingUp, TrendingDown } from 'lucide-react';

interface MetricCardsProps {
    totalPII: number;
    highRiskFindings: number;
    assetsHit: number;
    actionsRequired: number;
    loading?: boolean;
}

const metrics = [
    {
        label: 'PII Instances Found',
        value: 'totalPII',
        subtitle: 'Total sensitive data detected',
        description: 'Number of PII occurrences across all scanned data sources',
        icon: Shield,
        color: 'from-blue-600 to-blue-700',
        bgColor: 'from-blue-50 to-blue-100/50',
        borderColor: 'border-blue-200',
        iconBg: 'bg-blue-100 text-blue-600',
        trend: 'up' as const,
        priority: 'info' as const,
        actionText: 'View Details',
    },
    {
        label: 'Critical Findings',
        value: 'highRiskFindings',
        subtitle: 'High-risk PII requiring action',
        description: 'Findings classified as high or critical risk that need immediate attention',
        icon: AlertTriangle,
        color: 'from-red-600 to-red-700',
        bgColor: 'from-red-50 to-red-100/50',
        borderColor: 'border-red-200',
        iconBg: 'bg-red-100 text-red-600',
        trend: 'up' as const,
        priority: 'critical' as const,
        actionText: 'Review Now',
    },
    {
        label: 'Data Sources Impacted',
        value: 'assetsHit',
        subtitle: 'Systems containing PII',
        description: 'Number of databases, files, and cloud storage locations with sensitive data',
        icon: Database,
        color: 'from-amber-600 to-amber-700',
        bgColor: 'from-amber-50 to-amber-100/50',
        borderColor: 'border-amber-200',
        iconBg: 'bg-amber-100 text-amber-600',
        trend: 'neutral' as const,
        priority: 'medium' as const,
        actionText: 'Manage Sources',
    },
    {
        label: 'Remediation Tasks',
        value: 'actionsRequired',
        subtitle: 'Pending resolution items',
        description: 'PII findings awaiting review, masking, or other remediation actions',
        icon: CheckCircle,
        color: 'from-emerald-600 to-emerald-700',
        bgColor: 'from-emerald-50 to-emerald-100/50',
        borderColor: 'border-emerald-200',
        iconBg: 'bg-emerald-100 text-emerald-600',
        trend: 'down' as const,
        priority: 'low' as const,
        actionText: 'Coming Soon',
    },
];

export default function MetricCards({
    totalPII,
    highRiskFindings,
    assetsHit,
    actionsRequired,
    loading = false
}: MetricCardsProps) {
    const values = { totalPII, highRiskFindings, assetsHit, actionsRequired };

    return (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
            {metrics.map((metric, index) => {
                const Icon = metric.icon;
                const value = values[metric.value as keyof typeof values];
                const hasValue = value > 0;

                return (
                    <motion.div
                        key={metric.label}
                        initial={{ opacity: 0, y: 20 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: index * 0.1 }}
                        className={`relative overflow-hidden bg-white border ${metric.borderColor} rounded-xl p-6 hover:shadow-lg transition-all duration-300 group cursor-pointer shadow-sm`}
                        title={metric.description}
                    >
                        {/* Priority indicator */}
                        {metric.priority === 'critical' && hasValue && (
                            <div className="absolute top-3 right-3 w-2 h-2 bg-red-500 rounded-full animate-pulse" />
                        )}

                        {/* Background Gradient */}
                        <div className={`absolute inset-0 bg-gradient-to-br ${metric.bgColor} opacity-50`} />

                        <div className="relative z-10">
                            {/* Icon with priority styling */}
                            <div className={`inline-flex p-3 rounded-lg mb-4 transition-all duration-300 ${hasValue
                                ? `bg-white shadow-sm border border-slate-100`
                                : 'bg-slate-100'
                                }`}>
                                <Icon className={`w-6 h-6 transition-colors duration-300 ${hasValue ? metric.iconBg.split(' ')[1] : 'text-slate-400'
                                    }`} />
                            </div>

                            {/* Value with better formatting */}
                            <div className="flex items-center justify-between mb-3">
                                <div className={`text-3xl font-bold transition-all duration-300 ${hasValue
                                    ? `bg-gradient-to-r ${metric.color} bg-clip-text text-transparent`
                                    : 'text-slate-400'
                                    }`}>
                                    {loading ? (
                                        <div className="h-8 w-16 bg-slate-200 rounded animate-pulse" />
                                    ) : (
                                        <span className="font-mono">
                                            {value.toLocaleString()}
                                        </span>
                                    )}
                                </div>

                                {/* Trend Indicator with better context */}
                                {!loading && metric.trend !== 'neutral' && hasValue && (
                                    <div className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium transition-all duration-300 bg-white/80 backdrop-blur-sm border ${metric.borderColor} ${metric.trend === 'up' && metric.priority === 'critical'
                                        ? 'text-red-600'
                                        : metric.trend === 'up'
                                            ? 'text-amber-600'
                                            : 'text-emerald-600'
                                        }`}>
                                        {metric.trend === 'up' ? (
                                            <TrendingUp className="w-3 h-3" />
                                        ) : (
                                            <TrendingDown className="w-3 h-3" />
                                        )}
                                        <span className="hidden sm:inline">
                                            {metric.trend === 'up' ? 'Increasing' : 'Decreasing'}
                                        </span>
                                    </div>
                                )}
                            </div>

                            {/* Label with better hierarchy */}
                            <h3 className={`font-semibold text-lg mb-1 transition-colors duration-300 ${hasValue ? 'text-slate-800' : 'text-slate-400'
                                }`}>
                                {metric.label}
                            </h3>

                            {/* Subtitle with action hint */}
                            <p className="text-slate-500 text-sm mb-4 line-clamp-1">
                                {metric.subtitle}
                            </p>

                            {/* Action button for better UX */}
                            <button className={`w-full py-2 px-3 rounded-lg text-xs font-semibold transition-all duration-300 ${hasValue
                                ? `bg-white hover:bg-slate-50 text-slate-700 border border-slate-200 shadow-sm hover:shadow`
                                : 'bg-slate-100 text-slate-400 cursor-not-allowed border border-slate-200'
                                }`}>
                                {metric.actionText}
                            </button>
                        </div>
                    </motion.div>
                );
            })}
        </div>
    );
}