'use client';

import React from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
    LayoutDashboard,
    ScanSearch,
    Database,
    Search,
    Shield,
    Settings,
    Compass,
    ClipboardList,
    Regex,
    Plug,
    BarChart3,
    CheckSquare,
    FileText,
    Users,
} from 'lucide-react';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { cn } from '@/lib/utils';

const navigation = [
    { name: 'Dashboard', href: '/', icon: LayoutDashboard },
    { name: 'Discovery', href: '/discovery', icon: Compass },
    { name: 'Scans', href: '/scans', icon: ScanSearch },
    { name: 'Assets', href: '/assets', icon: Database },
    { name: 'Findings', href: '/findings', icon: Search },
    { name: 'Remediation', href: '/remediation', icon: Shield },
    { name: 'Audit Logs', href: '/audit', icon: ClipboardList },
];

const analyticsNav = [
    { name: 'Compliance', href: '/compliance', icon: CheckSquare },
    { name: 'Analytics', href: '/analytics', icon: BarChart3 },
    { name: 'Reports', href: '/reports', icon: FileText },
];

const systemNav = [
    { name: 'Connectors', href: '/settings/connectors', icon: Plug },
    { name: 'Regex Patterns', href: '/settings/regex', icon: Regex },
    { name: 'Users', href: '/settings/users', icon: Users },
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
                    <div className="text-lg font-bold">ARCompli</div>
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
                                </Link>
                            </Button>
                        );
                    })}
                </nav>

                {/* Analytics Section */}
                <div className="mt-6 pt-4">
                    <Separator className="mb-3" />
                    <div className="text-xs font-semibold text-muted-foreground uppercase tracking-wider px-3 mb-2">
                        Analytics
                    </div>
                    <nav className="space-y-1">
                        {analyticsNav.map((item) => {
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

                {/* System Section */}
                <div className="mt-6 pt-4">
                    <Separator className="mb-3" />
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
            <div className="p-3 border-t">
                <Button
                    asChild
                    variant="ghost"
                    size="sm"
                    className="w-full justify-start gap-2 text-xs text-muted-foreground hover:text-foreground"
                >
                    <a
                        href="https://www.digitalindia.gov.in/press_release/dpdp-act-2023-upholds-privacy-while-preserving-transparency-under-rti/"
                        target="_blank"
                        rel="noopener noreferrer"
                    >
                        <FileText className="h-3.5 w-3.5" />
                        <span>DPDPA 2023 Act</span>
                    </a>
                </Button>
            </div>
        </aside>
    );
}
