"use client";

import { cn, formatNumber } from "@/lib/utils";
import { TrendingUp, TrendingDown, Minus } from "lucide-react";
import {
  ResponsiveContainer,
  AreaChart,
  Area,
} from "recharts";

interface MetricCardProps {
  label: string;
  value: number;
  format?: "number" | "percent";
  trend?: number;
  sparkData?: { value: number }[];
  sparkColor?: string;
  icon?: React.ReactNode;
  className?: string;
}

export function MetricCard({
  label,
  value,
  format = "number",
  trend,
  sparkData,
  sparkColor = "#3B82F6",
  icon,
  className,
}: MetricCardProps) {
  const displayValue =
    format === "percent" ? `${value}%` : formatNumber(value);

  return (
    <div
      className={cn(
        "rounded-xl border border-gray-200 bg-white p-5 shadow-sm",
        className
      )}
    >
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500">{label}</p>
          <p className="mt-1 text-2xl font-bold text-gray-900">
            {displayValue}
          </p>
          {trend !== undefined && (
            <div
              className={cn(
                "mt-1 flex items-center gap-1 text-xs font-medium",
                trend > 0
                  ? "text-red-600"
                  : trend < 0
                    ? "text-green-600"
                    : "text-gray-500"
              )}
            >
              {trend > 0 ? (
                <TrendingUp className="h-3 w-3" aria-hidden="true" />
              ) : trend < 0 ? (
                <TrendingDown className="h-3 w-3" aria-hidden="true" />
              ) : (
                <Minus className="h-3 w-3" aria-hidden="true" />
              )}
              <span>
                {trend > 0 ? "+" : ""}
                {trend}% from last period
              </span>
            </div>
          )}
        </div>
        {icon && (
          <div className="rounded-lg bg-blue-50 p-2 text-blue-600">
            {icon}
          </div>
        )}
      </div>
      {sparkData && sparkData.length > 0 && (
        <div className="mt-3 h-10">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={sparkData}>
              <defs>
                <linearGradient id={`spark-${label}`} x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={sparkColor} stopOpacity={0.2} />
                  <stop offset="95%" stopColor={sparkColor} stopOpacity={0} />
                </linearGradient>
              </defs>
              <Area
                type="monotone"
                dataKey="value"
                stroke={sparkColor}
                strokeWidth={1.5}
                fill={`url(#spark-${label})`}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
