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
                    className="bg-white/80 backdrop-blur-md border-b border-slate-200/60 shadow-sm sticky top-0 z-50"
                >
                    <div className="flex items-center justify-between px-6 py-4">
                        {/* Left: Brand & Title */}
                        <div className="flex items-center gap-6">
                            <div className="flex items-center gap-3">
                                <div className="p-2 bg-gradient-to-br from-blue-600 to-purple-700 rounded-lg shadow-md shadow-blue-500/20">
                                    <Shield className="w-6 h-6 text-white" />
                                </div>
                                <div>
                                    <h1 className="text-xl font-bold bg-gradient-to-r from-blue-700 to-purple-700 bg-clip-text text-transparent">
                                        ARC-Hawk
                                    </h1>
                                    <p className="text-xs text-slate-500 font-medium tracking-wide">Enterprise PII Governance</p>
                                </div>
                            </div>
                        </div>

                        {/* Right: User Actions */}
                        <div className="flex items-center gap-4">
                            {/* Quick Actions */}
                            <div className="hidden md:flex items-center gap-3">
                                <motion.button
                                    whileHover={{ scale: 1.02 }}
                                    whileTap={{ scale: 0.98 }}
                                    onClick={() => setIsAddSourceOpen(true)}
                                    className="flex items-center gap-2 px-4 py-2 bg-white border border-slate-200 hover:border-blue-500/50 hover:bg-blue-50/50 text-slate-700 hover:text-blue-700 rounded-lg text-sm font-medium transition-all shadow-sm"
                                >
                                    <Plus className="w-4 h-4" />
                                    <span>Add Source</span>
                                </motion.button>

                                <motion.button
                                    whileHover={{ scale: 1.02 }}
                                    whileTap={{ scale: 0.98 }}
                                    onClick={() => setIsRunScanOpen(true)}
                                    className="flex items-center gap-2 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white rounded-lg text-sm font-medium transition-all shadow-md shadow-blue-600/20"
                                >
                                    <Play className="w-4 h-4" />
                                    <span>Run Scan</span>
                                </motion.button>
                            </div>

                            {/* Navigation Links */}
                            <div className="flex items-center gap-2 border-l border-slate-200 pl-4 ml-2">
                                <Link
                                    href="/reports"
                                    className="flex items-center justify-center w-9 h-9 text-slate-500 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-all"
                                    title="Reports"
                                >
                                    <FileText className="w-4 h-4" />
                                </Link>

                                <Link
                                    href="/settings"
                                    className="flex items-center justify-center w-9 h-9 text-slate-500 hover:text-blue-600 hover:bg-blue-50 rounded-lg transition-all"
                                    title="Settings"
                                >
                                    <Settings className="w-4 h-4" />
                                </Link>
                            </div>

                            {/* User Menu */}
                            <div className="flex items-center gap-3 pl-2">
                                <Button variant="ghost" size="icon" className="relative text-slate-500 hover:text-blue-600 hover:bg-blue-50">
                                    <Bell className="h-5 w-5" />
                                    <Badge
                                        variant="destructive"
                                        className="absolute -top-1 -right-1 h-4 w-4 flex items-center justify-center p-0 text-[10px] shadow-sm"
                                    >
                                        3
                                    </Badge>
                                </Button>

                                <DropdownMenu>
                                    <DropdownMenuTrigger asChild>
                                        <Button variant="ghost" className="gap-3 h-auto pl-2 pr-4 hover:bg-slate-50 rounded-full border border-transparent hover:border-slate-200">
                                            <Avatar className="h-8 w-8 ring-2 ring-white shadow-sm">
                                                <AvatarImage src="" />
                                                <AvatarFallback className="bg-gradient-to-br from-blue-600 to-purple-700 text-white text-xs">
                                                    AU
                                                </AvatarFallback>
                                            </Avatar>
                                            <div className="hidden md:block text-left">
                                                <div className="text-sm font-semibold text-slate-700">Admin User</div>
                                                <div className="text-[10px] text-slate-500 font-medium">admin@company.com</div>
                                            </div>
                                        </Button>
                                    </DropdownMenuTrigger>
                                    <DropdownMenuContent align="end" className="w-56 mt-2">
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
                                        <DropdownMenuItem className="text-red-600 focus:text-red-600 focus:bg-red-50">
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
                <main className="flex-1 overflow-auto bg-white border-l border-slate-200">
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


