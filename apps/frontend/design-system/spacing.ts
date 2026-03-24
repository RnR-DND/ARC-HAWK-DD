/**
 * ARC-Hawk Design System - Spacing
 * 
 * 8px-based spacing scale for consistent layout rhythm
 */

// ============================================
// SPACING SCALE (8px base unit)
// ============================================

export const spacing = {
    0: '0',
    1: '0.25rem',   // 4px
    2: '0.5rem',    // 8px
    3: '0.75rem',   // 12px
    4: '1rem',      // 16px
    5: '1.25rem',   // 20px
    6: '1.5rem',    // 24px
    8: '2rem',      // 32px
    10: '2.5rem',   // 40px
    12: '3rem',     // 48px
    16: '4rem',     // 64px
    20: '5rem',     // 80px
    24: '6rem',     // 96px
    32: '8rem',     // 128px
} as const;

// ============================================
// SEMANTIC SPACING
// ============================================

export const semanticSpacing = {
    // Component internal spacing
    componentXs: spacing[1],    // 4px
    componentSm: spacing[2],    // 8px
    componentMd: spacing[4],    // 16px
    componentLg: spacing[6],    // 24px
    componentXl: spacing[8],    // 32px

    // Layout spacing
    layoutSm: spacing[6],       // 24px
    layoutMd: spacing[8],       // 32px
    layoutLg: spacing[12],      // 48px
    layoutXl: spacing[16],      // 64px

    // Section spacing
    sectionSm: spacing[8],      // 32px
    sectionMd: spacing[12],     // 48px
    sectionLg: spacing[16],     // 64px
} as const;

// ============================================
// BORDER RADIUS
// ============================================

export const borderRadius = {
    none: '0',
    sm: '0.375rem',   // 6px
    md: '0.5rem',     // 8px
    lg: '0.75rem',    // 12px
    xl: '1rem',       // 16px
    '2xl': '1.5rem',  // 24px
    full: '9999px',   // Fully rounded
} as const;

// ============================================
// SHADOWS
// ============================================

export const shadows = {
    none: 'none',
    sm: '0 1px 2px 0 rgba(0, 0, 0, 0.05)',
    md: '0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -2px rgba(0, 0, 0, 0.1)',
    lg: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -4px rgba(0, 0, 0, 0.1)',
    xl: '0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 8px 10px -6px rgba(0, 0, 0, 0.1)',
    '2xl': '0 25px 50px -12px rgba(0, 0, 0, 0.25)',
    inner: 'inset 0 2px 4px 0 rgba(0, 0, 0, 0.05)',
} as const;

// ============================================
// Z-INDEX LAYERS
// ============================================

export const zIndex = {
    base: 0,
    dropdown: 10,
    sticky: 20,
    overlay: 30,
    modal: 40,
    popover: 50,
    tooltip: 60,
} as const;

// ============================================
// EXPORTS
// ============================================

export const layout = {
    spacing,
    semanticSpacing,
    borderRadius,
    shadows,
    zIndex,
} as const;

export default layout;
