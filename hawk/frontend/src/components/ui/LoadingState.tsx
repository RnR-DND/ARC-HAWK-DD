"use client";

import { cn } from "@/lib/utils";

interface LoadingStateProps {
  rows?: number;
  type?: "table" | "cards" | "chart";
  className?: string;
}

export function LoadingState({
  rows = 5,
  type = "table",
  className,
}: LoadingStateProps) {
  if (type === "cards") {
    return (
      <div className={cn("grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4", className)}>
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="animate-pulse rounded-xl border border-gray-200 bg-white p-5"
          >
            <div className="h-3 w-20 rounded bg-gray-200" />
            <div className="mt-3 h-6 w-16 rounded bg-gray-200" />
            <div className="mt-2 h-2 w-24 rounded bg-gray-200" />
            <div className="mt-4 h-10 w-full rounded bg-gray-100" />
          </div>
        ))}
      </div>
    );
  }

  if (type === "chart") {
    return (
      <div className={cn("animate-pulse rounded-xl border border-gray-200 bg-white p-5", className)}>
        <div className="h-4 w-32 rounded bg-gray-200" />
        <div className="mt-4 h-64 w-full rounded bg-gray-100" />
      </div>
    );
  }

  return (
    <div className={cn("animate-pulse rounded-xl border border-gray-200 bg-white", className)}>
      <div className="border-b border-gray-100 px-5 py-3">
        <div className="h-4 w-40 rounded bg-gray-200" />
      </div>
      <div className="p-5 space-y-3">
        {Array.from({ length: rows }).map((_, i) => (
          <div key={i} className="flex gap-4">
            <div className="h-4 w-1/4 rounded bg-gray-100" />
            <div className="h-4 w-1/3 rounded bg-gray-100" />
            <div className="h-4 w-1/6 rounded bg-gray-100" />
            <div className="h-4 w-1/6 rounded bg-gray-100" />
          </div>
        ))}
      </div>
    </div>
  );
}
