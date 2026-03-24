/**
 * ARC-Hawk Design System - Colors (Dark Enterprise Theme)
 *
 * Risk-First, Dark Mode Palette.
 * Slate 900 backgrounds, vivid risk colors.
 */

// ============================================
// PRIMITIVES (Slate Palette)
// ============================================

export const neutral = {
    50: '#F8FAFC',
    100: '#F1F5F9',
    200: '#E2E8F0',
    300: '#CBD5E1',
    400: '#94A3B8',
    500: '#64748B',
    600: '#475569',
    700: '#334155',
    800: '#1E293B',
    900: '#0F172A', // Main Background
} as const;

export const brand = {
    primary: '#3B82F6',     // Blue 500
    hover: '#2563EB',       // Blue 600
} as const;

export const status = {
    success: '#10B981',     // Emerald 500
    warning: '#F59E0B',     // Amber 500
    risk: '#DC2626',        // Red 600
    info: '#3B82F6',        // Blue 500
} as const;

// ============================================
// SEMANTIC MAPPINGS
// ============================================

export const background = {
    primary: '#0F172A',     // Slate 900
    surface: '#1E293B',     // Slate 800 (Card)
    elevated: '#334155',    // Slate 700 (Hover)
    muted: '#1E293B',       // Slate 800 - was neutral[100]
    card: '#1E293B',
} as const;

export const border = {
    default: '#334155',     // Slate 700
    subtle: '#1E293B',      // Slate 800
    strong: '#475569',      // Slate 600
} as const;

export const text = {
    primary: '#F8FAFC',     // Slate 50
    secondary: '#CBD5E1',   // Slate 300
    muted: '#94A3B8',       // Slate 400
    inverse: '#0F172A',
} as const;

export const nodeColors = {
    system: '#3B82F6',      // Blue
    asset: '#CBD5E1',       // Slate 300
    pii: '#EAB308',         // Amber 500
    sensitive: '#DC2626',   // Red 600
    category: '#94A3B8',    // Slate 400
} as const;

// ============================================
// THEME EXPORTS
// ============================================

export const colors = {
    neutral,
    brand,
    status,
    background,
    border,
    text,
    nodeColors,

    // Backward compatibility
    state: {
        risk: status.risk,
        success: status.success,
        warning: status.warning,
        info: status.info,
    },
    semantic: {
        background,
        border,
        text,
        state: status,
        node: nodeColors,
    },
    // Keep these but they shouldn't be main drivers
    blue: {
        50: '#EFF6FF', 100: '#DBEAFE', 200: '#BFDBFE', 300: '#93C5FD',
        400: '#60A5FA', 500: '#3B82F6', 600: '#2563EB', 700: '#1D4ED8',
        800: '#1E40AF', 900: '#1E3A8A'
    },
    red: {
        50: '#FEF2F2', 100: '#FEE2E2', 200: '#FECACA', 300: '#FCA5A5',
        400: '#F87171', 500: '#EF4444', 600: '#DC2626', 700: '#B91C1C',
        800: '#991B1B', 900: '#7F1D1D'
    },
    amber: {
        50: '#FFFBEB', 100: '#FEF3C7', 200: '#FDE68A', 300: '#FCD34D',
        400: '#FBBF24', 500: '#F59E0B', 600: '#D97706', 700: '#B45309',
        800: '#92400E', 900: '#78350F'
    },
    emerald: {
        50: '#ECFDF5', 100: '#D1FAE5', 200: '#A7F3D0', 300: '#6EE7B7',
        400: '#34D399', 500: '#10B981', 600: '#059669', 700: '#047857',
        800: '#065F46', 900: '#064E3B'
    },
    indigo: {
        50: '#EEF2FF', 100: '#E0E7FF', 200: '#C7D2FE', 300: '#A5B4FC',
        400: '#818CF8', 500: '#6366F1', 600: '#4F46E5', 700: '#4338CA',
        800: '#3730A3', 900: '#312E81'
    },
    purple: {
        50: '#FAF5FF', 100: '#F3E8FF', 200: '#E9D5FF', 300: '#D8B4FE',
        400: '#C084FC', 500: '#A855F7', 600: '#9333EA', 700: '#7E22CE',
        800: '#6B21A8', 900: '#581C87'
    },
} as const;

export function getNodeColor(type: string, riskScore: number = 0) {
    switch (type) {
        case 'system':
            return {
                bg: background.surface,
                border: brand.primary,
                text: text.primary
            };
        case 'asset':
        case 'file':
        case 'table':
            return {
                bg: background.surface,
                border: border.default,
                text: text.secondary
            };
        case 'finding':
            const isRisk = riskScore >= 90;
            return {
                bg: isRisk ? 'rgba(220, 38, 38, 0.1)' : background.surface, // Red tint
                border: isRisk ? status.risk : border.default,
                text: isRisk ? status.risk : text.secondary
            };
        case 'pii_category':
            return {
                bg: background.surface,
                border: status.warning,
                text: text.secondary
            }
        default:
            return { bg: background.surface, border: border.default, text: text.secondary };
    }
}

export function getEdgeColor(type: string) {
    if (type === 'EXPOSES') return status.risk;
    return '#475569'; // Slate 600
}

export default colors;
