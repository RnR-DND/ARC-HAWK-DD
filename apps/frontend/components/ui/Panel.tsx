"use client";

import { useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";
import { cn } from "@/lib/utils";

interface PanelProps {
  title?: string;
  collapsible?: boolean;
  defaultCollapsed?: boolean;
  actions?: React.ReactNode;
  className?: string;
  children: React.ReactNode;
}

export function Panel({
  title,
  collapsible = false,
  defaultCollapsed = false,
  actions,
  className,
  children,
}: PanelProps) {
  const [collapsed, setCollapsed] = useState(defaultCollapsed);

  return (
    <div
      className={cn(
        "rounded-xl border border-gray-200 bg-white shadow-sm",
        className
      )}
    >
      {title && (
        <div className="flex items-center justify-between border-b border-gray-100 px-5 py-3">
          <div className="flex items-center gap-2">
            {collapsible && (
              <button
                onClick={() => setCollapsed(!collapsed)}
                className="rounded p-0.5 text-gray-400 hover:text-gray-600"
                aria-label={collapsed ? "Expand panel" : "Collapse panel"}
                aria-expanded={!collapsed}
              >
                {collapsed ? (
                  <ChevronDown className="h-4 w-4" />
                ) : (
                  <ChevronUp className="h-4 w-4" />
                )}
              </button>
            )}
            <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
          </div>
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
      )}
      {!collapsed && <div className="p-5">{children}</div>}
    </div>
  );
}
