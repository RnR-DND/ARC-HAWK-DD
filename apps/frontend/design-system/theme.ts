export const theme = {
    colors: {
        // Backgrounds - Modern Light Theme
        background: {
            primary: '#FFFFFF',   // Pure White
            secondary: '#F8FAFC', // Slate 50
            tertiary: '#F1F5F9',  // Slate 100
            card: '#FFFFFF',
            overlay: 'rgba(255, 255, 255, 0.8)',
            gradient: 'linear-gradient(135deg, #FFFFFF 0%, #F8FAFC 100%)',
        },

        // Text - Enhanced typography for Light Mode
        text: {
            primary: '#0F172A',   // Slate 900
            secondary: '#475569', // Slate 600
            tertiary: '#64748B',  // Slate 500
            muted: '#94A3B8',     // Slate 400
            inverse: '#FFFFFF',
            accent: '#3B82F6',    // Blue 500
        },

        // Risk Levels (MANDATORY: Risk-First Color Language)
        risk: {
            critical: '#EF4444', // Red 500
            high: '#F97316',     // Orange 500
            medium: '#EAB308',   // Yellow 500
            low: '#22C55E',      // Green 500
            info: '#3B82F6',     // Blue 500
            none: '#94A3B8',     // Slate 400
        },

        // Brand / Interactive
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

        // Borders - Enhanced Light Contrast
        border: {
            default: '#E2E8F0', // Slate 200
            active: '#CBD5E1',  // Slate 300
            subtle: '#F1F5F9',  // Slate 100
            accent: '#3B82F6',  // Blue 500
        },

        // Status - Consistent with risk colors
        status: {
            success: '#22C55E', // Green 500
            warning: '#F59E0B',  // Amber 500
            error: '#EF4444',    // Red 500
            info: '#3B82F6',     // Blue 500
        },

        // Glassmorphism effects for Light Mode
        glass: {
            background: 'rgba(255, 255, 255, 0.7)',
            border: 'rgba(0, 0, 0, 0.05)',
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

    // Shadows & Effects - Standard Light Mode Shadows
    shadows: {
        sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
        md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06)',
        lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
        xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04)',
        glow: '0 0 20px rgba(59, 130, 246, 0.15)',
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

// Start Risk Badge Component Styles - Updated for Light Mode
export const riskBadgeStyles = {
    critical: 'bg-red-50 text-red-700 border-red-200',
    high: 'bg-orange-50 text-orange-700 border-orange-200',
    medium: 'bg-yellow-50 text-yellow-700 border-yellow-200',
    low: 'bg-emerald-50 text-emerald-700 border-emerald-200',
    info: 'bg-blue-50 text-blue-700 border-blue-200',
    default: 'bg-slate-50 text-slate-600 border-slate-200',
};
