import React from 'react';
import { render, screen } from '@testing-library/react';
import '@testing-library/jest-dom';
import { MobileSidebarContent } from '../components/layout/Sidebar';

jest.mock('next/navigation', () => ({
    usePathname: () => '/',
}));

jest.mock('next/link', () => {
    const Link = ({ href, children, onClick, ...rest }: any) => (
        <a href={typeof href === 'string' ? href : '#'} onClick={onClick} {...rest}>
            {children}
        </a>
    );
    Link.displayName = 'NextLink';
    return Link;
});

describe('MobileSidebarContent (hamburger drawer)', () => {
    it('renders main navigation items inside the drawer', () => {
        render(<MobileSidebarContent />);

        // The drawer reuses the same nav items as the desktop sidebar.
        expect(screen.getByText('Dashboard')).toBeInTheDocument();
        expect(screen.getByText('Scans')).toBeInTheDocument();
        expect(screen.getByText('Findings')).toBeInTheDocument();
        expect(screen.getByText('Remediation')).toBeInTheDocument();
    });

    it('renders analytics section items', () => {
        render(<MobileSidebarContent />);
        expect(screen.getByText('Compliance')).toBeInTheDocument();
        // "Analytics" appears as both section header and nav link
        expect(screen.getAllByText('Analytics').length).toBeGreaterThanOrEqual(1);
        expect(screen.getByText('Reports')).toBeInTheDocument();
    });

    it('renders system section items', () => {
        render(<MobileSidebarContent />);
        expect(screen.getByText('Connectors')).toBeInTheDocument();
        expect(screen.getByText('Settings')).toBeInTheDocument();
    });

    it('renders DPDPA footer link', () => {
        render(<MobileSidebarContent />);
        const link = screen.getByText('DPDPA 2023 Act').closest('a');
        expect(link).toHaveAttribute('target', '_blank');
        expect(link).toHaveAttribute('rel', expect.stringContaining('noopener'));
    });
});
