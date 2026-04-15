import { clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: any[]) {
  return twMerge(clsx(inputs));
}

export function formatNumber(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return String(value);
}

export const RISK_BG_CLASSES: Record<string, string> = {
  Critical: 'bg-red-100 text-red-800 border-red-200',
  High:     'bg-orange-100 text-orange-800 border-orange-200',
  Medium:   'bg-yellow-100 text-yellow-800 border-yellow-200',
  Low:      'bg-green-100 text-green-800 border-green-200',
  None:     'bg-slate-100 text-slate-600 border-slate-200',
};

export function tierLabel(tier: string): string {
  return tier || 'Unknown';
}
