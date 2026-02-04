'use client';

import React, { ReactNode, useState } from 'react';
import { motion } from 'framer-motion';
import { Sidebar } from './Sidebar';
import { ScanContextBar } from './ScanContextBar';
import { Plus, Play, FileText, Bell, User, Shield, Settings } from 'lucide-react';
import Link from 'next/link';
import { AddSourceModal } from '../sources/AddSourceModal';
import { ScanConfigModal } from '../scans/ScanConfigModal';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from '@/components/ui/dropdown-menu';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import ErrorBoundary from '../ErrorBoundary';

interface GlobalLayoutProps {
    children: ReactNode;
}

export function GlobalLayout({ children }: GlobalLayoutProps) {
    const [isAddSourceOpen, setIsAddSourceOpen] = useState(false);
    const [isRunScanOpen, setIsRunScanOpen] = useState(false);

    return (
        <div className="flex h-screen bg-background">
            {/* Left Navigation */}
            <Sidebar />

            {/* Main Content Area */}
            <div className="flex-1 flex flex-col overflow-hidden">
                {/* Top Bar */}
                <motion.header
                    className="bg-card border-b shadow-sm"
                >
                    <div className="flex items-center justify-between px-6 py-4">
                        {/* Left: Brand & Title */}
                        <div className="flex items-center gap-6">
                            <div className="flex items-center gap-3">
                                <div className="p-2 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg">
                                    <Shield className="w-6 h-6 text-white" />
                                </div>
                                <div>
                                    <h1 className="text-xl font-bold bg-gradient-to-r from-blue-400 to-purple-400 bg-clip-text text-transparent">
                                        ARC-Hawk
                                    </h1>
                                    <p className="text-xs text-slate-500">Enterprise PII Governance</p>
                                </div>
                            </div>
                        </div>

                        {/* Right: User Actions */}
                        <div className="flex items-center gap-4">
                            {/* Quick Actions */}
                            <div className="hidden md:flex items-center gap-3">
                                <motion.button
                                    whileHover={{ scale: 1.05 }}
                                    whileTap={{ scale: 0.95 }}
                                    onClick={() => setIsAddSourceOpen(true)}
                                    className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-blue-600 to-blue-700 hover:from-blue-700 hover:to-blue-800 text-white rounded-lg text-sm font-medium transition-all shadow-lg hover:shadow-xl"
                                >
                                    <Plus className="w-4 h-4" />
                                    <span>Add Source</span>
                                </motion.button>

                                <motion.button
                                    whileHover={{ scale: 1.05 }}
                                    whileTap={{ scale: 0.95 }}
                                    onClick={() => setIsRunScanOpen(true)}
                                    className="flex items-center gap-2 px-4 py-2 bg-gradient-to-r from-emerald-600 to-emerald-700 hover:from-emerald-700 hover:to-emerald-800 text-white rounded-lg text-sm font-medium transition-all shadow-lg hover:shadow-xl"
                                >
                                    <Play className="w-4 h-4" />
                                    <span>Run Scan</span>
                                </motion.button>
                            </div>

                            {/* Navigation Links */}
                            <div className="flex items-center gap-2">
                                <Link
                                    href="/reports"
                                    className="flex items-center gap-2 px-3 py-2 bg-secondary hover:bg-accent text-slate-300 hover:text-white rounded-lg text-sm font-medium transition-all"
                                >
                                    <FileText className="w-4 h-4" />
                                    <span className="hidden lg:inline">Reports</span>
                                </Link>

                                <Link
                                    href="/settings"
                                    className="flex items-center gap-2 px-3 py-2 bg-secondary hover:bg-accent text-slate-300 hover:text-white rounded-lg text-sm font-medium transition-all"
                                >
                                    <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" className="lucide lucide-settings"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15-.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.38a2 2 0 0 0-.73-2.73l-.15-.1a2 2 0 0 1-1-1.72v-.51a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" /><circle cx="12" cy="12" r="3" /></svg>
                                    <span className="hidden lg:inline">Settings</span>
                                </Link>
                            </div>

                            {/* User Menu */}
                            <div className="flex items-center gap-3 pl-4 border-l border-border">
                                <Button variant="ghost" size="icon" className="relative">
                                    <Bell className="h-5 w-5" />
                                    <Badge
                                        variant="destructive"
                                        className="absolute -top-1 -right-1 h-5 w-5 flex items-center justify-center p-0 text-[10px]"
                                    >
                                        3
                                    </Badge>
                                </Button>

                                <DropdownMenu>
                                    <DropdownMenuTrigger asChild>
                                        <Button variant="ghost" className="gap-3 h-auto">
                                            <Avatar className="h-8 w-8">
                                                <AvatarImage src="" />
                                                <AvatarFallback className="bg-gradient-to-br from-blue-500 to-purple-600 text-white">
                                                    AU
                                                </AvatarFallback>
                                            </Avatar>
                                            <div className="hidden md:block text-left">
                                                <div className="text-sm font-medium">Admin User</div>
                                                <div className="text-xs text-muted-foreground">admin@company.com</div>
                                            </div>
                                        </Button>
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end" className="w-56">
                                        <DropdownMenuLabel>My Account</DropdownMenuLabel>
                                        <DropdownMenuSeparator />
                                        <DropdownMenuItem>
                                            <User className="mr-2 h-4 w-4" />
                                            <span>Profile</span>
                                        </DropdownMenuItem>
                                        <DropdownMenuItem>
                                            <Settings className="mr-2 h-4 w-4" />
                                            <span>Settings</span>
                                        </DropdownMenuItem>
                                        <DropdownMenuSeparator />
                                        <DropdownMenuItem>
                                            <span>Log out</span>
                                        </DropdownMenuItem>
                                    </DropdownMenuContent>
                                </DropdownMenu>
                            </div>
                        </div>
                    </div>

                    {/* Scan Context Bar */}
                    <ScanContextBar />
                </motion.header>

                {/* Main Content */}
                <main className="flex-1 overflow-auto bg-muted/30">
                    <ErrorBoundary>
                        {children}
                    </ErrorBoundary>
                </main>
            </div>

            {/* Global Modals */}
            <AddSourceModal
                isOpen={isAddSourceOpen}
                onClose={() => setIsAddSourceOpen(false)}
            />
            <ScanConfigModal
                isOpen={isRunScanOpen}
                onClose={() => setIsRunScanOpen(false)}
                onRunScan={(config) => {
                    console.log('Running Scan:', config);
                    setIsRunScanOpen(false);
                }}
            />
        </div>
    );
}


