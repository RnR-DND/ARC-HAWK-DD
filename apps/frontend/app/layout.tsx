import type { Metadata } from 'next';
import { Inter } from 'next/font/google';
import './globals.css';
import { GlobalLayout } from '@/components/layout/GlobalLayout';
import { ScanContextProvider } from '@/contexts/ScanContext';

const inter = Inter({ subsets: ['latin'] });

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
        <html lang="en">
            <body className={inter.className}>
                <ScanContextProvider>
                    <GlobalLayout>{children}</GlobalLayout>
                </ScanContextProvider>
            </body>
        </html>
    );
}
