'use client';

import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

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
    const [currentScanId, setCurrentScanId] = useState<string | null>(null);
    const [currentScanName, setCurrentScanName] = useState<string | null>(null);
    const [environment, setEnvironment] = useState<'PROD' | 'DEV' | 'QA' | null>(null);
    const [zeroValueMode, setZeroValueMode] = useState(true); // Default: hide values

    // Persist to localStorage
    useEffect(() => {
        const saved = localStorage.getItem('arc-hawk-scan-context');
        if (saved) {
            try {
                const parsed = JSON.parse(saved);
                setCurrentScanId(parsed.scanId);
                setCurrentScanName(parsed.scanName);
                setEnvironment(parsed.environment);
                setZeroValueMode(parsed.zeroValueMode ?? true);
            } catch (e) {
                console.error('Failed to parse scan context', e);
            }
        }
    }, []);

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
