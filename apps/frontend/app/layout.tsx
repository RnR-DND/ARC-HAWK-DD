import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { GlobalLayout } from '@/components/layout/GlobalLayout';
import { ScanContextProvider } from '@/contexts/ScanContext';
import { ToastProvider } from '@/contexts/ToastContext';

const inter = Inter({
    subsets: ['latin'],
    display: 'swap',
    variable: '--font-inter',
});

export const metadata: Metadata = {
    title: 'ARC-Hawk Enterprise Risk',
    description: 'Data Lineage and Risk Management',
};

export default function RootLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <html lang="en" suppressHydrationWarning>
            <body className={`${inter.className} antialiased`} suppressHydrationWarning>
                <ToastProvider>
                    <ScanContextProvider>
                        <GlobalLayout>{children}</GlobalLayout>
                    </ScanContextProvider>
                </ToastProvider>
            </body>
        </html>
    );
}
