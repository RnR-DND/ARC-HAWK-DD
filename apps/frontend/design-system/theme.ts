export const theme = {
    colors: {
        // Backgrounds - Modern Dark Theme with gradients
        background: {
            primary: '#0F172A',   // Slate 900
            secondary: '#1E293B', // Slate 800
            tertiary: '#334155',  // Slate 700
            card: '#1E293B',
            overlay: 'rgba(15, 23, 42, 0.8)',
            gradient: 'linear-gradient(135deg, #0F172A 0%, #1E293B 100%)',
        },

        // Text - Enhanced typography
        text: {
            primary: '#F8FAFC',   // Slate 50
            secondary: '#CBD5E1', // Slate 300
            tertiary: '#94A3B8',  // Slate 400
            muted: '#64748B',     // Slate 500
            inverse: '#0F172A',
            accent: '#60A5FA',    // Blue 400
        },

        // Risk Levels (MANDATORY: Risk-First Color Language)
        risk: {
            critical: '#EF4444', // Red 500
            high: '#F97316',     // Orange 500
            medium: '#EAB308',   // Yellow 500
            low: '#22C55E',      // Green 500
            info: '#3B82F6',     // Blue 500
            none: '#64748B',     // Slate 500
        },

        // Brand / Interactive - Enhanced gradients
        primary: {
            DEFAULT: '#3B82F6', // Blue 500
            hover: '#2563EB',   // Blue 600
            active: '#1D4ED8',  // Blue 700
            text: '#FFFFFF',
            gradient: 'linear-gradient(135deg, #3B82F6 0%, #6366F1 100%)',
        },

        // Secondary brand colors
        secondary: {
            DEFAULT: '#8B5CF6', // Violet 500
            hover: '#7C3AED',   // Violet 600
            active: '#6D28D9',  // Violet 700
            gradient: 'linear-gradient(135deg, #8B5CF6 0%, #A855F7 100%)',
        },

        // Borders - Enhanced
        border: {
            default: '#334155', // Slate 700
            active: '#475569',  // Slate 600
            subtle: '#1E293B',  // Slate 800
            accent: '#60A5FA',  // Blue 400
        },

        // Status - Consistent with risk colors
        status: {
            success: '#22C55E', // Green 500
            warning: '#F59E0B',  // Amber 500
            error: '#EF4444',    // Red 500
            info: '#3B82F6',     // Blue 500
        },

        // Glassmorphism effects
        glass: {
            background: 'rgba(30, 41, 59, 0.8)',
            border: 'rgba(51, 65, 85, 0.3)',
            backdrop: 'blur(12px)',
        },
    },

    // Spacing & Layout
    layout: {
        sidebarWidth: '280px',
        headerHeight: '80px',
        containerMaxWidth: '1440px',
        borderRadius: {
            sm: '0.375rem',
            md: '0.5rem',
            lg: '0.75rem',
            xl: '1rem',
            '2xl': '1.5rem',
        },
    },

    // Typography
    fonts: {
        sans: 'Inter, system-ui, -apple-system, sans-serif',
        mono: 'JetBrains Mono, Fira Code, monospace',
    },

    // Shadows & Effects
    shadows: {
        sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
        md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
        lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
        xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
        glow: '0 0 20px rgba(59, 130, 246, 0.3)',
    },

    // Animations
    animations: {
        fadeIn: 'fadeIn 0.3s ease-in-out',
        slideUp: 'slideUp 0.3s ease-out',
        pulse: 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
    },
};

// Helper for risk colors
export const getRiskColor = (riskLevel: string) => {
    const level = riskLevel?.toLowerCase();
    switch (level) {
        case 'critical': return theme.colors.risk.critical;
        case 'high': return theme.colors.risk.high;
        case 'medium': return theme.colors.risk.medium;
        case 'low': return theme.colors.risk.low;
        case 'info': return theme.colors.risk.info;
        default: return theme.colors.risk.none;
    }
};

// Start Risk Badge Component Styles
export const riskBadgeStyles = {
    critical: 'bg-red-900/30 text-red-400 border-red-800',
    high: 'bg-orange-900/30 text-orange-400 border-orange-800',
    medium: 'bg-yellow-900/30 text-yellow-400 border-yellow-800',
    low: 'bg-emerald-900/30 text-emerald-400 border-emerald-800',
    info: 'bg-blue-900/30 text-blue-400 border-blue-800',
    default: 'bg-slate-800 text-slate-400 border-slate-700',
};
