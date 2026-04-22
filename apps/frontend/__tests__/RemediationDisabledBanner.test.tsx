import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';

// Mock Next.js
jest.mock('next/navigation', () => ({ usePathname: () => '/remediation', useRouter: () => ({}) }));
jest.mock('next/link', () => {
    const Link = ({ href, children, ...rest }: any) => <a href={href} {...rest}>{children}</a>;
    Link.displayName = 'Link';
    return Link;
});

// jest.mock is hoisted — pendingTask must be inlined here, not referenced from outer scope
jest.mock('@/services/auth.api', () => ({
    authApi: {
        getProfile: jest.fn().mockResolvedValue({ id: 'user-1', email: 'admin@example.com' }),
    },
}));

jest.mock('@/services/remediation.api', () => ({
    remediationApi: {
        getRemediationHistory: jest.fn().mockResolvedValue({
            history: [{
                id: 'task-1',
                finding_id: 'finding-1',
                asset_name: 'users.csv',
                asset_path: '/data/users.csv',
                pii_type: 'EMAIL',
                risk_level: 'High',
                action: 'MASK',
                status: 'PENDING',
                executed_at: '2026-01-01T00:00:00Z',
            }],
        }),
        executeRemediation: jest.fn(),
        getSOPs: jest.fn().mockResolvedValue({ sops: [] }),
        previewEscalation: jest.fn(),
        runEscalation: jest.fn(),
        rollback: jest.fn(),
    },
}));

import RemediationPage from '../app/remediation/page';
import { remediationApi } from '@/services/remediation.api';

// Suppress expected console.error noise from the component's error handlers
beforeAll(() => { jest.spyOn(console, 'error').mockImplementation(() => {}); });
afterAll(() => { (console.error as jest.Mock).mockRestore?.(); });

describe('Remediation disabled banner', () => {
    it('is hidden when remediation is enabled', async () => {
        (remediationApi.executeRemediation as jest.Mock).mockResolvedValue({ success: true });
        render(<RemediationPage />);
        await waitFor(() =>
            expect(screen.queryByText('Loading remediation dashboard...')).not.toBeInTheDocument()
        );
        expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    });

    it('appears when executeRemediation returns 503 remediation_disabled', async () => {
        const err: any = new Error('Service Unavailable');
        err.response = { status: 503, data: { error: 'remediation_disabled' } };
        (remediationApi.executeRemediation as jest.Mock).mockRejectedValue(err);

        render(<RemediationPage />);
        await waitFor(() =>
            expect(screen.queryByText('Loading remediation dashboard...')).not.toBeInTheDocument()
        );

        // PENDING task in history enables the Run All Pending button
        const runBtn = screen.getByRole('button', { name: /run.*pending/i });
        fireEvent.click(runBtn);

        await waitFor(() =>
            expect(screen.getByRole('alert')).toBeInTheDocument()
        );
        expect(screen.getByText('Remediation is disabled on this deployment')).toBeInTheDocument();
    });

    it('does NOT show banner for non-503 errors', async () => {
        const err: any = new Error('Internal Server Error');
        err.response = { status: 500, data: { error: 'internal_error' } };
        (remediationApi.executeRemediation as jest.Mock).mockRejectedValue(err);

        render(<RemediationPage />);
        await waitFor(() =>
            expect(screen.queryByText('Loading remediation dashboard...')).not.toBeInTheDocument()
        );

        const runBtn = screen.getByRole('button', { name: /run.*pending/i });
        fireEvent.click(runBtn);

        await new Promise((r) => setTimeout(r, 150));
        expect(screen.queryByRole('alert')).not.toBeInTheDocument();
    });
});
