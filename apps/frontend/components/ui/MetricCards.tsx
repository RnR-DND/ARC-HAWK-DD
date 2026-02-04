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
        color: 'from-blue-500 to-blue-600',
        bgColor: 'from-blue-500/10 to-blue-600/10',
        borderColor: 'border-blue-500/30',
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
        color: 'from-red-500 to-red-600',
        bgColor: 'from-red-500/10 to-red-600/10',
        borderColor: 'border-red-500/30',
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
        color: 'from-amber-500 to-amber-600',
        bgColor: 'from-amber-500/10 to-amber-600/10',
        borderColor: 'border-amber-500/30',
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
        color: 'from-emerald-500 to-emerald-600',
        bgColor: 'from-emerald-500/10 to-emerald-600/10',
        borderColor: 'border-emerald-500/30',
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
                        className={`relative overflow-hidden bg-gradient-to-br ${metric.bgColor} backdrop-blur-sm border ${metric.borderColor} rounded-xl p-6 hover:scale-105 transition-all duration-300 group cursor-pointer`}
                        title={metric.description}
                    >
                        {/* Priority indicator */}
                        {metric.priority === 'critical' && hasValue && (
                            <div className="absolute top-3 right-3 w-2 h-2 bg-red-500 rounded-full animate-pulse" />
                        )}

                        {/* Background Pattern */}
                        <div className="absolute inset-0 opacity-5">
                            <div className="absolute top-0 right-0 w-32 h-32 bg-white rounded-full -mr-16 -mt-16" />
                        </div>

                        <div className="relative">
                            {/* Icon with priority styling */}
                            <div className={`inline-flex p-3 rounded-lg mb-4 transition-all duration-300 ${hasValue
                                    ? `bg-gradient-to-br ${metric.color} shadow-lg`
                                    : 'bg-slate-600/50'
                                }`}>
                                <Icon className={`w-6 h-6 transition-colors duration-300 ${hasValue ? 'text-white' : 'text-muted-foreground'
                                    }`} />
                            </div>

                            {/* Value with better formatting */}
                            <div className="flex items-center justify-between mb-3">
                                <div className={`text-3xl font-bold transition-all duration-300 ${hasValue
                                        ? `bg-gradient-to-r ${metric.color} bg-clip-text text-transparent`
                                        : 'text-slate-500'
                                    }`}>
                                    {loading ? (
                                        <div className="h-8 w-16 bg-slate-600 rounded animate-pulse" />
                                    ) : (
                                        <span className="font-mono">
                                            {value.toLocaleString()}
                                        </span>
                                    )}
                                </div>

                                {/* Trend Indicator with better context */}
                                {!loading && metric.trend !== 'neutral' && hasValue && (
                                    <div className={`flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium transition-all duration-300 ${metric.trend === 'up' && metric.priority === 'critical'
                                            ? 'bg-red-500/20 text-red-400 border border-red-500/30'
                                            : metric.trend === 'up'
                                                ? 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
                                                : 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
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
                            <h3 className={`font-semibold text-lg mb-1 transition-colors duration-300 ${hasValue ? 'text-white' : 'text-muted-foreground'
                                }`}>
                                {metric.label}
                            </h3>

                            {/* Subtitle with action hint */}
                            <p className="text-muted-foreground text-sm mb-3">
                                {metric.subtitle}
                            </p>

                            {/* Action button for better UX */}
                            <button className={`w-full py-2 px-3 rounded-lg text-xs font-medium transition-all duration-300 ${hasValue
                                    ? `bg-white/10 hover:bg-white/20 text-white border border-white/20`
                                    : 'bg-slate-600/30 text-slate-500 cursor-not-allowed'
                                } ${hasValue ? 'hover:scale-105' : ''}`}>
                                {metric.actionText}
                            </button>
                        </div>

                        {/* Enhanced Hover Effect */}
                        <div className="absolute inset-0 bg-gradient-to-br from-white/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 rounded-xl" />

                        {/* Tooltip on hover */}
                        <div className="absolute bottom-full left-1/2 transform -translate-x-1/2 mb-2 px-3 py-2 bg-card text-white text-xs rounded-lg opacity-0 group-hover:opacity-100 transition-opacity duration-300 pointer-events-none whitespace-nowrap z-10 border border-border">
                            {metric.description}
                            <div className="absolute top-full left-1/2 transform -translate-x-1/2 border-4 border-transparent border-t-slate-900"></div>
                        </div>
                    </motion.div>
                );
            })}
        </div>
    );
}