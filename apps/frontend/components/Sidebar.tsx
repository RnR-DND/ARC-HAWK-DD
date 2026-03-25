'use client';

import React, { useState } from 'react';
import { usePathname } from 'next/navigation';
import { theme } from '@/design-system/theme';
import {
    Shield,
    Flame,
    BarChart3,
    FolderOpen,
    GitBranch,
    Search,
    Settings,
    ChevronLeft,
    ChevronRight,
    BookOpen,
    Zap
} from 'lucide-react';

interface SidebarProps {
    collapsed: boolean;
    onToggle: () => void;
}

export default function Sidebar({ collapsed, onToggle }: SidebarProps) {
    const pathname = usePathname();

    const navItems = [
        {
            section: 'Compliance',
            items: [
                { icon: Shield, label: 'Compliance', href: '/compliance' },
                { icon: Flame, label: 'Analytics', href: '/analytics' },
                { icon: BarChart3, label: 'Posture', href: '/posture' },
            ]
        },
        {
            section: 'Data',
            items: [
                { icon: FolderOpen, label: 'Assets', href: '/assets' },
                { icon: GitBranch, label: 'Lineage', href: '/lineage' },
                { icon: Search, label: 'Findings', href: '/findings' },
            ]
        },
        {
            section: 'System',
            items: [
                { icon: Settings, label: 'Settings', href: '/settings' },
            ]
        }
    ];

    return (
        <aside
            style={{
                width: collapsed ? '64px' : '240px',
                height: '100vh',
                background: theme.colors.background.secondary,
                borderRight: `1px solid ${theme.colors.border.default}`,
                transition: 'width 0.2s ease',
                position: 'fixed',
                left: 0,
                top: 0,
                zIndex: 100,
                display: 'flex',
                flexDirection: 'column',
                overflow: 'hidden',
            }}
        >
            {/* Header */}
            <div
                style={{
                padding: collapsed ? '20px 16px' : '20px',
                borderBottom: `1px solid ${theme.colors.border.default}`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                    minHeight: '64px',
                }}
            >
                {!collapsed && (
                    <div style={{
                        display: 'flex',
                        alignItems: 'center',
                        gap: '12px',
                    }}>
                        <div
                            style={{
                                width: '32px',
                                height: '32px',
                                background: theme.colors.primary.DEFAULT,
                                borderRadius: '8px',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                            }}
                        >
                            <Zap size={18} color={theme.colors.text.inverse} strokeWidth={2} />
                        </div>
                        <div
                            style={{
                                fontSize: '16px',
                                fontWeight: 700,
                                color: theme.colors.text.primary,
                            }}
                        >
                            ARC-Hawk
                        </div>
                    </div>
                )}

                <button
                    onClick={onToggle}
                    style={{
                        background: 'transparent',
                        border: `1px solid ${theme.colors.border.default}`,
                        borderRadius: '6px',
                        width: '32px',
                        height: '32px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        cursor: 'pointer',
                        color: theme.colors.text.secondary,
                        transition: 'all 0.15s',
                    }}
                    onMouseEnter={(e) => {
                        e.currentTarget.style.borderColor = theme.colors.border.active;
                    }}
                    onMouseLeave={(e) => {
                        e.currentTarget.style.borderColor = theme.colors.border.default;
                    }}
                >
                    {collapsed ? <ChevronRight size={16} /> : <ChevronLeft size={16} />}
                </button>
            </div>

            {/* Navigation */}
            <nav
                style={{
                    flex: 1,
                    padding: collapsed ? '16px 12px' : '20px 16px',
                    overflowY: 'auto',
                    overflowX: 'hidden',
                }}
            >
                {navItems.map((group, groupIndex) => (
                    <div key={groupIndex} style={{ marginBottom: '24px' }}>
                        {!collapsed && (
                            <div
                                style={{
                                    fontSize: '11px',
                                    fontWeight: 700,
                                    color: theme.colors.text.muted,
                                    textTransform: 'uppercase',
                                    letterSpacing: '0.05em',
                                    marginBottom: '8px',
                                    paddingLeft: '12px',
                                }}
                            >
                                {group.section}
                            </div>
                        )}
                        {group.items.map((item, itemIndex) => (
                            <NavItem
                                key={itemIndex}
                                {...item}
                                collapsed={collapsed}
                                active={pathname === item.href}
                            />
                        ))}
                    </div>
                ))}
            </nav>

            {/* Footer */}
            <div
                style={{
                    padding: collapsed ? '12px' : '16px',
                    borderTop: `1px solid ${theme.colors.border.default}`,
                }}
            >
                <a
                    href="https://www.digitalindia.gov.in/press_release/dpdp-act-2023-upholds-privacy-while-preserving-transparency-under-rti/"
                    target="_blank"
                    rel="noopener noreferrer"
                    style={{
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: collapsed ? 'center' : 'flex-start',
                        gap: '8px',
                        padding: '10px',
                        borderRadius: '8px',
                        textDecoration: 'none',
                        color: theme.colors.primary.DEFAULT,
                        fontSize: '13px',
                        fontWeight: 500,
                        background: 'rgba(59, 130, 246, 0.05)',
                        border: '1px solid rgba(59, 130, 246, 0.1)',
                        transition: 'all 0.15s',
                    }}
                    onMouseEnter={(e) => {
                        e.currentTarget.style.background = 'rgba(59, 130, 246, 0.1)';
                    }}
                    onMouseLeave={(e) => {
                        e.currentTarget.style.background = 'rgba(59, 130, 246, 0.05)';
                    }}
                >
                    <BookOpen size={14} />
                    {!collapsed && <span>DPDPA Guide</span>}
                </a>
            </div>
        </aside>
    );
}

function NavItem({ icon: Icon, label, href, collapsed, active }: {
    icon: any;
    label: string;
    href: string;
    collapsed: boolean;
    active: boolean;
}) {
    const [isHovered, setIsHovered] = useState(false);

    return (
        <a
            href={href}
            style={{
                display: 'flex',
                alignItems: 'center',
                gap: '12px',
                padding: collapsed ? '12px' : '10px 12px',
                borderRadius: '8px',
                color: active ? theme.colors.primary.DEFAULT : theme.colors.text.secondary,
                textDecoration: 'none',
                fontSize: '14px',
                fontWeight: active ? 600 : 400,
                marginBottom: '4px',
                transition: 'all 0.15s',
                background: active ? `${theme.colors.primary.DEFAULT}10` : isHovered ? theme.colors.background.tertiary : 'transparent',
                borderLeft: active ? `2px solid ${theme.colors.primary.DEFAULT}` : '2px solid transparent',
                justifyContent: collapsed ? 'center' : 'flex-start',
            }}
            onMouseEnter={() => setIsHovered(true)}
            onMouseLeave={() => setIsHovered(false)}
        >
            <Icon size={18} strokeWidth={active ? 2 : 1.5} />
            {!collapsed && <span>{label}</span>}
        </a>
    );
}
