'use client';

import React from 'react';
import { Asset } from '@/types';
import { AlertTriangle, Database, FileCode, Server, Trash2 } from 'lucide-react';

interface AssetTableProps {
    assets: Asset[];
    total: number;
    loading?: boolean;
    onAssetClick: (id: string) => void;
    onDeleteAsset?: (id: string) => void;
}

export default function AssetTable({ assets, loading, onAssetClick, onDeleteAsset }: AssetTableProps) {
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
            <div className="p-12 text-center border-2 border-dashed border-slate-200 rounded-xl">
                <div className="text-4xl mb-4 opacity-50">📦</div>
                <h3 className="text-lg font-semibold text-slate-900 mb-2">No Assets Found</h3>
                <p className="text-slate-500">Run a scan or adjust filters to see assets.</p>
            </div>
        );
    }

    return (
        <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
                <thead>
                    <tr className="bg-slate-50 text-slate-600 border-b border-slate-200">
                        <th className="px-6 py-4 font-medium">Asset Name</th>
                        <th className="px-6 py-4 font-medium">Type</th>
                        <th className="px-6 py-4 font-medium">Risk Score</th>
                        <th className="px-6 py-4 font-medium">System</th>
                        <th className="px-6 py-4 font-medium">Findings</th>
                        <th className="px-6 py-4 font-medium text-right">Actions</th>
                    </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                    {assets.map((asset) => (
                        <AssetRow key={asset.id} asset={asset} onClick={() => onAssetClick(asset.id)} onDelete={onDeleteAsset ? () => onDeleteAsset(asset.id) : undefined} />
                    ))}
                </tbody>
            </table>
        </div>
    );
}

function AssetRow({ asset, onClick, onDelete }: { asset: Asset; onClick: () => void; onDelete?: () => void }) {
    return (
        <tr
            onClick={onClick}
            className="group hover:bg-slate-50 cursor-pointer transition-colors"
        >
            <td className="px-6 py-4">
                <div className="font-semibold text-slate-900 group-hover:text-blue-600 transition-colors">
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
            <td className="px-6 py-4 text-slate-600">
                <div className="flex items-center gap-2">
                    <Server className="w-4 h-4 text-slate-400" />
                    {asset.source_system}
                </div>
            </td>
            <td className="px-6 py-4">
                {asset.total_findings > 0 ? (
                    <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-red-50 text-red-600 border border-red-200 text-xs font-semibold">
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
            <td className="px-6 py-4 text-right">
                {onDelete && (
                    <button
                        onClick={(e) => { e.stopPropagation(); onDelete(); }}
                        className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                        title="Delete asset"
                    >
                        <Trash2 className="w-4 h-4" />
                    </button>
                )}
            </td>
        </tr>
    );
}

function TypeBadge({ type }: { type: string }) {
    let icon = <FileCode className="w-3 h-3" />;
    if (type === 'Table') icon = <Database className="w-3 h-3" />;

    return (
        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium bg-slate-100 text-slate-600 border border-slate-200">
            {icon}
            {type}
        </span>
    );
}

function RiskBadge({ score }: { score: number }) {
    let colorClass = "bg-slate-50 text-slate-600 border-slate-200";

    if (score >= 90) colorClass = "bg-red-50 text-red-700 border-red-200";
    else if (score >= 70) colorClass = "bg-orange-50 text-orange-700 border-orange-200";
    else if (score >= 40) colorClass = "bg-yellow-50 text-yellow-700 border-yellow-200";
    else colorClass = "bg-blue-50 text-blue-700 border-blue-200";

    return (
        <span className={`inline-flex items-center justify-center px-2 py-0.5 rounded border text-xs font-bold ${colorClass}`}>
            {score}
        </span>
    );
}
