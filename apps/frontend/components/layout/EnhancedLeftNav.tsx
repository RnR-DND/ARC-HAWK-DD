'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
    LayoutDashboard,
    ScanSearch,
    Database,
    Search,
    GitBranch,
    Shield,
    History,
    Settings
} from 'lucide-react';

const navigation = [
    { name: 'Dashboard', href: '/', icon: LayoutDashboard, shortcut: '1' },
    { name: 'Scans', href: '/scans', icon: ScanSearch, shortcut: '2' },
    { name: 'Assets', href: '/assets', icon: Database, shortcut: '3' },
    { name: 'Findings', href: '/findings', icon: Search, shortcut: '4' },
    { name: 'Lineage', href: '/lineage', icon: GitBranch, shortcut: '5' },
    { name: 'Remediation', href: '/remediation', icon: Shield, shortcut: '6' },
    { name: 'History', href: '/history', icon: History, shortcut: '7' },
];

const systemNav = [
    { name: 'Settings', href: '/settings', icon: Settings },
];

export function EnhancedLeftNav() {
    const pathname = usePathname();

    const isActive = (href: string) => {
        if (href === '/') return pathname === '/';
        return pathname.startsWith(href);
    };

    return (
        <aside className="w-64 bg-card border-r border-border flex flex-col h-screen sticky top-0">
            {/* Logo */}
            <div className="p-4 border-b border-border">
                <div className="flex items-center gap-2">
                    <div className="text-lg font-bold text-white">ARComply</div>
                    <div className="text-muted-foreground">▸</div>
                    <div className="text-sm font-semibold text-blue-400">ARC-HAWK</div>
                </div>
            </div>

            {/* Main Navigation */}
            <nav className="flex-1 p-3 overflow-y-auto">
                <div className="space-y-1">
                    {navigation.map((item) => {
                        const Icon = item.icon;
                        const active = isActive(item.href);

                        return (
                            <Link
                                key={item.name}
                                href={item.href}
                                className={`
                  flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all
                  ${active
                                        ? 'bg-blue-500/10 text-blue-400 border-l-2 border-blue-500'
                                        : 'text-muted-foreground hover:text-slate-200 hover:bg-secondary'
                                    }
                `}
                            >
                                <Icon className="w-4 h-4" />
                                <span className="flex-1">{item.name}</span>
                                <kbd className="px-1.5 py-0.5 text-xs bg-muted rounded text-slate-500">
                                    {item.shortcut}
                                </kbd>
                            </Link>
                        );
                    })}
                </div>

                {/* System Section */}
                <div className="mt-8 pt-4 border-t border-border">
                    <div className="text-xs font-semibold text-slate-500 uppercase tracking-wider px-3 mb-2">
                        System
                    </div>
                    <div className="space-y-1">
                        {systemNav.map((item) => {
                            const Icon = item.icon;
                            const active = isActive(item.href);

                            return (
                                <Link
                                    key={item.name}
                                    href={item.href}
                                    className={`
                    flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-all
                    ${active
                                            ? 'bg-blue-500/10 text-blue-400 border-l-2 border-blue-500'
                                            : 'text-muted-foreground hover:text-slate-200 hover:bg-secondary'
                                        }
                  `}
                                >
                                    <Icon className="w-4 h-4" />
                                    <span>{item.name}</span>
                                </Link>
                            );
                        })}
                    </div>
                </div>
            </nav>

            {/* Footer */}
            <div className="p-4 border-t border-border">
                <a
                    href="https://digitalindia.gov.in/dpdpa"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex items-center gap-2 px-3 py-2 rounded-lg text-xs font-medium bg-blue-500/5 border border-blue-500/10 text-blue-400 hover:bg-blue-500/10 transition-all"
                >
                    <span>📖</span>
                    <span>DPDPA Guide</span>
                </a>
            </div>
        </aside>
    );
}
