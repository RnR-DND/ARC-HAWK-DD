'use client';

import React from 'react';
import { Asset } from '@/types';
import { AlertTriangle, Database, FileCode, Server } from 'lucide-react';

interface AssetTableProps {
    assets: Asset[];
    total: number;
    loading?: boolean;
    onAssetClick: (id: string) => void;
}

export default function AssetTable({ assets, loading, onAssetClick }: AssetTableProps) {
    if (loading) {
        return (
            <div className="p-8 text-center text-muted-foreground flex flex-col items-center">
                <div className="animate-pulse w-12 h-12 bg-muted rounded-full mb-4"></div>
                Loading assets...
            </div>
        );
    }

    if (assets.length === 0) {
        return (
            <div className="p-12 text-center border-2 border-dashed border-border rounded-xl">
                <div className="text-4xl mb-4 opacity-50">📦</div>
                <h3 className="text-lg font-semibold text-white mb-2">No Assets Found</h3>
                <p className="text-muted-foreground">Run a scan or adjust filters to see assets.</p>
            </div>
        );
    }

    return (
        <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
                <thead>
                    <tr className="bg-secondary text-muted-foreground border-b border-border">
                        <th className="px-6 py-4 font-medium">Asset Name</th>
                        <th className="px-6 py-4 font-medium">Type</th>
                        <th className="px-6 py-4 font-medium">Risk Score</th>
                        <th className="px-6 py-4 font-medium">System</th>
                        <th className="px-6 py-4 font-medium">Findings</th>
                    </tr>
                </thead>
                <tbody className="divide-y divide-border">
                    {assets.map((asset) => (
                        <AssetRow key={asset.id} asset={asset} onClick={() => onAssetClick(asset.id)} />
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function AssetRow({ asset, onClick }: { asset: Asset; onClick: () => void }) {
    return (
        <tr
            onClick={onClick}
            className="group hover:bg-muted cursor-pointer transition-colors"
        >
            <td className="px-6 py-4">
                <div className="font-semibold text-slate-200 group-hover:text-blue-400 transition-colors">
                    {asset.name}
                </div>
                <div
                    className="text-xs text-slate-500 mt-1 font-mono truncate max-w-[300px]"
                    title={asset.path}
                >
                    {asset.path}
                </div>
            </td>
            <td className="px-6 py-4">
                <TypeBadge type={asset.asset_type} />
            </td>
            <td className="px-6 py-4">
                <RiskBadge score={asset.risk_score} />
            </td>
            <td className="px-6 py-4 text-muted-foreground">
                <div className="flex items-center gap-2">
                    <Server className="w-4 h-4 text-slate-600" />
                    {asset.source_system}
                </div>
            </td>
            <td className="px-6 py-4">
                {asset.total_findings > 0 ? (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-red-500/10 text-red-400 border border-red-500/20 text-xs font-semibold">
                        <AlertTriangle className="w-3 h-3" />
                        {asset.total_findings}
                    </span>
                ) : (
                    <span className="text-slate-500 text-xs flex items-center gap-1.5">
                        <div className="w-1.5 h-1.5 rounded-full bg-green-500/50" />
                        Safe
                    </span>
                )}
            </td>
        </tr>
    );
}

function TypeBadge({ type }: { type: string }) {
    let icon = <FileCode className="w-3 h-3" />;
    if (type === 'Table') icon = <Database className="w-3 h-3" />;

    return (
        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-muted text-slate-300 border border-border">
            {icon}
            {type}
        </span>
    );
}

function RiskBadge({ score }: { score: number }) {
    let colorClass = "bg-muted text-muted-foreground border-border";

    if (score >= 90) colorClass = "bg-red-500/10 text-red-500 border-red-500/20";
    else if (score >= 70) colorClass = "bg-orange-500/10 text-orange-500 border-orange-500/20";
    else if (score >= 40) colorClass = "bg-yellow-500/10 text-yellow-500 border-yellow-500/20";
    else colorClass = "bg-blue-500/10 text-blue-500 border-blue-500/20";

    return (
        <span className={`inline-flex items-center justify-center px-2 py-0.5 rounded border text-xs font-bold ${colorClass}`}>
            {score}
        </span>
    );
}
