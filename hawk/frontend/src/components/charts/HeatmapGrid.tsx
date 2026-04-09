"use client";

import { cn } from "@/lib/utils";
import type { DiscoverySource, RiskTier } from "@/types/api";

const HEAT_COLORS: Record<string, string> = {
  "0": "bg-green-50 text-green-800",
  "1": "bg-green-100 text-green-800",
  "2": "bg-yellow-50 text-yellow-800",
  "3": "bg-yellow-100 text-yellow-800",
  "4": "bg-orange-100 text-orange-800",
  "5": "bg-orange-200 text-orange-900",
  "6": "bg-red-100 text-red-800",
  "7": "bg-red-200 text-red-900",
  "8": "bg-red-300 text-red-900",
  "9": "bg-red-400 text-white",
  "10": "bg-red-600 text-white",
};

function getHeatLevel(density: number): string {
  const level = Math.min(10, Math.floor(density * 100));
  return String(level);
}

interface HeatmapGridProps {
  sources: DiscoverySource[];
  className?: string;
}

export function HeatmapGrid({ sources, className }: HeatmapGridProps) {
  const sortedSources = [...sources].sort(
    (a, b) => b.pii_density - a.pii_density
  );

  return (
    <div className={cn("space-y-2", className)}>
      <div className="flex items-center justify-between text-xs text-gray-500">
        <span>PII Density Heatmap</span>
        <div className="flex items-center gap-1">
          <span>Low</span>
          <div className="flex gap-0.5">
            {["0", "2", "4", "6", "8", "10"].map((level) => (
              <div
                key={level}
                className={cn("h-3 w-3 rounded-sm", HEAT_COLORS[level])}
              />
            ))}
          </div>
          <span>High</span>
        </div>
      </div>
      <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 md:grid-cols-4">
        {sortedSources.map((source) => {
          const level = getHeatLevel(source.pii_density);
          return (
            <div
              key={source.id}
              className={cn(
                "rounded-lg border p-3 transition-shadow hover:shadow-md",
                HEAT_COLORS[level]
              )}
              role="gridcell"
              aria-label={`${source.name}: ${(source.pii_density * 100).toFixed(1)}% PII density`}
            >
              <div className="text-xs font-medium truncate">
                {source.name}
              </div>
              <div className="mt-1 text-lg font-bold">
                {(source.pii_density * 100).toFixed(1)}%
              </div>
              <div className="mt-0.5 text-xs opacity-75">
                {source.asset_count.toLocaleString()} assets
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
