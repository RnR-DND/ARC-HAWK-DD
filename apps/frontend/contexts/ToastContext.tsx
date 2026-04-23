'use client';

import React, { createContext, useContext, useState, useCallback } from 'react';
import { CheckCircle, XCircle, Info, X } from 'lucide-react';

type ToastType = 'success' | 'error' | 'info';

interface ToastItem {
    id: string;
    message: string;
    type: ToastType;
}

interface ToastContextValue {
    showToast: (message: string, type?: ToastType) => void;
}

const ToastContext = createContext<ToastContextValue>({ showToast: () => {} });

export function useToast() {
    return useContext(ToastContext);
}

export function ToastProvider({ children }: { children: React.ReactNode }) {
    const [toasts, setToasts] = useState<ToastItem[]>([]);

    const showToast = useCallback((message: string, type: ToastType = 'info') => {
        const id = `${Date.now()}-${Math.random()}`;
        setToasts(prev => [...prev, { id, message, type }]);
        setTimeout(() => {
            setToasts(prev => prev.filter(t => t.id !== id));
        }, 4000);
    }, []);

    const dismiss = (id: string) => setToasts(prev => prev.filter(t => t.id !== id));

    return (
        <ToastContext.Provider value={{ showToast }}>
            {children}
            <div
                role="region"
                aria-label="Notifications"
                aria-live="polite"
                className="fixed top-5 right-5 z-[9999] flex flex-col gap-2 w-80 pointer-events-none"
            >
                {toasts.map(toast => (
                    <ToastBubble key={toast.id} toast={toast} onDismiss={() => dismiss(toast.id)} />
                ))}
            </div>
        </ToastContext.Provider>
    );
}

const CONFIG: Record<ToastType, { bg: string; Icon: React.FC<{ className?: string }> }> = {
    success: { bg: 'bg-green-600', Icon: CheckCircle },
    error:   { bg: 'bg-red-600',   Icon: XCircle },
    info:    { bg: 'bg-blue-600',  Icon: Info },
};

function ToastBubble({ toast, onDismiss }: { toast: ToastItem; onDismiss: () => void }) {
    const { bg, Icon } = CONFIG[toast.type];
    return (
        <div
            role="alert"
            className={`pointer-events-auto flex items-center gap-3 px-4 py-3 rounded-lg shadow-lg text-sm font-medium text-white ${bg}`}
        >
            <Icon className="w-4 h-4 shrink-0" />
            <span className="flex-1">{toast.message}</span>
            <button
                onClick={onDismiss}
                aria-label="Dismiss notification"
                className="opacity-70 hover:opacity-100 transition-opacity"
            >
                <X className="w-4 h-4" />
            </button>
        </div>
    );
}
