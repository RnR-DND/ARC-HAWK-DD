'use client';

import React, { useState } from 'react';
import Sidebar from './Sidebar';
import { theme } from '@/design-system/theme';

export default function AppLayout({ children }: { children: React.ReactNode }) {
    const [collapsed, setCollapsed] = useState(false);

    return (
        <div style={{ display: 'flex', minHeight: '100vh', backgroundColor: theme.colors.background.primary }}>
            {/* Sidebar */}
            <Sidebar collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} />

            {/* Main Content - Dynamically adjusts to sidebar width */}
            <main
                style={{
                    flex: 1,
                    marginLeft: collapsed ? '64px' : '280px',
                    transition: 'margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                    backgroundColor: theme.colors.background.primary,
                    minHeight: '100vh',
                    width: collapsed ? 'calc(100vw - 64px)' : 'calc(100vw - 280px)',
                }}
            >
                {children}
            </main>
        </div>
    );
}
