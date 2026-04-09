"use client";

import { cn, RISK_BG_CLASSES, tierLabel } from "@/lib/utils";
import type { RiskTier } from "@/types/api";
import { AlertOctagon, AlertTriangle, Info, CheckCircle } from "lucide-react";

const TIER_ICONS: Record<RiskTier, React.ElementType> = {
  critical: AlertOctagon,
  high: AlertTriangle,
  medium: Info,
  low: CheckCircle,
};

interface StatusBadgeProps {
  tier: RiskTier;
  size?: "sm" | "md";
  showIcon?: boolean;
  className?: string;
}

export function StatusBadge({
  tier,
  size = "sm",
  showIcon = true,
  className,
}: StatusBadgeProps) {
  const Icon = TIER_ICONS[tier];

  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full border font-medium",
        RISK_BG_CLASSES[tier],
        size === "sm" ? "px-2 py-0.5 text-xs" : "px-3 py-1 text-sm",
        className
      )}
      role="status"
      aria-label={`Risk tier: ${tierLabel(tier)}`}
    >
      {showIcon && <Icon className={size === "sm" ? "h-3 w-3" : "h-4 w-4"} aria-hidden="true" />}
      {tierLabel(tier)}
    </span>
  );
}
