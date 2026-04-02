'use client';

import React, { useState } from 'react';
import { Sidebar } from './layout/Sidebar';
import { theme } from '@/design-system/theme';

export default function AppLayout({ children }: { children: React.ReactNode }) {
    const [collapsed, setCollapsed] = useState(false);

    return (
        <div style={{ display: 'flex', minHeight: '100vh', backgroundColor: theme.colors.background.primary }}>
            {/* Sidebar */}
            <Sidebar />

            {/* Main Content - Dynamically adjusts to sidebar width */}
            <main
                style={{
                    flex: 1,
                    backgroundColor: theme.colors.background.primary,
                    minHeight: '100vh',
                }}
            >
                {children}
            </main>
        </div>
    );
}
