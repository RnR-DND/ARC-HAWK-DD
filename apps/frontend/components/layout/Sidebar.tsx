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
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';

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

export function Sidebar() {
    const pathname = usePathname();

    const isActive = (href: string) => {
        if (href === '/') return pathname === '/';
        return pathname.startsWith(href);
    };

    return (
        <aside className="w-64 border-r bg-card flex flex-col h-screen">
            {/* Logo */}
            <div className="p-4 border-b">
                <div className="flex items-center gap-2">
                    <div className="text-lg font-bold">ARComply</div>
                    <div className="text-muted-foreground">▸</div>
                    <div className="text-sm font-semibold text-primary">ARC-HAWK</div>
                </div>
            </div>

            {/* Main Navigation */}
            <ScrollArea className="flex-1 p-3">
                <nav className="space-y-1">
                    {navigation.map((item) => {
                        const Icon = item.icon;
                        const active = isActive(item.href);

                        return (
                            <Button
                                key={item.name}
                                asChild
                                variant={active ? 'secondary' : 'ghost'}
                                className={cn(
                                    'w-full justify-start gap-3 font-medium',
                                    active && 'bg-accent'
                                )}
                            >
                                <Link href={item.href}>
                                    <Icon className="h-4 w-4" />
                                    <span className="flex-1 text-left">{item.name}</span>
                                    <kbd className="pointer-events-none inline-flex h-5 select-none items-center gap-1 rounded border bg-muted px-1.5 font-mono text-[10px] font-medium text-muted-foreground opacity-100">
                                        {item.shortcut}
                                    </kbd>
                                </Link>
                            </Button>
                        );
                    })}
                </nav>

                {/* System Section */}
                <div className="mt-8 pt-4">
                    <Separator className="mb-4" />
                    <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider px-3 mb-2">
                        System
                    </div>
                    <nav className="space-y-1">
                        {systemNav.map((item) => {
                            const Icon = item.icon;
                            const active = isActive(item.href);

                            return (
                                <Button
                                    key={item.name}
                                    asChild
                                    variant={active ? 'secondary' : 'ghost'}
                                    className={cn(
                                        'w-full justify-start gap-3 font-medium',
                                        active && 'bg-accent'
                                    )}
                                >
                                    <Link href={item.href}>
                                        <Icon className="h-4 w-4" />
                                        <span>{item.name}</span>
                                    </Link>
                                </Button>
                            );
                        })}
                    </nav>
                </div>
            </ScrollArea>

            {/* Footer */}
            <div className="p-4 border-t">
                <Button
                    asChild
                    variant="outline"
                    size="sm"
                    className="w-full justify-start gap-2 text-xs"
                >
                    <a
                        href="https://digitalindia.gov.in/dpdpa"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        <span>📖</span>
                        <span>DPDPA Guide</span>
                    </a>
                </Button>
            </div>
        </aside>
    );
}
