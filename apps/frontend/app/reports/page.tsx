'use client';

import React, { useState, useEffect } from 'react';
import { FileText, Download, TrendingUp, Shield, AlertTriangle } from 'lucide-react';
import Topbar from '@/components/Topbar';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';

import { exportToCSV } from '@/utils/export';
import { findingsApi } from '@/services/findings.api';
import { assetsApi } from '@/services/assets.api';

interface ReportMetrics {
    totalFindings: number;
    criticalFindings: number;
    assetsScanned: number;
    complianceScore: number;
    generatedAt: string;
}

export default function ReportsPage() {
    const [metrics, setMetrics] = useState<ReportMetrics | null>(null);
    const [loading, setLoading] = useState(true);
    const [generating, setGenerating] = useState(false);

    useEffect(() => {
        fetchReportMetrics();
    }, []);

    const fetchReportMetrics = async () => {
        try {
            // Get metrics from multiple APIs
            const [findingsRes, assetsRes] = await Promise.all([
                findingsApi.getFindings({ page_size: 1000 }),
                assetsApi.getAssets({ page_size: 1000 })
            ]);

            const totalFindings = findingsRes.total || 0;
            const criticalFindings = findingsRes.findings?.filter(f => f.severity === 'Critical').length || 0;
            const assetsScanned = assetsRes.total || 0;

            // Calculate compliance score based on findings (simplified)
            const complianceScore = Math.max(0, 100 - (totalFindings * 2));

            setMetrics({
                totalFindings,
                criticalFindings,
                assetsScanned,
                complianceScore,
                generatedAt: new Date().toISOString()
            });
        } catch (error) {
            console.error('Failed to fetch metrics:', error);
        } finally {
            setLoading(false);
        }
    };

    const handleDownloadCSV = async (reportType: string) => {
        setGenerating(true);
        try {
            let data: any[] = [];
            let filename = '';

            switch (reportType) {
                case 'findings':
                    const findingsResult = await findingsApi.getFindings({ page_size: 1000 });
                    data = findingsResult.findings || [];
                    filename = 'findings_report';
                    break;
                case 'assets':
                    const assetsResult = await assetsApi.getAssets({ page_size: 1000 });
                    data = assetsResult.assets || [];
                    filename = 'assets_report';
                    break;
                case 'compliance':
                    // Generate compliance summary
                    data = [{
                        report_type: 'Compliance Summary',
                        generated_at: new Date().toISOString(),
                        compliance_score: metrics?.complianceScore || 0,
                        total_findings: metrics?.totalFindings || 0,
                        critical_findings: metrics?.criticalFindings || 0,
                        assets_scanned: metrics?.assetsScanned || 0
                    }];
                    filename = 'compliance_summary';
                    break;
            }

            exportToCSV(data, filename);
        } catch (e) {
            console.error(e);
            alert('Failed to generate report');
        } finally {
            setGenerating(false);
        }
    };

    return (
        <div className="min-h-screen bg-slate-50">
            <Topbar />
            <div className="container mx-auto px-4 py-8 max-w-6xl">
                {/* Header */}
                <div className="flex justify-between items-center mb-8">
                    <div>
                        <h1 className="text-3xl font-bold text-slate-900 mb-2 tracking-tight">
                            Compliance Reports
                        </h1>
                        <p className="text-slate-500 text-base">
                            Generate and export detailed compliance and risk assessment reports
                        </p>
                    </div>
                    <div className="text-sm text-slate-500">
                        Last updated: {metrics?.generatedAt ? new Date(metrics.generatedAt).toLocaleString() : 'Loading...'}
                    </div>
                </div>

                {/* Metrics Overview */}
                {metrics && (
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 mb-8">
                        <MetricCard
                            title="Compliance Score"
                            value={`${Math.round(metrics.complianceScore)}%`}
                            subtitle="Overall posture"
                            colorClass={metrics.complianceScore > 80 ? "text-emerald-600" : "text-amber-500"}
                            barColorClass={metrics.complianceScore > 80 ? "bg-emerald-500" : "bg-amber-500"}
                            icon="🛡️"
                        />
                        <MetricCard
                            title="Total Findings"
                            value={metrics.totalFindings.toLocaleString()}
                            subtitle="PII detections"
                            colorClass="text-blue-500"
                            barColorClass="bg-blue-500"
                            icon="🔍"
                        />
                        <MetricCard
                            title="Critical Issues"
                            value={metrics.criticalFindings.toLocaleString()}
                            subtitle="High-risk findings"
                            colorClass="text-red-500"
                            barColorClass="bg-red-500"
                            icon="⚠️"
                        />
                        <MetricCard
                            title="Assets Scanned"
                            value={metrics.assetsScanned.toLocaleString()}
                            subtitle="Data sources"
                            colorClass="text-blue-600"
                            barColorClass="bg-blue-600"
                            icon="📦"
                        />
                    </div>
                )}

                {/* Report Types */}
                <div className="mb-8">
                    <h2 className="text-xl font-bold text-slate-900 mb-6">
                        Generate Reports
                    </h2>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                        <ReportCard
                            title="Compliance Executive Summary"
                            description="High-level overview of risk trends, remediation actions, and compliance posture for executive stakeholders."
                            icon={<FileText className="w-5 h-5" />}
                            colorClass="text-blue-600 bg-blue-50"
                            buttonClass="bg-blue-600 hover:bg-blue-700"
                            onDownload={() => handleDownloadCSV('compliance')}
                            loading={generating}
                            features={['Executive metrics', 'Risk trends', 'Compliance score', 'PDF format']}
                        />
                        <ReportCard
                            title="Technical Findings Report"
                            description="Detailed breakdown of all PII detections with technical details for remediation teams."
                            icon={<AlertTriangle className="w-5 h-5" />}
                            colorClass="text-orange-600 bg-orange-50"
                            buttonClass="bg-orange-600 hover:bg-orange-700"
                            onDownload={() => handleDownloadCSV('findings')}
                            loading={generating}
                            features={['All findings', 'Technical details', 'Severity levels', 'CSV/Excel format']}
                        />
                        <ReportCard
                            title="Asset Inventory Report"
                            description="Complete catalog of scanned assets with risk scores and compliance status."
                            icon={<Shield className="w-5 h-5" />}
                            colorClass="text-blue-600 bg-blue-50"
                            buttonClass="bg-blue-600 hover:bg-blue-700"
                            onDownload={() => handleDownloadCSV('assets')}
                            loading={generating}
                            features={['Asset catalog', 'Risk assessment', 'Compliance status', 'Multiple formats']}
                        />
                        <ReportCard
                            title="Trend Analysis Report"
                            description="Historical analysis of PII exposure trends and remediation effectiveness over time."
                            icon={<TrendingUp className="w-5 h-5" />}
                            colorClass="text-indigo-600 bg-indigo-50"
                            buttonClass="bg-indigo-600 hover:bg-indigo-700"
                            onDownload={() => handleDownloadCSV('trend')}
                            loading={false}
                            features={['Historical data', 'Trend analysis', 'Effectiveness metrics', 'Visual charts']}
                        />
                    </div>
                </div>

                {/* Report Archive */}
                <div>
                    <h2 className="text-xl font-bold text-slate-900 mb-6">
                        Report Archive
                    </h2>
                    <Card>
                        <CardHeader className="border-b border-border pb-4">
                            <CardTitle className="text-base">Recent Reports</CardTitle>
                            <CardDescription>Previously generated reports and scheduled exports</CardDescription>
                        </CardHeader>
                        <CardContent className="p-0">
                            <div className="overflow-x-auto">
                                <table className="w-full">
                                    <thead className="bg-slate-50">
                                        <tr>
                                            <th className="px-6 py-4 text-left text-xs font-bold text-slate-500 uppercase tracking-wider border-b border-border">Report Name</th>
                                            <th className="px-6 py-4 text-left text-xs font-bold text-slate-500 uppercase tracking-wider border-b border-border">Generated</th>
                                            <th className="px-6 py-4 text-left text-xs font-bold text-slate-500 uppercase tracking-wider border-b border-border">Type</th>
                                            <th className="px-6 py-4 text-left text-xs font-bold text-slate-500 uppercase tracking-wider border-b border-border">Format</th>
                                            <th className="px-6 py-4 text-left text-xs font-bold text-slate-500 uppercase tracking-wider border-b border-border">Size</th>
                                            <th className="px-6 py-4 text-left text-xs font-bold text-slate-500 uppercase tracking-wider border-b border-border">Status</th>
                                            <th className="px-6 py-4 text-left text-xs font-bold text-slate-500 uppercase tracking-wider border-b border-border">Actions</th>
                                        </tr>
                                    </thead>
                                    <tbody className="divide-y divide-border">
                                        {[
                                            { name: 'Compliance_Summary_Jan_2026', date: '2026-01-15', type: 'Executive', format: 'PDF', size: '2.4 MB', status: 'Ready' },
                                            { name: 'Findings_Detailed_Report', date: '2026-01-14', type: 'Technical', format: 'CSV', size: '1.8 MB', status: 'Ready' },
                                            { name: 'Asset_Risk_Assessment', date: '2026-01-13', type: 'Inventory', format: 'Excel', size: '3.1 MB', status: 'Ready' },
                                            { name: 'Monthly_Trend_Analysis', date: '2026-01-10', type: 'Analytics', format: 'PDF', size: '4.2 MB', status: 'Processing' },
                                        ].map((report, i) => (
                                            <tr key={i} className="hover:bg-slate-50/50 transition-colors">
                                                <td className="px-6 py-4 whitespace-nowrap">
                                                    <div className="font-semibold text-slate-900">
                                                        {report.name.replace(/_/g, ' ')}
                                                    </div>
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500">
                                                    {new Date(report.date).toLocaleDateString()}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap">
                                                    <Badge variant="secondary" className="font-semibold">
                                                        {report.type}
                                                    </Badge>
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap">
                                                    <Badge className={`${getFormatBadgeColor(report.format)} text-white border-0`}>
                                                        {report.format}
                                                    </Badge>
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap text-sm text-slate-500 font-mono">
                                                    {report.size}
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap">
                                                    <Badge variant="outline" className={`${report.status === 'Ready' ? 'text-emerald-600 bg-emerald-50 border-emerald-200' : 'text-amber-600 bg-amber-50 border-amber-200'}`}>
                                                        {report.status}
                                                    </Badge>
                                                </td>
                                                <td className="px-6 py-4 whitespace-nowrap">
                                                    {report.status === 'Ready' ? (
                                                        <Button variant="ghost" size="sm" className="h-8 text-blue-600 hover:text-blue-700 hover:bg-blue-50">
                                                            Download
                                                        </Button>
                                                    ) : (
                                                        <span className="text-sm text-slate-400 italic">
                                                            Processing...
                                                        </span>
                                                    )}
                                                </td>
                                            </tr>
                                        ))}
                                    </tbody>
                                </table>
                            </div>
                        </CardContent>
                    </Card>
                </div>
            </div>
        </div>
    );
}

function MetricCard({ title, value, subtitle, colorClass, barColorClass, icon }: any) {
    return (
        <Card className="relative overflow-hidden">
            <div className={`absolute left-0 top-0 bottom-0 w-1 ${barColorClass}`} />
            <CardContent className="p-6">
                <div className="flex items-center gap-3 mb-3">
                    <span className="text-xl">{icon}</span>
                    <div className="text-sm font-semibold text-slate-500">
                        {title}
                    </div>
                </div>
                <div className={`text-3xl font-bold mb-1 ${colorClass}`}>
                    {value}
                </div>
                <div className="text-sm text-slate-500">
                    {subtitle}
                </div>
            </CardContent>
        </Card>
    );
}

function ReportCard({ title, description, icon, colorClass, buttonClass, onDownload, loading, features }: any) {
    return (
        <Card className="hover:shadow-lg transition-all duration-200 cursor-pointer hover:-translate-y-0.5">
            <CardContent className="p-6">
                <div className="flex items-center gap-3 mb-4">
                    <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${colorClass}`}>
                        {icon}
                    </div>
                    <h3 className="font-bold text-slate-900 leading-tight">
                        {title}
                    </h3>
                </div>

                <p className="text-sm text-slate-500 mb-5 leading-relaxed min-h-[60px]">
                    {description}
                </p>

                <div className="mb-5">
                    <div className="text-xs text-slate-400 font-medium mb-2 uppercase tracking-wide">
                        Includes
                    </div>
                    <div className="flex flex-wrap gap-2">
                        {features.map((feature: string, i: number) => (
                            <span key={i} className="px-2 py-1 rounded bg-slate-100 text-xs text-slate-600 font-medium">
                                {feature}
                            </span>
                        ))}
                    </div>
                </div>

                <Button
                    onClick={onDownload}
                    disabled={loading}
                    className={`w-full ${buttonClass} text-white`}
                >
                    {loading ? (
                        <>
                            <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                            Generating...
                        </>
                    ) : (
                        <>
                            <Download className="w-4 h-4 mr-2" />
                            Generate Report
                        </>
                    )}
                </Button>
            </CardContent>
        </Card>
    );
}

function getFormatBadgeColor(format: string) {
    switch (format.toLowerCase()) {
        case 'pdf': return 'bg-red-500 hover:bg-red-600';
        case 'csv': return 'bg-emerald-500 hover:bg-emerald-600';
        case 'excel': return 'bg-blue-500 hover:bg-blue-600';
        default: return 'bg-slate-500';
    }
}
