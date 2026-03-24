/**
 * ARC-Hawk Design System - Typography
 * 
 * Type scale and font definitions for consistent text hierarchy
 */

// ============================================
// FONT FAMILIES
// ============================================

export const fontFamily = {
    primary: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto", sans-serif',
    mono: '"JetBrains Mono", "Fira Code", "Courier New", monospace',
} as const;

// ============================================
// FONT SIZES
// ============================================

export const fontSize = {
    xs: '0.75rem',      // 12px (increased from 11px)
    sm: '0.875rem',     // 14px (increased from 13px)
    base: '1rem',       // 16px (increased from 14px)
    lg: '1rem',         // 16px
    xl: '1.125rem',     // 18px
    '2xl': '1.5rem',    // 24px
    '3xl': '1.875rem',  // 30px
    '4xl': '2.25rem',   // 36px
} as const;

// ============================================
// FONT WEIGHTS
// ============================================

export const fontWeight = {
    normal: 400,
    medium: 500,
    semibold: 600,
    bold: 700,
    extrabold: 800,
} as const;

// ============================================
// LINE HEIGHTS
// ============================================

export const lineHeight = {
    tight: 1.2,
    snug: 1.4,
    normal: 1.6,
    relaxed: 1.8,
    loose: 2,
} as const;

// ============================================
// LETTER SPACING
// ============================================

export const letterSpacing = {
    tighter: '-0.05em',
    tight: '-0.03em',
    normal: '0',
    wide: '0.025em',
    wider: '0.05em',
    widest: '0.1em',
} as const;

// ============================================
// TEXT STYLES (Composite)
// ============================================

export const textStyles = {
    // Headings
    h1: {
        fontSize: fontSize['4xl'],
        fontWeight: fontWeight.extrabold,
        lineHeight: lineHeight.tight,
        letterSpacing: letterSpacing.tight,
    },
    h2: {
        fontSize: fontSize['3xl'],
        fontWeight: fontWeight.bold,
        lineHeight: lineHeight.tight,
        letterSpacing: letterSpacing.tight,
    },
    h3: {
        fontSize: fontSize['2xl'],
        fontWeight: fontWeight.bold,
        lineHeight: lineHeight.snug,
    },
    h4: {
        fontSize: fontSize.xl,
        fontWeight: fontWeight.semibold,
        lineHeight: lineHeight.snug,
    },
    h5: {
        fontSize: fontSize.lg,
        fontWeight: fontWeight.semibold,
        lineHeight: lineHeight.normal,
    },

    // Body text
    body: {
        fontSize: fontSize.base,
        fontWeight: fontWeight.normal,
        lineHeight: lineHeight.normal,
    },
    bodyLarge: {
        fontSize: fontSize.lg,
        fontWeight: fontWeight.normal,
        lineHeight: lineHeight.relaxed,
    },
    bodySmall: {
        fontSize: fontSize.sm,
        fontWeight: fontWeight.normal,
        lineHeight: lineHeight.normal,
    },

    // Labels and UI text
    label: {
        fontSize: fontSize.sm,
        fontWeight: fontWeight.semibold,
        lineHeight: lineHeight.tight,
        textTransform: 'uppercase' as const,
        letterSpacing: letterSpacing.wider,
    },
    caption: {
        fontSize: fontSize.xs,
        fontWeight: fontWeight.medium,
        lineHeight: lineHeight.normal,
    },

    // Code
    code: {
        fontFamily: fontFamily.mono,
        fontSize: fontSize.sm,
        fontWeight: fontWeight.medium,
    },
} as const;

// ============================================
// EXPORTS
// ============================================

export const typography = {
    fontFamily,
    fontSize,
    fontWeight,
    lineHeight,
    letterSpacing,
    textStyles,
} as const;

export default typography;
