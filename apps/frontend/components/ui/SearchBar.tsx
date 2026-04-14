"use client";

import { useState, useCallback, useEffect, useRef } from "react";
import { Search, X } from "lucide-react";
import { cn } from "@/lib/utils";

interface FilterChip {
  key: string;
  label: string;
  value: string;
}

interface SearchBarProps {
  placeholder?: string;
  value?: string;
  onChange: (value: string) => void;
  chips?: FilterChip[];
  onRemoveChip?: (key: string) => void;
  debounceMs?: number;
  className?: string;
}

export function SearchBar({
  placeholder = "Search...",
  value: controlledValue,
  onChange,
  chips = [],
  onRemoveChip,
  debounceMs = 300,
  className,
}: SearchBarProps) {
  const [localValue, setLocalValue] = useState(controlledValue || "");
  const timerRef = useRef<ReturnType<typeof setTimeout>>();

  useEffect(() => {
    if (controlledValue !== undefined) {
      setLocalValue(controlledValue);
    }
  }, [controlledValue]);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const v = e.target.value;
      setLocalValue(v);
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => onChange(v), debounceMs);
    },
    [onChange, debounceMs]
  );

  const handleClear = useCallback(() => {
    setLocalValue("");
    onChange("");
  }, [onChange]);

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-2 rounded-lg border border-gray-300 bg-white px-3 py-2 focus-within:border-blue-500 focus-within:ring-1 focus-within:ring-blue-500",
        className
      )}
    >
      <Search className="h-4 w-4 shrink-0 text-gray-400" aria-hidden="true" />
      {chips.map((chip) => (
        <span
          key={chip.key}
          className="inline-flex items-center gap-1 rounded-md bg-blue-50 px-2 py-0.5 text-xs font-medium text-blue-700"
        >
          {chip.label}: {chip.value}
          {onRemoveChip && (
            <button
              onClick={() => onRemoveChip(chip.key)}
              className="rounded-full hover:bg-blue-100"
              aria-label={`Remove filter: ${chip.label}`}
            >
              <X className="h-3 w-3" />
            </button>
          )}
        </span>
      ))}
      <input
        type="search"
        value={localValue}
        onChange={handleChange}
        placeholder={placeholder}
        className="min-w-0 flex-1 border-none bg-transparent text-sm outline-none placeholder:text-gray-400"
        aria-label={placeholder}
      />
      {localValue && (
        <button
          onClick={handleClear}
          className="rounded-full p-0.5 text-gray-400 hover:text-gray-600"
          aria-label="Clear search"
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  );
}
