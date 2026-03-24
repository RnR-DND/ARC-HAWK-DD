'use client';

import React, { useState } from 'react';
import Sidebar from './Sidebar';

export default function AppLayout({ children }: { children: React.ReactNode }) {
    const [collapsed, setCollapsed] = useState(false);

    return (
        <div style={{ display: 'flex', minHeight: '100vh', backgroundColor: '#0f172a' }}>
            {/* Sidebar */}
            <Sidebar collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} />

            {/* Main Content - Dynamically adjusts to sidebar width */}
            <main
                style={{
                    flex: 1,
                    marginLeft: collapsed ? '64px' : '280px',
                    transition: 'margin-left 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                    backgroundColor: '#0f172a',
                    minHeight: '100vh',
                    width: collapsed ? 'calc(100vw - 64px)' : 'calc(100vw - 280px)',
                }}
            >
                {children}
            </main>
        </div>
    );
}
