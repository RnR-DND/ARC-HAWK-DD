"use client";

import { cn } from "@/lib/utils";
import { Inbox } from "lucide-react";

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  message: string;
  action?: React.ReactNode;
  className?: string;
}

export function EmptyState({
  icon,
  title,
  message,
  action,
  className,
}: EmptyStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-gray-200 bg-white px-6 py-16 text-center",
        className
      )}
    >
      <div className="mb-4 rounded-full bg-gray-100 p-3 text-gray-400">
        {icon || <Inbox className="h-8 w-8" />}
      </div>
      <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
      <p className="mt-1 max-w-sm text-sm text-gray-500">{message}</p>
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
