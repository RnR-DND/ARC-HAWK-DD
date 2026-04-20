'use client';

import React, { createContext, useContext, useState, ReactNode } from 'react';

function readSavedContext() {
    if (typeof window === 'undefined') return null;
    try {
        const saved = localStorage.getItem('arc-hawk-scan-context');
        return saved ? JSON.parse(saved) as Record<string, unknown> : null;
    } catch {
        return null;
    }
}

interface ScanContext {
    currentScanId: string | null;
    currentScanName: string | null;
    environment: 'PROD' | 'DEV' | 'QA' | null;
    zeroValueMode: boolean;
    setCurrentScan: (scanId: string, scanName: string, environment: 'PROD' | 'DEV' | 'QA') => void;
    clearScan: () => void;
    toggleZeroValueMode: () => void;
}

const ScanContextContext = createContext<ScanContext | undefined>(undefined);

export function ScanContextProvider({ children }: { children: ReactNode }) {
    const [currentScanId, setCurrentScanId] = useState<string | null>(() => (readSavedContext()?.scanId as string) ?? null);
    const [currentScanName, setCurrentScanName] = useState<string | null>(() => (readSavedContext()?.scanName as string) ?? null);
    const [environment, setEnvironment] = useState<'PROD' | 'DEV' | 'QA' | null>(() => (readSavedContext()?.environment as 'PROD' | 'DEV' | 'QA') ?? null);
    const [zeroValueMode, setZeroValueMode] = useState<boolean>(() => (readSavedContext()?.zeroValueMode as boolean) ?? true);

    const setCurrentScan = (scanId: string, scanName: string, env: 'PROD' | 'DEV' | 'QA') => {
        setCurrentScanId(scanId);
        setCurrentScanName(scanName);
        setEnvironment(env);

        localStorage.setItem('arc-hawk-scan-context', JSON.stringify({
            scanId,
            scanName,
            environment: env,
            zeroValueMode
        }));
    };

    const clearScan = () => {
        setCurrentScanId(null);
        setCurrentScanName(null);
        setEnvironment(null);
        localStorage.removeItem('arc-hawk-scan-context');
    };

    const toggleZeroValueMode = () => {
        const newMode = !zeroValueMode;
        setZeroValueMode(newMode);

        if (currentScanId) {
            localStorage.setItem('arc-hawk-scan-context', JSON.stringify({
                scanId: currentScanId,
                scanName: currentScanName,
                environment,
                zeroValueMode: newMode
            }));
        }
    };

    return (
        <ScanContextContext.Provider
            value={{
                currentScanId,
                currentScanName,
                environment,
                zeroValueMode,
                setCurrentScan,
                clearScan,
                toggleZeroValueMode
            }}
        >
            {children}
        </ScanContextContext.Provider>
    );
}

export function useScanContext() {
    const context = useContext(ScanContextContext);
    if (!context) {
        throw new Error('useScanContext must be used within ScanContextProvider');
    }
    return context;
}
